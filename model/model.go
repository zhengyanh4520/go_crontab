package model

import (
	"encoding/json"
	"strings"
)

type Task struct {
	Id         string `form:"id" json:"id"`                   //任务主键
	Name       string `form:"name" json:"name"`               //任务名称
	UserId     string `form:"user_id" json:"user_id"`         //任务所属
	TimeFormat string `form:"time_format" json:"time_format"` //定时时间
	Command    string `form:"command" json:"command"`         //命令
	Host       string `form:"host" json:"host"`               //任务运行主机
	Timeout    int    `form:"timeout" json:"timeout"`         //超时时间
	Alone      bool   `form:"alone" json:"alone"`             //单机任务，相同指令不能发布在其他主机
	Once       bool   `form:"once" json:"once"`               //任务只运行一次，TimeFormat为一个时刻
	Repeat     bool   `form:"repeat" json:"repeat"`           //定时任务在主机上执行未完成，但到了第二次执行，评定是否允许任务的第二次执行
	System     bool   `form:"system" json:"system"`           //主机的系统，默认为false为Linux
	Share      bool   `form:"share" json:"share"`             //决定单机任务的指令在用户自己或系统中共享，默认为false，即自己
	Off        bool   `form:"off" json:"off"`                 //任务是否关闭，默认任务为开启状态，即任务主机启动时，任务即下发
}

type TaskList struct {
	Tasks []*Task `form:"tasks" json:"tasks"`
}

type StatusInfo struct {
	TaskId  string `form:"task_id" json:"task_id"`
	Name    string `form:"name" json:"name"`
	Host    string `form:"host" json:"host"`
	Start   string `form:"start" json:"start"`
	End     string `form:"end" json:"end"`
	Status  string `form:"status" json:"status"` //执行结果
	Error   string `form:"error" json:"error"`   //失败原因
	Numbers int    `form:"numbers" json:"numbers"`
}

type User struct {
	Id         string `form:"id" json:"id"`
	Password   string `form:"password" json:"password"`
	Name       string `form:"name" json:"name"`
	Company    string `form:"company" json:"company"`
	Department string `form:"department" json:"department"`
	Duties     string `form:"duties" json:"duties"`
	Phone      string `form:"phone" json:"phone"`
}

type HostList struct {
	UserId string `form:"user_id" json:"user_id" binding:"required"`
	Host   string `form:"host" json:"host" binding:"required"`
	Name   string `form:"name" json:"name" binding:"required"`
	Port   string `form:"port" json:"port" binding:"required"`
	Status string `form:"status" json:"status"`
}

type CommandList struct {
	User    string `form:"user" json:"user"`
	Command string `form:"command" json:"command"`
	Host    string `form:"host" json:"host"`
	Status  string `form:"status" json:"status"`
}

type Response struct {
	Res   int    `json:"res"`
	Error string `json:"error"`
}

type LogText struct {
	Text string `json:"text"`
}

type LogFile struct {
	FileName []string `json:"fileName"`
}

func (t *Task) ToString() string {
	result, _ := json.Marshal(t)
	return strings.Replace(string(result), "\"", "", -1)
}

func (t *TaskList) ToString() string {
	result, _ := json.Marshal(t)
	return strings.Replace(string(result), "\"", "", -1)
}

func (s *StatusInfo) ToString() string {
	result, _ := json.Marshal(s)
	return strings.Replace(string(result), "\"", "", -1)
}

func (u *User) ToString() string {
	result, _ := json.Marshal(u)
	return strings.Replace(string(result), "\"", "", -1)
}
