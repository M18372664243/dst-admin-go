package autoCheck

import (
	"dst-admin-go/config/database"
	"dst-admin-go/constant/consts"
	"dst-admin-go/model"
	"dst-admin-go/service"
	"dst-admin-go/utils/clusterUtils"
	"dst-admin-go/utils/dstConfigUtils"
	"dst-admin-go/utils/dstUtils"
	"dst-admin-go/utils/fileUtils"
	"dst-admin-go/utils/levelConfigUtils"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	steamAPIKey = "73DF9F781D195DFD3D19DED1CB72EEE6"
	appID       = 322330
	language    = 6
)

var Manager *AutoCheckManager

type AutoCheckManager struct {
	AutoChecks  []model.AutoCheck
	statusMap   map[string]chan int
	launchMutex sync.Mutex
	mutex       sync.Mutex
}

var gameConsoleService service.GameConsoleService
var gameService service.GameService
var logRecordService service.LogRecordService

func (m *AutoCheckManager) ReStart(clusterName string) {
	for s := range m.statusMap {
		close(m.statusMap[s])  // 关闭通道
		delete(m.statusMap, s) // 从 statusMap 中删除键值对
	}

	// 清空所有表
	db := database.DB
	db.Where("1 = 1").Delete(&model.AutoCheck{})

	// TODO 添加表数据
	var autoChecks []model.AutoCheck
	config, _ := levelConfigUtils.GetLevelConfig(dstConfigUtils.GetDstConfig().Cluster)
	for i := range config.LevelList {
		level := config.LevelList[i]
		autoChecks = append(autoChecks, model.AutoCheck{
			ClusterName:  clusterName,
			LevelName:    level.Name,
			Uuid:         level.File,
			Enable:       0,
			Announcement: "",
			Times:        1,
			Sleep:        5,
			Interval:     10,
			CheckType:    consts.LEVEL_DOWN,
		})
		autoChecks = append(autoChecks, model.AutoCheck{
			ClusterName:  clusterName,
			LevelName:    level.Name,
			Uuid:         level.File,
			Enable:       0,
			Announcement: "",
			Times:        1,
			Sleep:        5,
			Interval:     10,
			CheckType:    consts.LEVEL_MOD,
		})
	}

	autoChecks = append(autoChecks, model.AutoCheck{
		ClusterName:  clusterName,
		LevelName:    clusterName,
		Uuid:         clusterName,
		Enable:       0,
		Announcement: "",
		Times:        1,
		Sleep:        5,
		Interval:     10,
		CheckType:    consts.UPDATE_GAME,
	})
	db.Save(&autoChecks)

	m.Start()
}

func (m *AutoCheckManager) Start() {
	// TODO 这里是防止1.2.5 版本残留的问题
	db2 := database.DB

	db2.Where("uuid is null or uuid = '' ").Delete(&model.AutoCheck{})

	config, _ := levelConfigUtils.GetLevelConfig(dstConfigUtils.GetDstConfig().Cluster)
	var uuidSet []string
	for i := range config.LevelList {
		level := config.LevelList[i]
		uuidSet = append(uuidSet, level.File)
	}
	var autoChecks []model.AutoCheck

	db := database.DB
	db.Where("uuid in ?", uuidSet).Find(&autoChecks)

	var autoCheck2 = model.AutoCheck{}
	db.Where("check_type = ?", consts.UPDATE_GAME).Find(&autoCheck2)
	autoChecks = append(autoChecks, autoCheck2)

	log.Println("autoChecks", autoChecks)
	m.AutoChecks = autoChecks
	m.statusMap = make(map[string]chan int)

	for i := range autoChecks {
		taskId := autoChecks[i].Uuid
		m.statusMap[taskId] = make(chan int)
	}

	for i := range autoChecks {
		go func(index int) {
			defer func() {
				if r := recover(); r != nil {
					log.Println(r)
				}
			}()
			taskId := autoChecks[index].Uuid
			if autoChecks[index].Uuid == "" {
				taskId = autoChecks[index].ClusterName
			}
			m.run(autoChecks[index], m.statusMap[taskId])
		}(i)
	}

}

