package etcd

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	log "github.com/sirupsen/logrus"
	"go_crontab/config"
	"go_crontab/constants"
	"go_crontab/model"
	"go_crontab/notify"
	"strings"
	"time"
)

type SchedulerEtcd struct {
	Client  *clientv3.Client   //客户端
	Lease   clientv3.Lease     //租约
	LeaseId clientv3.LeaseID   //租约id
	Txn     clientv3.Txn       //事务，用于分布式锁
	Cancel  context.CancelFunc //用于关闭租约的函数
	Ttl     int64              //设置租约时间
	Host    string             //当前调度器主机，用于设置leader调度器的键值
	Addr    string             //当前调度器监听地址，用于设置leader调度器的键值
}

var (
	sEtcd  *SchedulerEtcd
	leader = false
)

func InitSchedulerEtcd(con *config.EtcdConfig, hcfg *config.HttpConfig, reset chan int) error {
	var conf = clientv3.Config{
		Endpoints:   con.Endpoints,
		DialTimeout: time.Duration(con.DialTimeout) * time.Second,
	}

	cli, err := clientv3.New(conf)
	if err != nil {
		log.Error("etcd初始化失败：", err.Error())
		return err
	}

	sEtcd = &SchedulerEtcd{
		Client: cli,
		Ttl:    con.Ttl,
		Host:   hcfg.Host,
		Addr:   hcfg.Host + ":" + hcfg.Port,
	}

	//租约
	sEtcd.Lease = clientv3.NewLease(sEtcd.Client)
	leaseResp, err := sEtcd.Lease.Grant(context.TODO(), sEtcd.Ttl)
	if err != nil {
		return err
	}
	sEtcd.LeaseId = leaseResp.ID

	var ctx context.Context
	ctx, sEtcd.Cancel = context.WithCancel(context.TODO())

	if err := sEtcd.keepAlive(ctx); err != nil {
		return err
	}

	//注册
	if err := sEtcd.register(); err != nil {
		return err
	}

	//监听
	ch := make(chan error, 0)
	go sEtcd.watchSchedulerLeader(ctx, ch, reset)
	go sEtcd.watchSchedulerList(ctx)
	go sEtcd.watchNodeList(ctx)

	//尝试抢锁，决定调度器的leader
	result, err := sEtcd.lock()
	if err != nil {
		log.Error("etcd锁出错：", err.Error())
		return err
	}

	if result {
		fmt.Println("当前调度器为leader，启动")
		leader = true
		return nil
	}

	fmt.Println("当前调度器为follower，只监听...")
	err = <-ch
	leader = true
	return err
}

func CancelSchedulerEtcd() {
	sEtcd.Cancel()
	sEtcd.Lease.Revoke(context.TODO(), sEtcd.LeaseId)
	log.Info("etcd服务关闭")
}

func (s *SchedulerEtcd) register() error {
	//在etcd上注册这台主机，租约失效时说明宕机
	_, err := s.Client.Put(context.TODO(), constants.SchedulerPrefix+s.Host, s.Addr, clientv3.WithLease(s.LeaseId))
	if err != nil {
		return err
	}
	return nil
}

func (s *SchedulerEtcd) keepAlive(ctx context.Context) error {
	//续租
	_, err := sEtcd.Lease.KeepAlive(ctx, sEtcd.LeaseId)
	if err != nil {
		return err
	}
	return nil
}

