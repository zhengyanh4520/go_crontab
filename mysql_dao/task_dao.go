package mysql_dao

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"go_crontab/model"
)

type TaskDao struct {
}

func (t *TaskDao) InsertTask(task *model.Task) error {
	logger := log.WithFields(log.Fields{
		"task": task.ToString(),
	})

	logger.Info("任务数据入库")

	sql := "insert into task_table(id,name,user_id,time_format,command,host,timeout,alone," +
		"once,repeats,run_system,share,off) values(?,?,?,?,?,?,?,?,?,?,?,?,?)"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("任务数据入库出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(task.Id, task.Name, task.UserId, task.TimeFormat, task.Command, task.Host, task.Timeout,
		task.Alone, task.Once, task.Repeat, task.System, task.Share, task.Off)
	if err != nil {
		log.Error("任务数据入库出错：" + err.Error())
		return err
	}

	return nil
}

func (t *TaskDao) DeleteTask(task *model.Task) (*model.Task, error) {
	logger := log.WithFields(log.Fields{
		"task_id": task.Id,
	})

	logger.Info("删除任务数据")

	sql := "delete from task_table where id=?"

	oldTask, err := t.GetTask(task.Id)
	if err != nil {
		logger.Error("获取原任务失败：" + err.Error())
		return nil, err
	}

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("删除任务数据出错：" + err.Error())
		return nil, err
	}

	res, err := stmt.Exec(task.Id)
	if err != nil {
		logger.Error("删除任务数据出错：" + err.Error())
		return nil, err
	}

	//没有该任务
	n, _ := res.RowsAffected()
	if n == 0 {
		err = errors.New("没有此任务")
		logger.Error("删除任务数据出错：" + err.Error())
		return nil, err
	}

	return oldTask, nil
}

func (t *TaskDao) DeleteTaskData(id string) error {
	logger := log.WithFields(log.Fields{
		"task_id": id,
	})

	logger.Info("删除任务数据")

	sql := "delete from task_table where id=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("删除任务数据出错：" + err.Error())
		return err
	}

	res, err := stmt.Exec(id)
	if err != nil {
		logger.Error("删除任务数据出错：" + err.Error())
		return err
	}

	//没有该任务
	n, _ := res.RowsAffected()
	if n == 0 {
		err = errors.New("没有此任务")
		logger.Error("删除任务数据出错：" + err.Error())
		return err
	}

	return nil
}

func (t *TaskDao) UpdateTaskData(task *model.Task) error {
	logger := log.WithFields(log.Fields{
		"task": task.ToString(),
	})

	logger.Info("更新任务数据")

	err := t.DeleteTaskData(task.Id)
	if err != nil {
		log.Error("更新任务数据出错：" + err.Error())
		return err
	}

	err = t.InsertTask(task)
	if err != nil {
		log.Error("更新任务数据出错：" + err.Error())
		return err
	}

	return nil
}

func (t *TaskDao) UpdateTaskOff(taskId string, off bool) error {
	logger := log.WithFields(log.Fields{
		"task_id": taskId,
		"off":     off,
	})

	logger.Info("修改任务开关属性")

	sql := "update task_table set off=? where id=?"
	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("修改任务开关属性出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(off, taskId)
	if err != nil {
		logger.Error("修改任务开关属性出错：" + err.Error())
		return err
	}

	return nil
}

func (t *TaskDao) UpdateTask(task *model.Task) (*model.Task, error) {
	log.WithField("task_id", task.Id).Info("更新任务数据")

	oldTask, err := t.DeleteTask(task)
	if err != nil {
		log.Error("更新任务数据出错：" + err.Error())
		return nil, err
	}

	err = t.InsertTask(task)
	if err != nil {
		log.Error("更新任务数据出错：" + err.Error())
		return nil, err
	}

	return oldTask, nil
}

func (t *TaskDao) CheckTask(task *model.Task) (bool, error) {
	logger := log.WithFields(log.Fields{
		"task_id": task.Id,
	})

	logger.Info("校验任务数据")

	sql := "select user_id from task_table where id=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("校验任务数据出错：" + err.Error())
		return false, err
	}

	rows, err := stmt.Query(task.Id)
	if err != nil {
		logger.Error("校验任务数据出错：" + err.Error())
		return false, err
	}
	defer rows.Close()

	user_id := ""
	for rows.Next() {
		err = rows.Scan(&user_id)
		if err != nil {
			logger.Error("校验任务数据出错：" + err.Error())
			return false, err
		}

		if user_id == task.UserId {
			return true, nil
		} else {
			return false, nil
		}
	}

	return false, errors.New("没有此任务")
}

