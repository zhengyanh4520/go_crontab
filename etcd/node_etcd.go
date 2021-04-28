package etcd

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	log "github.com/sirupsen/logrus"
	"go_crontab/config"
	"go_crontab/constants"
	"strings"
	"time"
)

type NodeEtcd struct {
	Client  *clientv3.Client   //客户端
	Lease   clientv3.Lease     //租约
	LeaseId clientv3.LeaseID   //租约id
	Txn     clientv3.Txn       //事务，用于分布式锁
	Cancel  context.CancelFunc //用于关闭租约的函数
	Ttl     int64              //设置租约时间
	Host    string             //当前主机，用于设置键值
	Addr    string             //当前监听地址，用于设置键值
}

var (
	nEtcd *NodeEtcd
)

func InitNodeEtcd(con *config.EtcdConfig, hcon *config.HttpConfig, reset chan int) error {
	var conf = clientv3.Config{
		Endpoints:   con.Endpoints,
		DialTimeout: time.Duration(con.DialTimeout) * time.Second,
	}

	cli, err := clientv3.New(conf)
	if err != nil {
		log.Error("etcd初始化失败：", err.Error())
		return err
	}

	nEtcd = &NodeEtcd{
		Client: cli,
		Ttl:    con.Ttl,
		Host:   hcon.Host,
		Addr:   hcon.Host + ":" + hcon.Port,
	}

	//租约
	nEtcd.Lease = clientv3.NewLease(nEtcd.Client)
	leaseResp, err := nEtcd.Lease.Grant(context.TODO(), nEtcd.Ttl)
	if err != nil {
		return err
	}
	nEtcd.LeaseId = leaseResp.ID

	var ctx context.Context
	ctx, nEtcd.Cancel = context.WithCancel(context.TODO())
	_, err = nEtcd.Lease.KeepAlive(ctx, nEtcd.LeaseId)
	if err != nil {
		log.Error("lease error" + err.Error())
		return err
	}

	// 监听自己，失效重新注册
	go nEtcd.watchNodeList(ctx, reset)

	//在etcd上注册这台主机，租约失效时说明宕机
	_, err = nEtcd.Client.Put(context.TODO(), constants.NodePrefix+hcon.Host, nEtcd.Addr, clientv3.WithLease(nEtcd.LeaseId))
	if err != nil {
		fmt.Println("register error")
		return err
	}

	return nil
}

func CancelNodeEtcd() {
	nEtcd.Cancel()
	nEtcd.Lease.Revoke(context.TODO(), nEtcd.LeaseId)
}

func (n *NodeEtcd) watchNodeList(ctx context.Context, reset chan int) {
	resp := n.Client.Watch(context.TODO(), constants.NodePrefix, clientv3.WithPrefix())
	for {
		select {
		case c := <-resp:
			for _, e := range c.Events {
				switch e.Type {
				case clientv3.EventTypeDelete:
					host := string(e.Kv.Key)[len(constants.NodePrefix):]
					if host == n.Host {
						log.WithField("host", string(e.Kv.Key)).Error("节点续租c出错，重置")
						reset <- 1
					}
				}

			}
		case <-ctx.Done():
			return
		}
	}
}

func (n *NodeEtcd) GetSchedulerLeaderAddr() (string, error) {
	//获取当前作为leader的调度器

	resp, err := nEtcd.Client.Get(context.TODO(), constants.SchedulerLeader)
	if err != nil {
		return "", errors.New("获取etcd键值出错" + err.Error())
	}

	for _, v := range resp.Kvs {
		addr := string(v.Value)
		return addr, nil
	}

	//没有leader
	return "", nil
}

func (n *NodeEtcd) Lock(key string, value string) (bool, error) {
	//节点对单机任务加锁：尝试put键值

	nEtcd.Txn = clientv3.NewKV(nEtcd.Client).Txn(context.TODO())

	key = constants.AlonePrefix + key
	nEtcd.Txn.If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, value, clientv3.WithLease(nEtcd.LeaseId))).
		Else()

	resp, err := nEtcd.Txn.Commit()
	if err != nil {
		return false, err
	}

	if !resp.Succeeded {
		return false, nil
	}
	return true, nil
}

func (n *NodeEtcd) UnLock(key string, host string) (bool, error) {
	//对单机任务解锁,即删除自己的键值，当出错或任务不在此主机上，返回错误

	key = constants.AlonePrefix + key
	resp, err := nEtcd.Client.Get(context.Background(), key)
	if err != nil {
		log.Error("获取etcd键值出错" + err.Error())
		return false, err
	}

	if len(resp.Kvs) == 0 {
		return true, nil
	}

	value := string(resp.Kvs[0].Value)
	log.Info("尝试解锁，获取到etcd单机任务键值：" + value)

	nowHost := strings.Split(value, ":")[0]
	host = strings.Split(host, ":")[0]

	if nowHost != host {
		return false, errors.New("该任务不是此节点负责")
	}

	_, err = nEtcd.Client.Delete(context.Background(), key)
	if err != nil {
		log.Error("删除etcd键值出错：" + err.Error())
		return false, err
	}

	return true, nil
}
