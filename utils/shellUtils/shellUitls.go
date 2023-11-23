package shellUtils

import (
	"bytes"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"os"
	"os/exec"
)

func ExecuteCommandInWin(command string) (string, error) {
	cmd := exec.Command("cmd", "/C", command)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ExecuteCommand 执行给定的 Shell 命令，并返回输出和错误（如果有的话）。
func ExecuteCommand(command string) (string, error) {
	cmd := exec.Command("cmd", "/C", command)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("run command error: %v, ERROR: %s", err, stderr.String())
	}
	if stderr.String() != "" {
		return "", fmt.Errorf("exec command error: %s", stderr.String())
	}
	return out.String(), nil
}

// 执行shell命令
func Shell(cmd string) (res string, err error) {
	return ExecuteCommand(cmd)
}

type Charset string

const (
	UTF8    = Charset("UTF-8")
	GB18030 = Charset("GB18030")
)

func ConvertByte2String(byte []byte, charset Charset) string {

	var str string
	switch charset {
	case GB18030:
		decodeBytes, _ := simplifiedchinese.GB18030.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case UTF8:
		fallthrough
	default:
		str = string(byte)
	}
	return str
}

func Chmod(filePath string) error {
	// 获取文件的当前权限
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// 添加可执行权限
	newMode := fileInfo.Mode() | 0100

	// 更改文件权限
	err = os.Chmod(filePath, newMode)
	if err != nil {
		return err
	}

	return nil
}
