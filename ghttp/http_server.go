package ghttp

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go_crontab/config"
	"net/http"
)

type HttpServer struct {
	host   string
	port   string
	server *http.Server
}

var (
	//定制路由
	acceptTask           = "/acceptTask"           // 调度/任务节点接收定时任务的路由
	updateTask           = "/updateTask"           // 更新
	deleteTask           = "/deleteTask"           // 删除
	sendLog              = "/sendLog"              // 任务节点发送某一任务的执行日志
	deleteUselessTaskLog = "/deleteUselessTaskLog" //任务节点删除多余任务日志
)

func (hs *HttpServer) Start() error {
	log.WithFields(log.Fields{
		"host": hs.host,
		"post": hs.port,
	}).Info("http服务启动")

	err := hs.server.ListenAndServe()
	if err != nil {
		log.Error("http服务启动出错:" + err.Error())
		return err
	}

	return nil
}

func (hs *HttpServer) Close() {
	log.Info("http服务退出")
	err := hs.server.Shutdown(context.TODO())
	if err != nil {
		log.Error("http服务退出出错：" + err.Error())
	}
}

func NewSchedulerServer(con *config.HttpConfig) *HttpServer {
	hs := &HttpServer{
		host: con.Host,
		port: con.Port,
	}

	r := gin.Default().Delims("{[", "]}")
	r.LoadHTMLGlob("../static/html/*")

	r.StaticFS("/static", http.Dir("../static"))

	r.GET("/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	handler := newSchedulerHandler(hs)

	r.POST("/login", handler.login)
	r.POST("/register", handler.register)
	r.POST("/modifyPassword", handler.modifyPassword)
	r.POST("/modifyName", handler.modifyName)
	r.GET("/exit", handler.exit)

	r.POST("/addNode", handler.addNode)
	r.POST("/readNodeList", handler.readNodeList)
	r.POST("/deleteNode", handler.deleteNode)
	r.POST("/updateNode", handler.updateNode)
	r.POST("/readSchedulerList", handler.readSchedulerList)
	r.POST("/readCommandList", handler.readCommandList)

	r.POST("/acceptTask", handler.acceptTask)
	r.POST("/updateTask", handler.updateTask)
	r.POST("/updateTaskData", handler.updateTaskData)
	r.POST("/deleteTask", handler.deleteTask)
	r.POST("/queryTask", handler.queryTask)
	r.POST("/deleteTaskData", handler.deleteTaskData)
	r.POST("/readTask", handler.readTaskList)
	r.POST("/acceptStatus", handler.acceptStatus)
	r.POST("/readTaskStatus", handler.readTaskStatus)
	r.POST("/readTaskLog", handler.readTaskLog)
	r.POST("/openOrCloseTask", handler.openOrCloseTask)
	r.POST("/deleteUselessTaskLog", handler.deleteUselessTaskLog)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", hs.host, hs.port),
		Handler: r,
	}

	hs.server = srv

	log.Info("http配置成功")
	return hs
}

func NewNodeServer(con *config.HttpConfig) *HttpServer {
	hs := &HttpServer{
		host: con.Host,
		port: con.Port,
	}

	handler := newNodeHandler(hs)

	mux := http.NewServeMux()
	mux.HandleFunc(acceptTask, handler.acceptTask)
	mux.HandleFunc(updateTask, handler.updateTask)
	mux.HandleFunc(deleteTask, handler.deleteTask)
	mux.HandleFunc(sendLog, handler.sendLog)
	mux.HandleFunc(deleteUselessTaskLog, handler.deleteUselessTaskLog)

	hs.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", hs.host, hs.port),
		Handler: mux,
	}

	log.Info("http配置成功")
	return hs
}
