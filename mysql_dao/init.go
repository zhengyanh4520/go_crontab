package mysql_dao

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"go_crontab/config"
	"sync"
)

var (
	db   *sql.DB = nil
	once sync.Once
)

func InitMysql(mysqlCon *config.MySqlConfig) error {
	var err error = nil

	once.Do(func() {

		str := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysqlCon.User,
			mysqlCon.Password, mysqlCon.Host, mysqlCon.Port, mysqlCon.Database)

		tempDb, e := sql.Open("mysql", str)
		if e != nil {
			fmt.Println(e)
			err = e
			return
		}

		db = tempDb
		log.Info("连接数据库成功")

	})

	return err
}

func getMysqlClient() *sql.DB {
	if db == nil {
		log.Error("数据库未连接")
	}
	return db
}
