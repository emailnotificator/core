package model

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"
)

var (
	UnreadMails = UnreadEmails{}
	AppConfig   *Config
	NewEmails   = make([]string, 0)
	CheckTime   = 0
	Ticker      <-chan time.Time
)

// UnreadEmails struct for work with unread emails
type UnreadEmails struct {
	EmailMap map[string][]Email
}

// Count return count emails from all boxes
func (um *UnreadEmails) Count() int {
	cnt := 0

	for _, mails := range um.EmailMap {
		cnt += len(mails)
	}

	return cnt
}

// All return email list sorting by date (desc) from all boxes
func (um *UnreadEmails) All() []Email {
	unread := make([]Email, 0, um.Count())

	for _, value := range um.EmailMap {
		unread = append(unread, value...)
	}

	sort.Sort(ByDate(unread))

	return unread
}

// InitConfig create config object and file if is it needed
func InitConfig() error {
	cfgDirPath, err := os.UserConfigDir()
	if err != nil {
		log.Println("get user config dir error:", err)
		return err
	}

	// create directory and file if it is not exists
	if _, err = os.Stat(cfgDirPath + "/email_checker/settings.json"); os.IsNotExist(err) {
		if err = os.MkdirAll(cfgDirPath+"/email_checker", 0777); err != nil {
			log.Println("create cfg dir error:", err)
		}
		if _, err = os.Create(cfgDirPath + "/email_checker/settings.json"); err != nil {
			log.Println("create cfg file error:", err)
		}
	}

	// read config and create settings object
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
