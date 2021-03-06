package main

import (
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"time"

	"go_crontab/config"
	"go_crontab/etcd"
	"go_crontab/ghttp"
	db "go_crontab/mysql_dao"
	"os"
	"os/signal"
	"sync"
)

var (
	wg            sync.WaitGroup
	con           *config.Config
	schedulerHttp *ghttp.HttpServer
	reset         = make(chan int, 0)
)

func main() {
	src := "../config_file/scheduler_config.yml"

	con, err := config.ReadConfig(src)
	if err != nil {
		panic("读取配置文件出错: " + err.Error())
	}

	initConfig(con)
	go resetConf(con)

	log.Info("调度器启动")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c

	cancelConf()

	wg.Wait()

	log.Info("调度器退出")

}

func initConfig(con *config.Config) {
	fmt.Println("初始化配置")

	//init log
	initLog(con.Log)

	//init mysql
	err := db.InitMysql(con.MySql)
	if err != nil {
		panic("初始化数据库出错：" + err.Error())
	}

	//init etcd
	err = etcd.InitSchedulerEtcd(con.Etcd, con.Http, reset)
	if err != nil {
		panic("初始化etcd出错" + err.Error())
	}

	wg.Add(1)

	schedulerHttp = ghttp.NewSchedulerServer(con.Http)
	go func() {
		defer wg.Done()
		_ = schedulerHttp.Start()
	}()
}

func initLog(lc *config.LogConfig) {
	//日志文件
	file, _ := os.OpenFile(lc.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	log.SetOutput(file)
	//设置最低loglevel
	log.SetLevel(log.TraceLevel)

	// 设置 rotatelogs
	logWriter, err := rotatelogs.New(
		//分割后文件名称
		lc.Path+"%Y%m%d.log",

		//生成软链，指向最新日志文件
		rotatelogs.WithLinkName(lc.Path),

		//最大保存时间
		rotatelogs.WithMaxAge(time.Duration(lc.Day)*24*time.Hour),

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
	etcd.CancelSchedulerEtcd()
	schedulerHttp.Close()
}

func resetConf(c *config.Config) {
	for {
		select {
		case <-reset:
			cancelConf()
			initConfig(c)
		}
	}
}
