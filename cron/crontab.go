package cron

import (
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"go_crontab/etcd"
	"go_crontab/job"
	"go_crontab/model"
	"strings"
)

type Crontab struct {
}

var (
	taskMap     = make(map[string]string)       //记录任务id及其执行的命令
	lockMap     = make(map[string]cron.EntryID) //记录command和启动该定时任务id
	onceMap     = make(map[string]chan int)     //记录单次定时任务，用于删除任务时停用设定的任务
	defaultCron *cron.Cron
)

func InitCron() {
	defaultCron = cron.New(cron.WithSeconds())
	defaultCron.Start()
}

func StopCron() {
	defaultCron.Stop()
	//重置
	taskMap = make(map[string]string)
	lockMap = make(map[string]cron.EntryID)
	onceMap = make(map[string]chan int)
}

func (c *Crontab) AddTask(tasks []*model.Task) (string, error) {
	repeatTaskErr := ""
	number := 0

	for _, task := range tasks {

		//保证命令的唯一
		key := strings.Replace(task.Command, " ", "", -1)
		key = strings.Replace(key, "\n", "", -1)
		key = strings.Replace(key, "\t", "", -1)

		if !task.Share {
			key = task.UserId + key
		}

		//检查任务是否在map中，在的就放弃
		//一个任务丢了，它之后的可能不需要丢，所以继续
		if _, ok := lockMap[key]; ok {
			errStr := fmt.Sprintf("出现重复任务，丢弃，任务命令为：%s;", task.Command)
			repeatTaskErr = repeatTaskErr + errStr
			continue
		}

		//单机任务
		if task.Alone {
			n := &etcd.NodeEtcd{}
			//分布式锁
			value := task.Host + ":" + task.Command + ":" + task.UserId
			if !task.Share {
				value = value + ":" + "User Layer"
			} else {
				value = value + ":" + "System Layer"
			}

			ok, err := n.Lock(key, value)
			if err != nil {
				errStr := fmt.Sprintf("有单机任务出错，任务id为：%s，任务命令为：%s，error：%s;", task.Id, task.Command, err.Error())
				repeatTaskErr = repeatTaskErr + errStr
				continue
			}

			if !ok {
				errStr := fmt.Sprintf("该单机任务指令已在运行， 丢弃，任务id为：%s，任务命令为：%s，可查看共享任务指令表;", task.Id, task.Command)
				repeatTaskErr = repeatTaskErr + errStr
				continue
			}
		}

		nodeJob := &job.NodeJob{
			Task: task,
		}

		//只执行一次的定时任务
		if task.Once {
			del := make(chan int, 0)
			mapChan := make(chan int, 0)
			err := nodeJob.RunOnce(del, mapChan)
			if err != nil {
				log.WithField("taskId", task.Id).Error("设置定时任务出错：" + err.Error())
				errStr := fmt.Sprintf("单次定时任务出错，丢弃，任务id为：%s,任务时间为：%s，error=%s;", task.Id, task.TimeFormat, err.Error())
				repeatTaskErr = repeatTaskErr + errStr

				//丢了该任务，如果任务同时还是单机，要解锁
				if task.Alone {
					n := etcd.NodeEtcd{}
					if _, err := n.UnLock(key, task.Host); err != nil {
						log.WithField("taskId", task.Id).Error("解锁出错：" + err.Error())
					}
				}
				continue
			}

			onceMap[task.Id] = del
			taskMap[task.Id] = key
			number++
			go func() {
				<-mapChan
				delete(onceMap, task.Id)
			}()
			continue
		}

		log.Info(nodeJob.Task.ToString())

		//启动多次执行的定时任务
		entryId, err := defaultCron.AddJob(task.TimeFormat, nodeJob)
		if err != nil {
			log.WithField("task", task.ToString()).Error("设置多次定时任务出错：" + err.Error())

			//解锁
			if task.Alone {
				n := etcd.NodeEtcd{}
				if _, err := n.UnLock(key, task.Host); err != nil {
					log.WithField("taskId", task.Id).Error("解锁出错：" + err.Error())
				}
			}

			errStr := fmt.Sprintf("设置多次定时任务出错，丢弃，任务id为：%s，error=%s;", task.Id, err.Error())
			repeatTaskErr = repeatTaskErr + errStr

			continue
		}

		taskMap[task.Id] = key
		lockMap[key] = entryId
		number++
	}

	log.WithField("任务数量", number).Info("设置任务完成")
	return repeatTaskErr, nil
}

func (c *Crontab) DeleteTask(task *model.Task) error {
	if _, ok := taskMap[task.Id]; !ok {
		return errors.New("no such task")
	}

	key := taskMap[task.Id]

	if task.Alone {
		n := &etcd.NodeEtcd{}
		if _, err := n.UnLock(key, task.Host); err != nil {
			log.WithField("taskId", task.Id).Error("解锁出错：" + err.Error())
			return err
		}
	}

	if task.Once {
		del := onceMap[task.Id]
		del <- 1
		delete(taskMap, task.Id)
		delete(onceMap, task.Id)
	} else {
		entryId := lockMap[key]
		defaultCron.Remove(entryId)
		delete(taskMap, task.Id)
		delete(lockMap, key)
	}

	log.WithField("taskId", task.Id).Info("删除任务")
	return nil
}

func (c *Crontab) UpdateTask(task *model.Task) error {
	key, ok := taskMap[task.Id]
	if !ok {
		return errors.New("主机上没有这个任务,ID=" + task.Id)
	}

	//保证命令唯一
	newKey := strings.Replace(task.Command, " ", "", -1)
	newKey = strings.Replace(newKey, "\n", "", -1)
	newKey = strings.Replace(newKey, "\t", "", -1)

	if key != newKey {
		if _, ok := lockMap[newKey]; ok {
			return errors.New("更新任务失败，主机上已有该命令的任务")
		}
	}

	err := c.DeleteTask(task)
	if err != nil {
		return err
	}

	str, err := c.AddTask([]*model.Task{task})
	if err != nil {
		return err
	}
	if str != "" {
		return errors.New(str)
	}

	log.WithField("task", task.ToString()).Info("更新任务")
	return nil
}
