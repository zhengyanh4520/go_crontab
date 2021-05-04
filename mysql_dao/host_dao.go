package mysql_dao

import (
	log "github.com/sirupsen/logrus"
	"go_crontab/model"
)

type HostDao struct {
}

func (h *HostDao) GetHostList(user_id string) ([]model.HostList, error) {
	logger := log.WithFields(log.Fields{
		"user_id": user_id,
	})

	logger.Info("获取用户任务主机列表")

	sql := "select host,name,port from HostTable where user_id=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("获取用户任务主机列表出错：" + err.Error())
		return nil, err
	}

	rows, err := stmt.Query(user_id)
	if err != nil {
		logger.Error("获取用户任务主机列表出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	var hostList []model.HostList
	for rows.Next() {
		var host string
		var name string
		var port string
		err = rows.Scan(&host, &name, &port)
		if err != nil {
			logger.Error("获取用户任务主机列表出错：" + err.Error())
			return nil, err
		}

		temp := model.HostList{
			Host:   host,
			Name:   name,
			Port:   port,
			Status: "离线",
		}
		hostList = append(hostList, temp)
	}

	return hostList, nil
}

func (h *HostDao) InsertHostTable(u *model.HostList) error {
	logger := log.WithFields(log.Fields{
		"user_id": u.UserId,
		"host":    u.Host,
		"name":    u.Name,
		"port":    u.Port,
	})

	logger.Info("新增用户任务主机列表数据入库")

	sql := "insert into HostTable(name,user_id,host,port) values(?,?,?,?)"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("新增用户任务主机列表数据入库出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(u.Name, u.UserId, u.Host, u.Port)
	if err != nil {
		logger.Error("新增用户任务主机列表数据入库出错：" + err.Error())
		return err
	}

	return nil
}

func (h *HostDao) DeleteHostList(user_id string, host string) error {
	logger := log.WithFields(log.Fields{
		"user_id": user_id,
		"host":    host,
	})

	logger.Info("删除用户任务主机数据")

	sql := "delete from HostTable where user_id=? and host=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("删除用户任务主机数据出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(user_id, host)
	if err != nil {
		logger.Error("删除用户任务主机数据出错：" + err.Error())
		return err
	}

	return nil
}

func (h *HostDao) UpdateHostTable(u *model.HostList) error {
	logger := log.WithFields(log.Fields{
		"user_id": u.UserId,
		"host":    u.Host,
		"name":    u.Name,
		"port":    u.Port,
	})

	logger.Info("更新用户任务主机列表数据")

	sql := "update HostTable set name=?,port=? where user_id=? and host=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("更新用户任务主机列表数据出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(u.Name, u.Port, u.UserId, u.Host)
	if err != nil {
		logger.Error("更新用户任务主机列表数据出错：" + err.Error())
		return err
	}

	return nil
}