func (m *AutoCheckManager) run(task model.AutoCheck, stop chan int) {
	for {
		select {
		case <-stop:
			return
		default:
			m.check(task)
		}
	}
}
func (m *AutoCheckManager) GetAutoCheck(clusterName, levelName, checkType, uuid string) *model.AutoCheck {
	db := database.DB
	autoCheck := model.AutoCheck{}
	db.Where("cluster_name = ? and level_name = ? and check_type = ? and uuid = ?", clusterName, levelName, checkType, uuid).Find(&autoCheck)
	if autoCheck.Interval == 0 {
		autoCheck.Interval = 10
	}
	return &autoCheck
}

// TODO 这里要修改
func (m *AutoCheckManager) check(task model.AutoCheck) {

	if task.Uuid != "" {
		task = *m.GetAutoCheck(task.ClusterName, task.LevelName, task.CheckType, task.Uuid)
	}

	// log.Println("开始检查", task.ClusterName, task.LevelName, task.CheckType, task.Enable)
	if task.Enable != 1 {
		time.Sleep(10 * time.Second)
	} else {
		checkInterval := time.Duration(task.Interval) * time.Minute
		strategy := StrategyMap[task.CheckType]
		if !strategy.Check(task.ClusterName, task.Uuid) {
			log.Println(task.ClusterName, task.Uuid, task.CheckType, " is not running, waiting for ", checkInterval)
			time.Sleep(checkInterval)
			if !strategy.Check(task.ClusterName, task.Uuid) {
				log.Println(task.ClusterName, task.Uuid, task.CheckType, "has not started, starting it...")
				err := strategy.Run(task.ClusterName, task.Uuid)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		log.Println("check true ", task.ClusterName, task.LevelName, task.CheckType)
		time.Sleep(checkInterval)
	}
}

func (m *AutoCheckManager) AddAutoCheckTasks(task model.AutoCheck) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	db := database.DB
	db.Save(&task)

	taskId := task.Uuid
	if oldChan, ok := m.statusMap[taskId]; ok {
		close(oldChan) // 关闭旧的通道
	}
	m.statusMap[taskId] = make(chan int)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
		m.run(task, m.statusMap[taskId])
	}()

}

func (m *AutoCheckManager) DeleteAutoCheck(taskId string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var autoCheck model.AutoCheck
	db := database.DB
	db.Delete(&autoCheck).Where("uuid = ?", taskId)

	if ch, ok := m.statusMap[taskId]; ok {
		close(ch)                   // 关闭通道
		delete(m.statusMap, taskId) // 从 statusMap 中删除键值对
	}

}

var StrategyMap = map[string]CheckStrategy{}

func init() {
	StrategyMap[consts.UPDATE_GAME] = &GameUpdateCheck{}
	StrategyMap[consts.LEVEL_MOD] = &LevelModCheck{}
	StrategyMap[consts.LEVEL_DOWN] = &LevelDownCheck{}
}

type CheckStrategy interface {
	Check(string, string) bool
	Run(string, string) error
}

type LevelModCheck struct{}

func (s *LevelModCheck) Check(clusterName, levelName string) bool {
	// 找到当前存档的modId, 然后根据判断当前存档的
	cluster := clusterUtils.GetCluster(clusterName)
	modoverridesPath := dstUtils.GetLevelModoverridesPath(clusterName, levelName)
	content, err := fileUtils.ReadFile(modoverridesPath)
	if err != nil {
		return true
	}
	workshopIds := dstUtils.WorkshopIds(content)
	if len(workshopIds) == 0 {
		return true
	}

	acfPath := filepath.Join(cluster.ForceInstallDir, "ugc_mods", cluster.ClusterName, levelName, "appworkshop_322330.acf")
	acfWorkshops := dstUtils.ParseACFFile(acfPath)

	log.Println("acf path: ", acfPath)
	// log.Println("acf workshops: ", acfWorkshops)

	activeModMap := make(map[string]dstUtils.WorkshopItem)
	for i := range workshopIds {
		key := workshopIds[i]
		value, ok := acfWorkshops[key]
		if ok {
			activeModMap[key] = value
		}
	}
	return diffFetchModInfo2(activeModMap)
}

