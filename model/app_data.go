package model

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var UnreadEmails = map[string][]Email{}
var AppConfig *Config
var NewEmails = make([]string, 0)
var CheckTime = 0
var Ticker <-chan time.Time

func InitConfig() error {
	cfgDirPath, err := os.UserConfigDir()

	if err != nil {
		log.Println("get user config dir error:", err)
		return err
	}
	if _, err = os.Stat(cfgDirPath + "/email_checker/settings.json"); os.IsNotExist(err) {
		if err = os.MkdirAll(cfgDirPath+"/email_checker", 0777); err != nil {
			log.Println("create cfg dir error:", err)
		}
		if _, err = os.Create(cfgDirPath + "/email_checker/settings.json"); err != nil {
			log.Println("create cfg file error:", err)
		}
	}

	bytes, err := ioutil.ReadFile(cfgDirPath + "/email_checker/settings.json")

	if err != nil {
		log.Println("Read model.AppConfig file error")
		return err
	}

	AppConfig = new(Config)

	if err = json.Unmarshal(bytes, AppConfig); err != nil {
		log.Println("unmarshal AppConfig file error: " + err.Error())
		return err
	}
	if AppConfig.CheckPeriod == 0 {
		AppConfig.CheckPeriod = 5
	}
	if AppConfig.Boxes == nil {
		AppConfig.Boxes = make([]MailBox, 0)
	}

	return nil
}
