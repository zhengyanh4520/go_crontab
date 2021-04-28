package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type LogConfig struct {
	Path string `yaml:"path"`
}

type HttpConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type EtcdConfig struct {
	Endpoints   []string `yaml:"end_points"`
	DialTimeout int      `yaml:"dial_timeout"`
	Ttl         int64    `yaml:"ttl"`
}

type MySqlConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type Config struct {
	Log   *LogConfig   `yaml:"log"`
	Http  *HttpConfig  `yaml:"http"`
	Etcd  *EtcdConfig  `yaml:"etcd"`
	MySql *MySqlConfig `yaml:"mysql"`
}

func ReadConfig(src string) (*Config, error) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		fmt.Println("init config error")
		return nil, err
	}

	//fmt.Println(string(content))

	newCon := &Config{}
	err = yaml.Unmarshal(content, &newCon)
	if err != nil {
		fmt.Println("Unmarshal error")
		return nil, err
	}

	return newCon, nil
}
