package ghttp

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"go_crontab/etcd"
	"go_crontab/model"
	"go_crontab/mysql_dao"
	"go_crontab/notify"
	"net/http"
	"strconv"
	"strings"
)

type schedulerHandler struct {
	server *HttpServer
}

func newSchedulerHandler(s *HttpServer) *schedulerHandler {
	return &schedulerHandler{
		server: s,
	}
}

func (s1 *schedulerHandler) login(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("用户登录")

	var inputUser model.User
	err := c.ShouldBind(&inputUser)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	u := &mysql_dao.UserDao{}

	user, err := u.CheckUser(&inputUser)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("user_id", user.Id, 3600*24, "/", s1.server.host, false, false)
	c.SetCookie("user_name", user.Name, 3600*24, "/", s1.server.host, false, false)

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) register(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("用户注册")

	var user model.User
	err := c.ShouldBind(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	u := &mysql_dao.UserDao{}

	err = u.InsertUser(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) readUserInfo(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("查看个人信息")

	user_id, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	//获取数据库数据
	u := &mysql_dao.UserDao{}
	user, err := u.GetUserInfo(user_id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"error": "",
		"user":  user,
	})
}

func (s1 *schedulerHandler) modifyPassword(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("修改用户密码")

	id := c.PostForm("id")
	newPwd := c.PostForm("newPassword")
	oldPwd := c.PostForm("oldPassword")

	u := &mysql_dao.UserDao{}

	err := u.ModifyUserPassword(id, newPwd, oldPwd)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) modifyInfo(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("修改用户信息")

	var user model.User
	err := c.ShouldBind(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	u := &mysql_dao.UserDao{}
	err = u.ModifyUserInfo(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.SetCookie("user_name", user.Name, 3600*24, "/", s1.server.host, false, false)

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) exit(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("用户注销")

	c.SetCookie("user_id", "", -1, "/", s1.server.host, false, false)
	c.SetCookie("user_name", "", -1, "/", s1.server.host, false, false)
	c.JSON(http.StatusOK, nil)
}

func (s1 *schedulerHandler) addNode(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("新增任务节点")

	var hostList model.HostList
	err := c.ShouldBind(&hostList)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	h := &mysql_dao.HostDao{}

	err = h.InsertHostTable(&hostList)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) readNodeList(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("查看任务节点列表")

	user_id, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	//获取数据库数据
	h := &mysql_dao.HostDao{}
	hostList, err := h.GetHostList(user_id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	setcd := &etcd.SchedulerEtcd{}
	//获取etcd数据
	hostList1, _ := setcd.GetOnlineNodeList()

	for _, h1 := range hostList1 {
		for v, h := range hostList {
			len := len(h.Host)

			if h1[:len] == h.Host {
				hostList[v].Status = "在线"
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"error":    "",
		"hostList": hostList,
	})
}

func (s1 *schedulerHandler) readSchedulerList(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("查看调度器列表")

	//获取etcd数据
	setcd := &etcd.SchedulerEtcd{}

	list, err := setcd.GetOnlineSchedulerList()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	//获取leader
	leader, err := setcd.GetSchedulerLeaderAddr()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	var schedulerList []model.HostList

	for _, l := range list {
		var temp model.HostList
		if l == leader {
			temp.Status = "leader"
		} else {
			temp.Status = "follow"
		}
		str := strings.Split(l, ":")
		temp.Host = str[0]
		temp.Port = str[1]
		schedulerList = append(schedulerList, temp)
	}

	c.JSON(http.StatusOK, gin.H{
		"error":    "",
		"hostList": schedulerList,
	})
}

func (s1 *schedulerHandler) readCommandList(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("查看共享命令列表")

	user_id, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	//获取etcd数据
	setcd := &etcd.SchedulerEtcd{}

	commandList, err := setcd.GetSystemShareCommand(user_id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error":       "",
		"commandList": commandList,
	})
}

func (s1 *schedulerHandler) deleteNode(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("删除任务节点")

	var user_id = c.PostForm("user_id")
	var host = c.PostForm("host")

	h := &mysql_dao.HostDao{}

	err := h.DeleteHostList(user_id, host)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) updateNode(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("更新任务节点")

	var hostList model.HostList
	err := c.ShouldBind(&hostList)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"res":   0,
			"error": err.Error(),
		})
		return
	}

	h := &mysql_dao.HostDao{}

	err = h.UpdateHostTable(&hostList)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) acceptTask(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	var task model.Task

	err := c.ShouldBind(&task)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "读取请求数据出错:" + err.Error()})
		return
	}

	if task.TimeFormat == "" || task.Command == "" || task.Host == "" || task.UserId == "" {
		logger.WithField("task", task.ToString()).Error("任务数据缺失")
		c.JSON(http.StatusOK, gin.H{"error": "任务数据缺失"})
		return
	}

	//随机生成任务id
	task.Id = uuid.New().String()
	logger.WithField("taskId", task.Id).Info("生成新任务id")
	logger.WithField("task", task.ToString()).Info("接收到新任务请求")

	taskList := &model.TaskList{
		Tasks: []*model.Task{&task},
	}

	//下发任务
	n := notify.Notify{}
	err = n.SendTaskToNode("acceptTask", task.Host, &taskList)
	if err != nil {
		logger.WithField("task", task.ToString()).Error("调度器下发任务出错：" + err.Error())

		c.JSON(http.StatusOK, gin.H{
			"error": "调度器下发任务出错：" + err.Error(),
		})
		return
	}

	logger.Info("调度器下发任务成功")

	//任务数据入库
	tdao := mysql_dao.TaskDao{}
	err = tdao.InsertTask(&task)
	if err != nil {
		logger.WithField("task", task.ToString()).Error("任务数据入库出错：" + err.Error())
		logger.WithField("task", task.ToString()).Warning("任务数据入库出错，调度器删去下发的任务")

		err1 := n.SendTaskToNode("deleteTask", task.Host, &task)
		if err1 != nil {
			logger.WithField("task", task.ToString()).Error("调度器删去下发的任务出错：" + err.Error())

			c.JSON(http.StatusOK, gin.H{
				"error":  "任务数据入库出错：" + err.Error(),
				"error1": "调度器删去下发的任务出错，请手动重置或重启任务节点：" + err1.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"error": "任务数据入库出错：" + err.Error(),
		})
		return
	}

	logger.Info("任务数据入库成功")

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) updateTask(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	var task model.Task
	err := c.ShouldBind(&task)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "读取请求数据出错:" + err.Error()})
		return
	}

	if task.TimeFormat == "" || task.Command == "" || task.Host == "" || task.UserId == "" {
		logger.WithField("task", task.ToString()).Error("任务数据缺失")
		c.JSON(http.StatusOK, gin.H{"error": "任务数据缺失"})
		return
	}

	logger.WithField("task", task.ToString()).Info("接收到更新任务请求")

	//校验
	tdao := &mysql_dao.TaskDao{}
	check, err := tdao.CheckTask(&task)
	if err != nil {
		logger.WithField("task", task.ToString()).Error("校验任务数据出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "校验任务数据出错：" + err.Error()})
		return
	}

	if !check {
		logger.WithField("task", task.ToString()).Error("该任务不属于此用户")
		c.JSON(http.StatusOK, gin.H{"error": "该任务不属于此用户"})
		return
	}

	//更新任务
	n := &notify.Notify{}
	err = n.SendTaskToNode("updateTask", task.Host, &task)
	if err != nil {
		logger.Error("调度器下发更新任务出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "调度器下发更新任务出错：" + err.Error()})
		return
	}

	logger.Info("调度器更新任务成功")

	//更新任务数据
	oldTask, err := tdao.UpdateTask(&task)
	if err != nil {
		logger.WithField("task", task.ToString()).Error("更新任务数据出错：" + err.Error())
		logger.WithField("task", task.ToString()).Warning("更新任务数据出错，调度器重置下发的任务")

		err1 := n.SendTaskToNode("deleteTask", task.Host, &task)
		if err1 != nil {
			logger.WithField("task", task.ToString()).Error("调度器删去下发的任务出错：" + err.Error())

			c.JSON(http.StatusOK, gin.H{
				"error":  "任务数据入库出错：" + err.Error(),
				"error1": "调度器重置任务时，删去更新任务出错，请手动重置或重启任务节点：" + err1.Error(),
			})
			return
		}

		oldTaskList := &model.TaskList{
			Tasks: []*model.Task{oldTask},
		}

		err1 = n.SendTaskToNode("acceptTask", task.Host, &oldTaskList)
		if err1 != nil {
			logger.WithField("task", task.ToString()).Error("调度器下发原任务出错：" + err.Error())

			c.JSON(http.StatusOK, gin.H{
				"error":  "任务数据入库出错：" + err.Error(),
				"error1": "调度器重置任务时，已删去更新任务，但下发原任务出错，请手动重置或重启任务节点：" + err1.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"error": "更新任务数据出错：" + err.Error()})
		return
	}

	logger.Info("更新任务数据成功")

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) updateTaskData(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	var task model.Task
	err := c.ShouldBind(&task)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "读取请求数据出错:" + err.Error()})
		return
	}

	logger.WithField("task", task.ToString()).Info("接收到更新任务数据请求")

	//直接更新数据库时，要让这个任务关闭，因为节点并没有这个任务
	task.Off = true

	tdao := &mysql_dao.TaskDao{}

	err = tdao.UpdateTaskData(&task)
	if err != nil {
		logger.Error("更新任务数据出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "更新任务数据出错：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"error": ""})
}

