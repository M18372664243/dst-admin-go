# dst-admin-go
> 饥荒联机版管理后台
>



**Now，Support Windows and Linux  platform**

**现已支持 windows 和 Linux 平台**

This is a management panel for Don't Starve Together, developed in Go. It offers simple deployment, low memory usage, an aesthetically pleasing interface, and user-friendly operations. The panel provides a visual interface for easily configuring game rooms and managing online mods. It also supports the management of multiple rooms. All of these features are designed to provide a smoother and more streamlined user experience.

使用go编写的饥荒管理面板,部署简单,占用内存少,界面美观,操作简单,提供可视化界面操作房间配置和模组在线配置,支持多房间管理，备份快照等功能

新增 **暗黑主题**，**国际化**，支持**多层世界**，支持更大屏幕显示

## 部署
注意目录必须要有读写权限。

点击查看 [部署文档](docs/install.md)

## 预览

在线预览地址 http://1.12.223.51:8082/
（admin 123456）
![首页效果](docs/image/登录.png)
![首页效果](docs/image/房间.png)
![首页效果](docs/image/mod.png)
![首页效果](docs/image/mod配置.png)
![统计效果](docs/image/统计.png)
![面板效果](docs/image/面板.png)
![日志效果](docs/image/日志.png)
    

## 运行

**修改config.yml**
```
#端口
port: 8082
database: dst-db
```


运行
```
go mod tidy
go run main.go
```

## 打包


### window 打包

window 下打包 Linux 二进制 

```
打开 cmd
set GOARCH=amd64
set GOOS=linux

go build
```