// Run 更新会重启世界
func (s *LevelModCheck) Run(clusterName, levelName string) error {
	log.Println("正在更新模组 ", clusterName, levelName)
	SendAnnouncement2(clusterName, levelName)
	gameService.StopLevel(clusterName, levelName)
	cluster := clusterUtils.GetCluster(clusterName)
	bin := cluster.Bin
	beta := cluster.Beta
	gameService.LaunchLevel(clusterName, levelName, bin, beta)
	return nil
}

type LevelDownCheck struct{}

func (s *LevelDownCheck) Check(clusterName, levelName string) bool {
	logRecord := logRecordService.GetLastLog(clusterName, levelName)
	if logRecord.ID != 0 {
		if logRecord.Action == model.STOP {
			return true
		}
	}

	status := gameService.GetLevelStatus(clusterName, levelName)
	log.Println("世界状态", "clusterName", clusterName, "levelName", levelName, "status", status)
	return status
}

func (s *LevelDownCheck) Run(clusterName, levelName string) error {
	log.Println("正在启动世界 ", clusterName, levelName)
	// TODO 加锁
	if !gameService.GetLevelStatus(clusterName, levelName) {
		gameService.StopLevel(clusterName, levelName)
		cluster := clusterUtils.GetCluster(clusterName)
		bin := cluster.Bin
		beta := cluster.Beta
		gameService.LaunchLevel(clusterName, levelName, bin, beta)
	}
	return nil
}

type GameUpdateCheck struct{}

func (s *GameUpdateCheck) Check(clusterName, levelName string) bool {
	localDstVersion := gameService.GetLocalDstVersion(clusterName)
	lastDstVersion := gameService.GetLastDstVersion()
	log.Println("localDstVersion", localDstVersion, "lastDstVersion", lastDstVersion, lastDstVersion < localDstVersion)
	return lastDstVersion <= localDstVersion
}

func (s *GameUpdateCheck) Run(clusterName, levelName string) error {
	log.Println("正在更新游戏 ", clusterName, levelName)
	SendAnnouncement2(clusterName, levelName)

	return gameService.UpdateGame(clusterName)
}

func SendAnnouncement2(clusterName string, levelName string) {
	db := database.DB
	autoCheck := model.AutoCheck{}
	db.Where("uuid = ?", levelName).Find(&autoCheck)
	size := autoCheck.Times
	for i := 0; i < size; i++ {
		announcement := autoCheck.Announcement
		if announcement != "" {
			lines := strings.Split(announcement, "\n")
			for j := range lines {
				gameConsoleService.SentBroadcast2(clusterName, levelName, lines[j])
				time.Sleep(300 * time.Millisecond)
			}
		}
		time.Sleep(time.Duration(autoCheck.Sleep) * time.Second)
	}

}

func diffFetchModInfo2(activeModMap map[string]dstUtils.WorkshopItem) bool {

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	var modIds []string
	for key := range activeModMap {
		modIds = append(modIds, key)
	}

	urlStr := "http://api.steampowered.com/IPublishedFileService/GetDetails/v1/"
	data := url.Values{}
	data.Set("key", steamAPIKey)
	data.Set("language", "6")
	for i := range modIds {
		data.Set("publishedfileids["+strconv.Itoa(i)+"]", modIds[i])
	}
	urlStr = urlStr + "?" + data.Encode()

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return true
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return true
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return true
	}

	dataList, ok := result["response"].(map[string]interface{})["publishedfiledetails"].([]interface{})
	if !ok {
		return true
	}
	for i := range dataList {

		data2 := dataList[i].(map[string]interface{})
		_, find := data2["time_updated"]
		if find {
			timeUpdated := data2["time_updated"].(float64)
			modId := data2["publishedfileid"].(string)
			value, ok := activeModMap[modId]
			if ok {
				if timeUpdated > float64(value.TimeUpdated) {
					return false
				}
			}
		}

	}

	return true
}