func (s1 *schedulerHandler) deleteTask(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	var task model.Task

	err := c.ShouldBind(&task)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "读取请求数据出错:" + err.Error()})
		return
	}

	if task.Id == "" {
		logger.WithField("task", task.ToString()).Error("任务数据缺失")
		c.JSON(http.StatusOK, gin.H{"error": "任务数据缺失"})
		return
	}

	logger.WithField("task", task.ToString()).Info("接收到删除任务请求")

	//校验
	tdao := &mysql_dao.TaskDao{}
	check, err := tdao.CheckTask(&task)
	if err != nil {
		logger.WithField("task", task.ToString()).Error("校验任务数据出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "校验任务数据出错：" + err.Error()})
		return
	}

	if !check {
		logger.WithField("task", task.ToString()).Error("该任务不属于此用户")
		c.JSON(http.StatusOK, gin.H{"error": "该任务不属于此用户"})
		return
	}

	//删除任务
	n := &notify.Notify{}
	err = n.SendTaskToNode("deleteTask", task.Host, &task)
	if err != nil {
		logger.Error("调度器删除任务出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "调度器删除任务出错：" + err.Error()})
		return
	}

	logger.Info("调度器删除任务成功")

	//删除任务成功，此时删除数据
	oldTask, err := tdao.DeleteTask(&task)
	if err != nil {
		logger.WithField("task", task.ToString()).Error("删除任务数据出错：" + err.Error())
		logger.WithField("task", task.ToString()).Warning("任务数据入库出错，调度器重新下发原任务")

		oldTaskList := &model.TaskList{
			Tasks: []*model.Task{oldTask},
		}

		err1 := n.SendTaskToNode("acceptTask", task.Host, &oldTaskList)
		if err1 != nil {
			logger.WithField("task", task.ToString()).Error("调度器重新下发原任务出错：" + err.Error())

			c.JSON(http.StatusOK, gin.H{
				"error":  "删除任务数据出错：" + err.Error(),
				"error1": "调度器重新下发原任务出错，请手动重置或重启任务节点：" + err1.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"error": "删除任务数据出错：" + err.Error()})
		return
	}

	logger.Info("删除任务数据成功")

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) openOrCloseTask(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	var task model.Task

	err := c.ShouldBind(&task)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "读取请求数据出错:" + err.Error()})
		return
	}

	if task.TimeFormat == "" || task.Command == "" || task.Host == "" || task.UserId == "" {
		logger.WithField("task", task.ToString()).Error("任务数据缺失")
		c.JSON(http.StatusOK, gin.H{"error": "任务数据缺失"})
		return
	}

	off := !task.Off

	if off {
		logger.WithField("task", task.ToString()).Info("接收到关闭任务请求")
	} else {
		logger.WithField("task", task.ToString()).Info("接收到开启任务请求")
	}

	taskList := &model.TaskList{
		Tasks: []*model.Task{&task},
	}

	//通知任务节点
	n := notify.Notify{}
	if off {
		err = n.SendTaskToNode("deleteTask", task.Host, &task)
		if err != nil {
			logger.WithField("task", task.ToString()).Error("调度器关闭任务出错：" + err.Error())

			c.JSON(http.StatusOK, gin.H{
				"error": "调度器关闭任务出错：" + err.Error(),
			})
			return
		}
		logger.Info("调度器关闭任务成功")
	} else {
		err = n.SendTaskToNode("acceptTask", task.Host, &taskList)
		if err != nil {
			logger.WithField("task", task.ToString()).Error("调度器开启任务出错：" + err.Error())

			c.JSON(http.StatusOK, gin.H{
				"error": "调度器开启任务出错：" + err.Error(),
			})
			return
		}
		logger.Info("调度器开启任务成功")
	}

	//更新任务数据入库，task.Off属性已改变
	task.Off = off
	tdao := mysql_dao.TaskDao{}
	err = tdao.UpdateTaskOff(task.Id, task.Off)
	if err != nil {
		logger.WithField("task", task.ToString()).Error("更新任务数据入库出错：" + err.Error())

		if off {
			logger.WithField("task", task.ToString()).Warning("任务数据入库出错，调度器开启刚关闭的任务")
			err1 := n.SendTaskToNode("acceptTask", task.Host, &task)
			if err1 != nil {
				logger.WithField("task", task.ToString()).Error("调度器开启刚关闭的任务出错：" + err.Error())

				c.JSON(http.StatusOK, gin.H{
					"error":  "更新任务数据入库出错：" + err.Error(),
					"error1": "调度器开启刚关闭的任务，请手动重置或重启任务节点：" + err1.Error(),
				})
				return
			}
		} else {
			logger.WithField("task", task.ToString()).Warning("任务数据入库出错，调度器关闭刚开启的任务")
			err1 := n.SendTaskToNode("deleteTask", task.Host, &task)
			if err1 != nil {
				logger.WithField("task", task.ToString()).Error("调度器关闭刚开启的任务出错：" + err.Error())

				c.JSON(http.StatusOK, gin.H{
					"error":  "更新任务数据入库出错：" + err.Error(),
					"error1": "调度器关闭刚开启的任务出错，请手动重置或重启任务节点：" + err1.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"error": "更新任务数据入库出错：" + err.Error(),
		})
		return
	}

	logger.Info("更新任务数据入库成功")

	c.JSON(http.StatusOK, gin.H{
		"error": "",
	})
}

func (s1 *schedulerHandler) deleteTaskData(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	taskId := c.PostForm("id")

	logger.WithField("id", taskId).Info("接收到删除任务数据请求")

	tdao := &mysql_dao.TaskDao{}

	err := tdao.DeleteTaskData(taskId)
	if err != nil {
		logger.Error("删除任务数据出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "删除任务数据出错：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"error": ""})
}

func (s1 *schedulerHandler) queryTask(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	var (
		name       = c.PostForm("name")
		host       = c.PostForm("host")
		timeout, _ = strconv.Atoi(c.PostForm("timeout"))
		system, _  = strconv.Atoi(c.PostForm("system"))
		once, _    = strconv.Atoi(c.PostForm("once"))
		alone, _   = strconv.Atoi(c.PostForm("alone"))
		share, _   = strconv.Atoi(c.PostForm("share"))
		repeat, _  = strconv.Atoi(c.PostForm("repeat"))
		off, _     = strconv.Atoi(c.PostForm("off"))
	)

	logger.Info("接收到查询任务请求")

	//查询任务
	tdao := &mysql_dao.TaskDao{}
	taskList, err := tdao.GetTasksByWords(name, host, timeout, system, once, alone, share, repeat, off)
	if err != nil {
		logger.Error("查询任务数据出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "查询任务数据出错：" + err.Error()})
		return
	}

	logger.Info("查询任务数据成功")

	c.JSON(http.StatusOK, gin.H{
		"error":    "",
		"taskList": taskList,
	})
}

func (s1 *schedulerHandler) readTaskList(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("查看任务列表")

	user_id, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}
	tdao := &mysql_dao.TaskDao{}

	taskList, err := tdao.GetTaskByUser(user_id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error":    "",
		"taskList": taskList,
	})
}

