package util

import "C"
import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"checker_core/model"
)

func ContainsString(data []string, str string) bool {
	str = strings.ToLower(str)

	for _, item := range data {
		if strings.ToLower(item) == str {
			return true
		}
	}

	return false
}

func UpdateConfig(strData string) {
	if err := json.Unmarshal([]byte(strData), model.AppConfig); err != nil {
		log.Println(err)
	}
	if model.AppConfig.CheckPeriod != model.CheckTime {
		model.Ticker = time.NewTicker(time.Minute * time.Duration(model.AppConfig.CheckPeriod)).C
	}

	configBytes, err := json.MarshalIndent(model.AppConfig, "", "    ")

	if err != nil {
		log.Println("marshal ident error:", err)
	}

	cfgDirPath, err := os.UserConfigDir()

	if err != nil {
		log.Println("get model.AppConfig dir error:", err)
	}
	if err = ioutil.WriteFile(cfgDirPath+"/email_checker/settings.json", configBytes, 0777); err != nil {
		log.Println("save model.AppConfig file error:", err)
	}
}