func (t *TaskDao) GetTasksByHost(host string) (*model.TaskList, error) {
	logger := log.WithFields(log.Fields{
		"host": host,
	})

	logger.Info("查询节点任务数据")

	sql := "select id,name,user_id,time_format,command,host,timeout,alone,once,repeats,run_system,share,off " +
		"from task_table where host=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("查询节点任务数据出错：" + err.Error())
		return nil, err
	}

	rows, err := stmt.Query(host)
	if err != nil {
		logger.Error("查询节点任务数据出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	var taskList model.TaskList

	for rows.Next() {
		var task model.Task

		err := rows.Scan(&task.Id, &task.Name, &task.UserId, &task.TimeFormat, &task.Command, &task.Host, &task.Timeout,
			&task.Alone, &task.Once, &task.Repeat, &task.System, &task.Share, &task.Off)
		if err != nil {
			logger.WithField("sql", sql).Error("查询节点任务数据出错：" + err.Error())
			return nil, err
		}

		taskList.Tasks = append(taskList.Tasks, &task)
	}

	logger.WithField("数据数量", len(taskList.Tasks)).Info("查询节点任务数据")
	return &taskList, nil
}

func (t *TaskDao) GetTasksByWords(name, host string, timeout, system, once, alone, share, repeat, off int) (*model.TaskList, error) {
	logger := log.WithFields(log.Fields{
		"name":    name,
		"host":    host,
		"timeout": timeout,
		"system":  system,
		"once":    once,
		"alone":   alone,
		"share":   share,
		"repeat":  repeat,
		"off":     repeat,
	})

	logger.Info("查询节点任务数据")

	sql := "select id,name,user_id,time_format,command,host,timeout,alone,once,repeats,run_system,share,off " +
		"from task_table where if(?='',1,name=?) and if(?='',1,host=?) and timeout=? and if(?=2,1,run_system=?) " +
		"and if(?=2,1,once=?)  and if(?=2,1,alone=?) and if(?=2,1,share=?) and if(?=2,1,repeats=?) and if(?=2,1,off=?);"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("查询节点任务数据出错：" + err.Error())
		return nil, err
	}

	rows, err := stmt.Query(name, name, host, host, timeout, system, system,
		once, once, alone, alone, share, share, repeat, repeat, off, off)
	if err != nil {
		logger.Error("查询节点任务数据出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	var taskList model.TaskList

	for rows.Next() {
		var task model.Task

		err := rows.Scan(&task.Id, &task.Name, &task.UserId, &task.TimeFormat, &task.Command, &task.Host, &task.Timeout,
			&task.Alone, &task.Once, &task.Repeat, &task.System, &task.Share, &task.Off)
		if err != nil {
			logger.WithField("sql", sql).Error("查询节点任务数据出错：" + err.Error())
			return nil, err
		}

		taskList.Tasks = append(taskList.Tasks, &task)
	}

	logger.WithField("数据数量", len(taskList.Tasks)).Info("查询节点任务数据")
	return &taskList, nil
}

func (t *TaskDao) GetAllTasks() (*model.TaskList, error) {
	sql := "select id,name,user_id,time_format,command,host,timeout,alone,once,repeats,run_system,share,off " +
		"from task_table"

	log.WithField("sql", sql).Info("查询全部任务数据")

	client := getMysqlClient()
	rows, err := client.Query(sql)
	if err != nil {
		log.WithField("sql", sql).Error("查询全部任务数据出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	var taskList model.TaskList

	for rows.Next() {
		var task model.Task

		err := rows.Scan(&task.Id, &task.Name, &task.UserId, &task.TimeFormat, &task.Command, &task.Host, &task.Timeout,
			&task.Alone, &task.Once, &task.Repeat, &task.System, &task.Share, &task.Off)
		if err != nil {
			log.WithField("sql", sql).Error("查询全部任务数据出错：" + err.Error())
			return nil, err
		}

		taskList.Tasks = append(taskList.Tasks, &task)
	}

	log.WithField("数据数量", len(taskList.Tasks)).Info("查询全部任务数据")
	return &taskList, nil
}

func (t *TaskDao) GetTask(id string) (*model.Task, error) {
	logger := log.WithFields(log.Fields{
		"task_id": id,
	})

	logger.Info("获取任务数据")

	sql := "select id,name,user_id,time_format,command,host,timeout,alone,once,repeats,run_system,share,off " +
		"from task_table where id=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("获取任务数据出错：" + err.Error())
		return nil, err
	}

	rows, err := stmt.Query(id)
	if err != nil {
		logger.Error("获取任务数据出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var task model.Task

		err = rows.Scan(&task.Id, &task.Name, &task.UserId, &task.TimeFormat, &task.Command, &task.Host, &task.Timeout,
			&task.Alone, &task.Once, &task.Repeat, &task.System, &task.Share, &task.Off)
		if err != nil {
			logger.Error("获取任务数据出错：" + err.Error())
			return nil, err
		}

		return &task, nil
	}

	return nil, errors.New("没有此任务")
}

func (t *TaskDao) GetTaskByUser(user_id string) (*model.TaskList, error) {
	logger := log.WithFields(log.Fields{
		"user": user_id,
	})

	logger.Info("查询用户任务数据")

	sql := "select id,name,user_id,time_format,command,host,timeout,alone,once,repeats,run_system,share,off " +
		"from task_table " +
		"where user_id=?"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("查询用户任务数据出错：" + err.Error())
	}

	rows, err := stmt.Query(user_id)
	if err != nil {
		logger.Error("查询用户任务数据出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	var taskList model.TaskList

	for rows.Next() {
		var task model.Task

		err := rows.Scan(&task.Id, &task.Name, &task.UserId, &task.TimeFormat, &task.Command, &task.Host, &task.Timeout,
			&task.Alone, &task.Once, &task.Repeat, &task.System, &task.Share, &task.Off)
		if err != nil {
			logger.Error("查询用户任务数据出错：" + err.Error())
			return nil, err
		}

		taskList.Tasks = append(taskList.Tasks, &task)
	}

	logger.WithField("数据数量", len(taskList.Tasks)).Info("查询用户任务数据")
	return &taskList, nil
}
