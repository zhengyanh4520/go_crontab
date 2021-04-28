package mysql_dao

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"go_crontab/model"
)

type UserDao struct {
}

func (u *UserDao) CheckUser(checkUser *model.User) (*model.User, error) {
	logger := log.WithFields(log.Fields{
		"check_id":  checkUser.Id,
		"check_pwd": checkUser.Password,
	})

	logger.Info("校验用户密码")

	sql := "select id,password,name,status from user_table where id=?"
	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("校验用户密码出错：" + err.Error())
		return nil, err
	}

	rows, err := stmt.Query(checkUser.Id)
	if err != nil {
		logger.Error("校验用户密码出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user model.User
		err = rows.Scan(&user.Id, &user.Password, &user.Name, &user.Status)
		if err != nil {
			logger.Error("校验用户密码出错：" + err.Error())
			return nil, err
		}
		if checkUser.Password != user.Password {
			return nil, errors.New("密码错误")
		} else {
			return &user, nil
		}
	}

	return nil, errors.New("没有此用户")
}

func (u *UserDao) InsertUser(user *model.User) error {
	logger := log.WithFields(log.Fields{
		"insert_id":     user.Id,
		"insert_psw":    user.Password,
		"insert_status": user.Status,
		"insert_name":   user.Name,
	})

	logger.Info("注册新用户数据入库")

	sql := "insert into user_table(id,password,name,status) values(?,?,?,?)"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("注册新用户数据入库出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(user.Id, user.Password, user.Name, user.Status)
	if err != nil {
		logger.Error("注册新用户数据入库出错：" + err.Error())
		return err
	}

	return nil
}

func (u *UserDao) ModifyUserPassword(id, newPwd, oldPwd string) error {
	logger := log.WithFields(log.Fields{
		"user_id":      id,
		"new_password": newPwd,
		"old_password": oldPwd,
	})

	logger.Info("修改用户密码")

	sql := "select password from user_table where id=?"
	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("修改用户密码出错：" + err.Error())
		return err
	}

	rows, err := stmt.Query(id)
	if err != nil {
		logger.Error("修改用户密码出错：" + err.Error())
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var password string
		err = rows.Scan(&password)
		if err != nil {
			logger.Error("修改用户密码出错：" + err.Error())
			return err
		}

		if oldPwd != password {
			return errors.New("原密码错误")
		} else {
			sql = "update user_table set password=? where id=?"
			stmt, err := client.Prepare(sql)
			if err != nil {
				logger.Error("修改用户密码出错：" + err.Error())
				return err
			}

			_, err = stmt.Exec(newPwd, id)
			if err != nil {
				logger.Error("修改用户密码出错：" + err.Error())
				return err
			}
			return nil
		}
	}

	return errors.New("没有此用户")
}

func (u *UserDao) ModifyUserName(id, name string) error {
	logger := log.WithFields(log.Fields{
		"user_id": id,
		"name":    name,
	})

	logger.Info("修改用户用户名")

	sql := "update user_table set name=? where id=?"
	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("修改用户用户名出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(name, id)
	if err != nil {
		logger.Error("修改用户用户名出错：" + err.Error())
		return err
	}

	return nil
}