func (s *SchedulerEtcd) watchSchedulerLeader(ctx context.Context, ch chan error, reset chan int) {
	//监听leader分布式锁，用于竞争leader

	resp := s.Client.Watch(context.TODO(), constants.SchedulerLeader, clientv3.WithPrefix())
	for {
		select {
		case c := <-resp:
			for _, e := range c.Events {
				switch e.Type {
				case clientv3.EventTypePut:
					log.Info("出现新的leader调度器：", string(e.Kv.Value))
					if leader == true && string(e.Kv.Value) != s.Addr {
						log.Error("续租出错，leader身份被抢占，重置")
						fmt.Println("续租出错，leader身份被抢占，重置")
						//取消leader身份
						leader = false
						//通知main函数重置
						reset <- 1
					}

				case clientv3.EventTypeDelete:
					log.Error("当前调度器宕机，尝试竞争为leader调度器")

					if !leader {
						res, err := s.lock()
						if err != nil {
							log.Error("调度器锁分布式锁出错：", err.Error())
							ch <- err
						}
						if res {
							fmt.Println("当前调度器为leader，启动")
							ch <- nil
						}
					} else {
						log.Error("续租出错，leader身份丢失，重置")
						fmt.Println("续租出错，leader身份丢失，重置")

						//取消leader身份
						leader = false
						//通知main函数重置
						reset <- 1
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *SchedulerEtcd) watchSchedulerList(ctx context.Context) {
	//监听当前调度器列表，即当前分布式系统中的调度器总数

	resp := s.Client.Watch(context.TODO(), constants.SchedulerPrefix, clientv3.WithPrefix())
	for {
		select {
		case c := <-resp:
			for _, e := range c.Events {
				switch e.Type {
				case clientv3.EventTypePut:
					log.Info("系统加入了新的调度器：" + string(e.Kv.Value))
				case clientv3.EventTypeDelete:
					log.Error("系统退出了一调度器：" + string(e.Kv.Key))

					//如果退出的是它自己，但此时还在监听，说明是租约出了问题
					if string(e.Kv.Key)[len(constants.SchedulerPrefix):] == s.Host {
						//重新续租
						err := s.keepAlive(ctx)
						if err != nil {
							//出错即此异常无法解决，直接让调度器中止
							log.Error("调度器etcd续租出错：", err.Error())
							panic("调度器etcd续租出错：" + err.Error())
						}

						//重注册，失败说明此异常无法解决，等待重启
						if err := s.register(); err != nil {
							fmt.Println("调度器注册etcd出错：" + err.Error())
							panic("调度器注册etcd出错：" + err.Error())
						}

						log.Warn("调度器租约异常，已重新注册")
						continue
					}
				}

			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *SchedulerEtcd) watchNodeList(ctx context.Context) {
	//监听任务节点主机列表，任务主机上线时尝试下发任务

	ch := s.Client.Watch(context.TODO(), constants.NodePrefix, clientv3.WithPrefix())
	for {
		select {
		case c := <-ch:
			for _, e := range c.Events {
				switch e.Type {
				case clientv3.EventTypePut:
					host := string(e.Kv.Value)
					log.Info("出现新的任务节点：" + host)

					//先等待一段时间，此时任务节点可能还在初始化配置
					time.Sleep(10 * time.Second)

					//尝试下发该节点的任务
					n := notify.Notify{}
					err := n.NotifyNodeTask(host)
					if err != nil {
						log.Error("对新节点下发任务出错：" + err.Error())
					}

				case clientv3.EventTypeDelete:
					log.Error("一个任务节点退出：" + string(e.Kv.Key))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *SchedulerEtcd) lock() (bool, error) {
	//etcd分布式锁

	s.Txn = clientv3.NewKV(s.Client).Txn(context.TODO())

	s.Txn.If(clientv3.Compare(clientv3.CreateRevision(constants.SchedulerLeader), "=", 0)).
		Then(clientv3.OpPut(constants.SchedulerLeader, s.Addr, clientv3.WithLease(s.LeaseId))).
		Else()

	resp, err := s.Txn.Commit()
	if err != nil {
		return false, err
	}

	if !resp.Succeeded {
		return false, nil
	}

	return true, nil
}

func (s *SchedulerEtcd) NodeIsOnline(host string) (bool, error) {
	resp, err := sEtcd.Client.Get(context.TODO(), constants.NodePrefix+host)
	if err != nil {
		log.Error("获取etcd键值出错：" + err.Error())
		return false, err
	}

	//主机是否在线
	return len(resp.Kvs) == 1, nil
}

func (s *SchedulerEtcd) GetOnlineNodeList() ([]string, error) {
	//获取当前在线的任务主机
	resp, err := sEtcd.Client.Get(context.TODO(), constants.NodePrefix, clientv3.WithPrefix())
	if err != nil {
		log.Error("获取etcd键值出错：" + err.Error())
		return nil, err
	}

	nodeList := make([]string, 0)

	for _, v := range resp.Kvs {
		nodeList = append(nodeList, string(v.Value))
	}

	return nodeList, nil
}

func (s *SchedulerEtcd) GetOnlineSchedulerList() ([]string, error) {
	resp, err := sEtcd.Client.Get(context.TODO(), constants.SchedulerPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Error("获取etcd键值出错：" + err.Error())
		return nil, err
	}

	schedulerList := make([]string, 0)

	for _, v := range resp.Kvs {
		schedulerList = append(schedulerList, string(v.Value))
	}

	return schedulerList, nil
}

func (s *SchedulerEtcd) GetSchedulerLeaderAddr() (string, error) {
	//获取当前作为leader的调度器
	resp, err := sEtcd.Client.Get(context.TODO(), constants.SchedulerLeader)
	if err != nil {
		log.Error("获取etcd键值出错" + err.Error())
		return "", err
	}

	for _, v := range resp.Kvs {
		addr := string(v.Value)
		return addr, nil
	}

	//没有leader
	return "", nil
}

func (s *SchedulerEtcd) GetSystemShareCommand(user_id string) ([]model.CommandList, error) {
	//获取当前系统层面共享的指令
	resp, err := sEtcd.Client.Get(context.TODO(), constants.AlonePrefix, clientv3.WithPrefix())
	if err != nil {
		log.Error("获取etcd键值出错：" + err.Error())
		return nil, err
	}

	var commandList []model.CommandList

	for _, v := range resp.Kvs {
		str := string(v.Value)
		value := strings.Split(str, ":")
		temp := model.CommandList{
			Host:    value[0],
			Command: value[2],
			UserId:  value[3],
			Status:  value[4],
		}

		if temp.Status == "System Layer" {
			commandList = append(commandList, temp)
		} else if temp.UserId == user_id {
			commandList = append(commandList, temp)
		}
	}

	return commandList, nil
}