func (s1 *schedulerHandler) acceptStatus(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"local_host:": c.Request.Host,
		"remote_host": c.Request.RemoteAddr,
		"url":         c.Request.URL,
	})

	var statusInfo model.StatusInfo

	err := c.ShouldBind(&statusInfo)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": "读取请求数据出错:" + err.Error()})
		return
	}

	//任务执行数据入库
	sdao := mysql_dao.StatusDao{}
	err = sdao.InsertTaskStatus(&statusInfo)
	if err != nil {
		logger.WithField("status_info", statusInfo.ToString()).Error("任务执行数据入库出错：" + err.Error())

		c.JSON(http.StatusOK, gin.H{
			"res": 0,
		})
		return
	}

	logger.Info("任务执行数据入库成功")

	c.JSON(http.StatusOK, gin.H{
		"res": 1,
	})
}

func (s1 *schedulerHandler) readTaskStatus(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("查看任务执行状态")

	id := c.PostForm("task_id")

	sdao := &mysql_dao.StatusDao{}
	statusList, err := sdao.GetTaskStatus(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error":      "",
		"statusList": statusList,
	})
}

func (s1 *schedulerHandler) readTaskLog(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("查看任务日志")

	id := c.PostForm("task_id")
	host := c.PostForm("host")

	n := &notify.Notify{}
	temp, err := n.NotifyNodeSendLog(id, host)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
		return
	}

	temp = strings.Trim(temp, "\"")
	str := strings.Split(temp, "time")

	var logText []model.LogText
	for _, v := range str[1:] {
		lt := model.LogText{
			Text: "time" + v,
		}
		logText = append(logText, lt)
	}

	c.JSON(http.StatusOK, gin.H{
		"error":   "",
		"logText": logText,
	})
}

func (s1 *schedulerHandler) deleteUselessTaskLog(c *gin.Context) {
	log.WithField("remote_host", c.Request.RemoteAddr).Info("通知任务节点删除多余任务日志")

	host := c.PostForm("host")

	tdao := &mysql_dao.TaskDao{}
	taskList, err := tdao.GetTasksByHost(host)
	if err != nil {
		log.Error("通知任务节点删除多余任务日志出错：" + err.Error())
		c.JSON(http.StatusOK, gin.H{"error": err})
		return
	}

	logFile := &model.LogFile{}
	for _, task := range taskList.Tasks {
		temp := task.Id + ".log"
		logFile.FileName = append(logFile.FileName, temp)
	}

	n := &notify.Notify{}
	number, err := n.NotifyNodeDeleteLog(host, logFile)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error":  "",
		"number": number,
	})
}
