package ghttp

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"go_crontab/cron"
	"go_crontab/model"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

type nodeHandler struct {
	server *HttpServer
}

func newNodeHandler(s *HttpServer) *nodeHandler {
	return &nodeHandler{
		server: s,
	}
}

func (v1 *nodeHandler) acceptTask(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"local_host:": r.Host,
		"remote_host": r.RemoteAddr,
		"url":         r.URL,
	})

	defer r.Body.Close()
	result, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var taskList model.TaskList
	err = json.Unmarshal(result, &taskList)
	if err != nil {
		logger.Error("json转换出错:" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.WithField("task", taskList.ToString()).Info("接收到新任务请求")

	c := &cron.Crontab{}

	repeatTaskErr, err := c.AddTask(taskList.Tasks)
	if err != nil {
		logger.Error("接收任务出错：" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(repeatTaskErr) != 0 {
		w.Write([]byte("WARN: \n" + repeatTaskErr))
		logger.Warn(repeatTaskErr)
		return
	}

	logger.Info("接收任务结束")
	w.Write([]byte("success"))
}

func (v1 *nodeHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"local_host:": r.Host,
		"remote_host": r.RemoteAddr,
		"url":         r.URL,
	})

	defer r.Body.Close()
	result, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var task model.Task
	err = json.Unmarshal(result, &task)
	if err != nil {
		logger.Error("json转换出错:" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.WithField("task", task.ToString()).Info("接收到更新任务请求")

	c := &cron.Crontab{}

	err = c.UpdateTask(&task)
	if err != nil {
		logger.Error("更新任务出错：" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("接收更新任务结束")
	w.Write([]byte("success"))
}

func (v1 *nodeHandler) deleteTask(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"local_host:": r.Host,
		"remote_host": r.RemoteAddr,
		"url":         r.URL,
	})

	defer r.Body.Close()
	result, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("读取请求数据出错:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var task model.Task
	err = json.Unmarshal(result, &task)
	if err != nil {
		logger.Error("json转换出错:" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.WithField("task", task.ToString()).Info("接收到删除任务请求")

	c := &cron.Crontab{}

	err = c.DeleteTask(&task)
	if err != nil {
		logger.Error("删除任务出错：" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("接收删除任务结束")
	w.Write([]byte("success"))
}

func (v1 *nodeHandler) sendLog(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"local_host:": r.Host,
		"remote_host": r.RemoteAddr,
		"url":         r.URL,
	})

	query := r.URL.Query()
	taskId := query.Get("taskId")

	logger.WithField("taskId", taskId).Info("接收到发送任务日志请求")

	file, err := os.Open("../log/task_log/" + taskId + ".log")
	if err != nil {
		logger.WithField("taskId", taskId).Info("发送任务日志请求出错：" + err.Error())
		http.Error(w, "读取日志文件出错"+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	logText, err := ioutil.ReadAll(file)
	if err != nil {
		logger.WithField("taskId", taskId).Info("发送任务日志请求出错：" + err.Error())
		http.Error(w, "读取日志文件出错"+err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("发送任务日志结束")
	w.Write(logText)
}

func (v1 *nodeHandler) deleteUselessTaskLog(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"local_host:": r.Host,
		"remote_host": r.RemoteAddr,
		"url":         r.URL,
	})

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("删除多余任务日志出错：" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	//获取当前节点的任务数据
	var logFile model.LogFile
	err = json.Unmarshal(data, &logFile)
	if err != nil {
		log.Error("删除多余任务日志出错：" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("接收到删除多余任务日志请求")

	pathname := "../log/task_log/"

	//读取任务日志文件夹下的所有文件
	fileInfo, err := ioutil.ReadDir(pathname)
	if err != nil {
		log.Error("删除多余任务日志出错：" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//获取了当前该节点的现存所有任务的日志名集合
	number := 0
	for _, fi := range fileInfo {
		//如果是任务日志
		if !fi.IsDir() {
			fileName := fi.Name()
			eq := true

			for _, log := range logFile.FileName {
				//如果读到的文件在现存任务里能找到，跳过，不用删
				if fileName == log {
					eq = false
					break
				}
			}

			if eq {
				//读完集合还没找到对应任务，那么这个日志多余了，删了
				err := os.Remove(pathname + fileName)
				if err != nil {
					log.WithField("文件名", fileName).Error("删除日志文件出错：" + err.Error())
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				number++
			}
		}
	}

	logger.Info("删除多余任务日志结束")
	w.Write([]byte(strconv.Itoa(number)))
}
