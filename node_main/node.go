package main

import (
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"go_crontab/config"
	"go_crontab/cron"
	"go_crontab/etcd"
	"go_crontab/ghttp"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	wg       sync.WaitGroup
	con      *config.Config
	nodeHttp *ghttp.HttpServer
	reset    chan int
)

func main() {
	src := "../config_file/node_config.yml"

	con, err := config.ReadConfig(src)
	if err != nil {
		panic("读取配置文件出错：" + err.Error())
	}

	initConfig(con)
	go resetConf()

	log.Info("任务节点启动")
	fmt.Println("任务节点启动")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c

	cancelConf()

	wg.Wait()

	log.Info("任务节点退出")

}

func initConfig(con *config.Config) {
	fmt.Println("初始化配置")

	//init log
	initLog(con.Log.Path)

	//init etcd
	err := etcd.InitNodeEtcd(con.Etcd, con.Http, reset)
	if err != nil {
		panic("初始化etcd出错：" + err.Error())
	}

	//init cron
	cron.InitCron()

	wg.Add(1)

	nodeHttp = ghttp.NewNodeServer(con.Http)
	go func() {
		defer wg.Done()
		nodeHttp.Start()
	}()
}

func initLog(path string) {
	//日志文件
	file, _ := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	log.SetOutput(file)
	//设置最低loglevel
	log.SetLevel(log.TraceLevel)

	// 设置 rotatelogs
	logWriter, err := rotatelogs.New(
		//分割后文件名称
		path+"%Y%m%d.log",

		//生成软链，指向最新日志文件
		rotatelogs.WithLinkName(path),

		//最大保存时间
		rotatelogs.WithMaxAge(7*24*time.Hour),

		//日志切割间隔
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		panic(err)
	}

	writeMap := lfshook.WriterMap{
		log.InfoLevel:  logWriter,
		log.FatalLevel: logWriter,
		log.DebugLevel: logWriter,
		log.WarnLevel:  logWriter,
		log.ErrorLevel: logWriter,
		log.PanicLevel: logWriter,
	}

	lfHook := lfshook.NewHook(writeMap, &log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	//新增Hook
	log.AddHook(lfHook)
}

func cancelConf() {
	etcd.CancelNodeEtcd()
	nodeHttp.Close()
	cron.StopCron()
}

func resetConf() {
	for {
		select {
		case <-reset:
			cancelConf()
			initConfig(con)
		}
	}
}
