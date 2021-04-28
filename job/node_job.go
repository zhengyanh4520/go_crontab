package job

import (
	"bytes"
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"go_crontab/etcd"
	"go_crontab/model"
	"go_crontab/notify"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

//节点任务
type NodeJob struct {
	Task *model.Task
}

var (
	nowMap = make(map[string]bool) //当前正在执行的任务表
	lock   sync.Mutex              //读写map时上锁
)

func GbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func (j *NodeJob) CreateTaskLog() *log.Logger {
	fileName := "../log/task_log/" + j.Task.Id + ".log"
	src, _ := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	taskLog := log.New()
	taskLog.Out = src
	//设置输出样式
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	taskLog.SetFormatter(customFormatter)
	//设置最低loglevel
	taskLog.SetLevel(log.TraceLevel)
	return taskLog
}

func (j *NodeJob) Run() {

	taskLog := j.CreateTaskLog()

	if !j.Task.Repeat {
		if _, ok := nowMap[j.Task.Id]; ok {
			taskLog.WithField("task", j.Task.ToString()).Error("该任务不可重复，上一次还在执行，本次丢弃")
			return
		} else {
			//记录本次任务，需要锁，否则多协程读写map会出错
			lock.Lock()
			nowMap[j.Task.Id] = true
			lock.Unlock()
		}
	}

	//超时时间
	long := j.Task.Timeout
	if j.Task.Timeout > 86400 || j.Task.Timeout <= 0 {
		long = 86400
	}

	timeOut := time.Duration(long) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()

	statusInfo := model.StatusInfo{
		TaskId: j.Task.Id,
		Host:   j.Task.Host,
	}
	statusInfo.Start = time.Now().Format("2006-01-02 15:04:05")

	if j.Task.System {
		j.RunOnWindows(ctx, &statusInfo, taskLog)
	} else {
		j.RunOnLinux(ctx, &statusInfo, taskLog)
	}

	if !j.Task.Repeat {
		lock.Lock()
		delete(nowMap, j.Task.Id)
		lock.Unlock()
	}

	statusInfo.End = time.Now().Format("2006-01-02 15:04:05")

	//执行完通知leader，leader保存执行结果
	nEtcd := etcd.NodeEtcd{}
	addr, err := nEtcd.GetSchedulerLeaderAddr()
	if err != nil {
		taskLog.Error("获取leader调度器出错：" + err.Error())
		return
	}

	if addr == "" {
		taskLog.Error("当前没有leader调度器")
		/*n := notify.Notify{}
		if err := n.NotifyRootNoLeader(j.Task); err != nil {
			log.Error("通知没有leader调度器出错：", zap.Error(err))
		}*/
		return
	}

	taskLog.WithField("status_info", statusInfo.ToString()).Info("通知调度器任务执行状态")
	n := notify.Notify{}
	err = n.NotifySchedulerTaskStatus(&statusInfo, addr)
	if err != nil {
		taskLog.Error("通知leader调度器任务执行状态出错：" + err.Error())
	}

	taskLog.Info("调度器接收任务状态成功")

	//执行完自己保存执行结果
	/*sdao := &mysql_dao.StatusDao{}
	err := sdao.InsertTaskStatus(j.Task, start, end)
	if err != nil {
		taskLog.Error("插入任务执行状态表出错：" + err.Error())
		return
	}

	if j.Task.Status {
		n := notify.Notify{}
		err = n.NotifySchedulerTaskError(j.Task)
		if err != nil {
			taskLog.Error("通知调度器任务执行失败时出错：" + err.Error())
			return
		}
		taskLog.WithField("taskId", j.Task.Id).Info("通知调度器任务执行失败")
	}*/

}

func (j *NodeJob) RunOnLinux(ctx context.Context, statusInfo *model.StatusInfo, taskLog *log.Logger) {
	cmd := exec.Command("/bin/bash", "-c", j.Task.Command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		//Setpgid: true,
	}

	result := make(chan int)
	go j.runCommand(cmd, result, statusInfo, taskLog)

	select {
	case <-ctx.Done():
		if cmd.Process.Pid > 0 {
			//syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		statusInfo.Status = "失败"
		statusInfo.Error = "运行时间超时，杀死此次任务"
	case <-result:
		taskLog.Info("任务运行结束")
	}
}

func (j *NodeJob) RunOnWindows(ctx context.Context, statusInfo *model.StatusInfo, taskLog *log.Logger) {
	cmd := exec.Command("cmd", "/C", j.Task.Command)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		//隐藏窗口，防止弹出
		HideWindow: true,
	}

	result := make(chan int)
	go j.runCommand(cmd, result, statusInfo, taskLog)

	select {
	case <-ctx.Done():
		if cmd.Process.Pid > 0 {
			exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
			cmd.Process.Kill()
		}
		statusInfo.Status = "失败"
		statusInfo.Error = "运行时间超时，杀死此次任务"
	case <-result:
		taskLog.Info("任务运行结束")
	}
}

func (j *NodeJob) runCommand(cmd *exec.Cmd, result chan int, statusInfo *model.StatusInfo, taskLog *log.Logger) {
	taskLog.Info("开始执行任务")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout // 标准输出
	cmd.Stderr = &stderr // 标准错误
	runErr := cmd.Run()

	var outStr, errStr []byte
	var err error
	if j.Task.System {
		outStr, err = GbkToUtf8(stdout.Bytes())
		if err != nil {
			taskLog.Error("标准任务输出转码失败，可能会出现乱码")
			outStr = stdout.Bytes()
		}
		errStr, err = GbkToUtf8(stderr.Bytes())
		if err != nil {
			taskLog.Error("标准错误输出转码失败，可能会出现乱码")
			outStr = stderr.Bytes()
		}
	} else {
		outStr = stdout.Bytes()
		errStr = stderr.Bytes()
	}

	if runErr != nil {
		taskLog.WithFields(log.Fields{
			"错误原因":     string(errStr),
			"任务指令执行输出": string(outStr),
		}).Error("执行任务出错，出现乱码时请自行检查指令")
		statusInfo.Status = "失败"
		statusInfo.Error = string(errStr)
		result <- 1
		return
	}

	taskLog.WithField("任务输出", string(outStr)).Info("任务执行成功")

	result <- 1
}

func (j *NodeJob) RunOnce(del chan int, mapChan chan int) error {
	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", j.Task.TimeFormat, time.Local)
	if err != nil {
		log.WithField("taskId", j.Task.Id).Error("任务时间格式化出错：" + err.Error())
		return err
	}

	now := time.Now()
	//转化为时间戳计算，多少秒后任务应该启动
	interval := startTime.Unix() - now.Unix()

	if interval < 0 {
		log.WithField("taskId", j.Task.Id).Error("任务时间失效")
		return errors.New("任务时间失效")
	}

	//启动协程执行定时任务
	go j.runTaskOnce(interval, del, mapChan)

	return nil
}

func (j *NodeJob) runTaskOnce(interval int64, del chan int, mapChan chan int) {
	//定时chan
	ch := time.After(time.Duration(interval) * time.Second)

	select {
	case <-ch:
		//时间到后通行,run函数实现过了，直接调用
		j.Run()
		//执行完成后，判断是否解锁
		if j.Task.Alone {
			//保证命令的唯一
			//保证命令的唯一
			key := strings.Replace(j.Task.Command, " ", "", -1)
			key = strings.Replace(key, "\n", "", -1)
			key = strings.Replace(key, "\t", "", -1)

			if !j.Task.Share {
				key = j.Task.UserId + key
			}

			n := &etcd.NodeEtcd{}
			if _, err := n.UnLock(key, j.Task.Host); err != nil {
				log.WithField("taskId", j.Task.Id).Error("解锁出错：" + err.Error())
			}
		}

	case <-del:
		log.WithField("task", j.Task.ToString()).Info("删除单次定时任务")
	}

	//通知删除oncemap
	mapChan <- 1
}
