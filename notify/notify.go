package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"go_crontab/constants"
	"go_crontab/model"
	"go_crontab/mysql_dao"
	"io/ioutil"
	"net/http"
	"time"
)

type Notify struct {
}

func (n *Notify) NotifyNodeTask(host string) error {
	//获取数据库中该节点的任务
	tdao := mysql_dao.TaskDao{}
	t, err := tdao.GetTasksByHost(host)
	if err != nil {
		log.Error("获取数据库数据出错：" + err.Error())
		return err
	}

	taskss := make([]*model.Task, 0)
	for _, task := range t.Tasks {
		//该任务是否开启
		if !task.Off {
			//要检测任务是否为单次任务
			if task.Once {
				startTime, err := time.ParseInLocation("2006-01-02 15:04:05", task.TimeFormat, time.Local)
				if err != nil {
					log.WithField("task", task).Error("该任务时间转换出错：" + err.Error())
					continue
				}

				//丢弃过期的任务
				if time.Now().Unix() > startTime.Unix() {
					continue
				}
			}

			taskss = append(taskss, task)
		}
	}

	if len(t.Tasks) == 0 {
		log.WithField("host", host).Info("该节点目前没有要执行的任务")
		return nil
	}

	err = n.SendTaskToNode("acceptTask", host, t)
	if err != nil {
		log.WithField("host", host).Error("任务下发出错：" + err.Error())
		return err
	}

	log.Info("新节点任务下发完成")
	return nil
}

func (n *Notify) SendTaskToNode(typ string, host string, task interface{}) error {
	contentType := "application/json"
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	jdata, err := json.Marshal(task)
	if err != nil {
		log.Error("任务数据转化为json时出错：", err.Error())
		return err
	}

	//生成url
	url := fmt.Sprintf(constants.TaskInterface, host, typ)

	resp, err := client.Post(url, contentType, bytes.NewBuffer(jdata))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	res, _ := ioutil.ReadAll(resp.Body)
	if string(res) != "success" {
		log.Error("节点接收任务出错：", string(res))
		return errors.New(string(res))
	}

	return nil
}

func (n *Notify) NotifySchedulerTaskStatus(statusInfo *model.StatusInfo, addr string) error {
	contentType := "application/json"
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	jdata, err := json.Marshal(statusInfo)
	if err != nil {
		return errors.New("任务数据转化为json时出错：" + err.Error())
	}

	//生成url
	url := fmt.Sprintf(constants.SchedulerAcceptStatus, addr)

	resp, err := client.Post(url, contentType, bytes.NewBuffer(jdata))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	var res model.Response
	err = json.Unmarshal(data, &res)
	if err != nil {
		return errors.New("json解析调度器返回数据出错,原数据为：" + string(data))
	}

	if res.Res != 1 {
		return errors.New("调度器接收任务状态出错：" + res.Error)
	}

	return nil
}

func (n *Notify) NotifyNodeSendLog(task_id string, host string) (string, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	//生成url
	url := fmt.Sprintf(constants.TaskInterface, host, "sendLog")
	url = url + fmt.Sprintf("?taskId=%s", task_id)

	resp, err := client.Get(url)
	if err != nil {
		log.Error("通知任务节点发送日志出错：" + err.Error())
		return "", err
	}

	defer resp.Body.Close()
	res, _ := ioutil.ReadAll(resp.Body)

	return string(res), nil
}

func (n *Notify) NotifyNodeDeleteLog(host string, logFile *model.LogFile) (string, error) {
	contentType := "application/json"
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	jdata, err := json.Marshal(logFile)
	if err != nil {
		log.Error("通知任务节点删除多余日志出错：", err.Error())
		return "", err
	}

	//生成url
	url := fmt.Sprintf(constants.TaskInterface, host, "deleteUselessTaskLog")

	resp, err := client.Post(url, contentType, bytes.NewBuffer(jdata))
	if err != nil {
		log.Error("通知任务节点删除多余日志出错：" + err.Error())
		return "", err
	}

	defer resp.Body.Close()
	res, _ := ioutil.ReadAll(resp.Body)

	return string(res), nil
}
