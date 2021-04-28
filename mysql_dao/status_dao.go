package mysql_dao

import (
	log "github.com/sirupsen/logrus"
	"go_crontab/model"
)

type StatusDao struct {
}

func (s *StatusDao) InsertTaskStatus(statusInfo *model.StatusInfo) error {
	logger := log.WithFields(log.Fields{
		"status_info": statusInfo.ToString(),
	})

	logger.Info("任务执行状态数据入库")

	sql := "insert into status_table(host,task_id, start_time,end_time,status,error)" +
		" values(?,?,?,?,?,?)"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("任务执行状态数据入库出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(statusInfo.Host, statusInfo.TaskId, statusInfo.Start, statusInfo.End, statusInfo.Status, statusInfo.Error)
	if err != nil {
		logger.Error("任务执行状态数据入库出错：" + err.Error())
		return err
	}

	return nil
}

func (s *StatusDao) GetTaskStatus(id string) ([]*model.StatusInfo, error) {
	logger := log.WithFields(log.Fields{
		"task_id": id,
	})

	//只取最近一天内
	/*sql := "select task_id,host,start,end,status,error " +
	"from status_table " +
	"where to_days(now()) - to_days(insert_time) <=1"*/

	sql := "select task_id,t.name,s.host,start_time,end_time,status,error " +
		"from status_table s,task_table t " +
		"where t.id=? and t.id=s.task_id " +
		"order by start_time desc;"

	logger.WithField("sql", sql).Info("查询任务执行状态表")

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.WithField("sql", sql).Error("查询任务执行状态表出错：" + err.Error())
		return nil, err
	}

	rows, err := stmt.Query(id)
	if err != nil {
		logger.WithField("sql", sql).Error("查询任务执行状态表出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	var statusList []*model.StatusInfo
	for rows.Next() {
		var (
			taskId string
			name   string
			host   string
			start  string
			end    string
			status string
			error  string
		)

		err := rows.Scan(&taskId, &name, &host, &start, &end, &status, &error)
		if err != nil {
			logger.WithField("sql", sql).Error("查询任务执行状态表出错：" + err.Error())
			return nil, err
		}

		if status == "" {
			status = "成功"
		}

		temp := &model.StatusInfo{
			TaskId: taskId,
			Name:   name,
			Host:   host,
			Start:  start,
			End:    end,
			Status: status,
			Error:  error,
		}

		statusList = append(statusList, temp)
	}

	return statusList, nil
}
