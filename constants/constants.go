package constants

const (
	//调度器接收任务执行状态的接口
	SchedulerAcceptStatus = "http://%s/acceptStatus"

	//任务节点接收任务的接口
	TaskInterface = "http://%s/%s"

	//etcd各键值前缀
	SchedulerPrefix = "/SchedulerList/"
	NodePrefix      = "/AgentList/"
	AlonePrefix     = "/AloneCommand/"
	SchedulerLeader = "/SchedulerLeader"

	//调度器异常类型
	SchedulerError    = "调度器宕机，需要重启。IP：%s"
	SchedulerNoLeader = "当前没有leader调度器，请尽快启动。任务执行状态未被接收。"
)
