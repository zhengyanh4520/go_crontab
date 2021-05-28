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

	sql := "select id,password,name from UserTable where id=?"
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
		err = rows.Scan(&user.Id, &user.Password, &user.Name)
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
		"insert_user": user.ToString(),
	})

	logger.Info("注册新用户数据入库")

	sql := "insert into UserTable(id,password,name,company,department,duties,phone) values(?,?,?,?,?,?,?)"

	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("注册新用户数据入库出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(user.Id, user.Password, user.Name, user.Company, user.Department, user.Duties, user.Phone)
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

	sql := "select password from UserTable where id=?"
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
			sql = "update UserTable set password=? where id=?"
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

func (u *UserDao) ModifyUserInfo(user *model.User) error {
	logger := log.WithFields(log.Fields{
		"modify_user": user.ToString(),
	})

	logger.Info("修改用户信息")

	sql := "update UserTable set name=?,company=?,department=?,duties=?,phone=? where id=?"
	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("修改用户信息出错：" + err.Error())
		return err
	}

	_, err = stmt.Exec(user.Name, user.Company, user.Department, user.Duties, user.Phone, user.Id)
	if err != nil {
		logger.Error("修改用户信息出错：" + err.Error())
		return err
	}

	return nil
}

func (u *UserDao) GetUserInfo(id string) (*model.User, error) {
	logger := log.WithFields(log.Fields{
		"user_id": id,
	})

	logger.Info("获取用户信息")

	sql := "select id,name,company,department,duties,phone from UserTable where id=?"
	client := getMysqlClient()
	stmt, err := client.Prepare(sql)
	if err != nil {
		logger.Error("获取用户信息出错：" + err.Error())
		return nil, err
	}

	rows, err := stmt.Query(id)
	if err != nil {
		logger.Error("获取用户信息出错：" + err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user model.User
		err = rows.Scan(&user.Id, &user.Name, &user.Company, &user.Department, &user.Duties, &user.Phone)
		if err != nil {
			logger.Error("获取用户信息出错：" + err.Error())
			return nil, err
		}
		return &user, nil
	}

	return nil, errors.New("没有此用户")
}
