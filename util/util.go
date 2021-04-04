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
	// unmarshal json to config struct
	if err := json.Unmarshal([]byte(strData), model.AppConfig); err != nil {
		log.Println(err)
	}

	// upd check period if it need
	if model.AppConfig.CheckPeriod != model.CheckTime {
		model.Ticker = time.NewTicker(time.Minute * time.Duration(model.AppConfig.CheckPeriod)).C
	}

	// pretty marshal to json
	configBytes, err := json.MarshalIndent(model.AppConfig, "", "    ")
	if err != nil {
		log.Println("marshal ident error:", err)
	}

	// write json to config file
	cfgDirPath, err := os.UserConfigDir()
	if err != nil {
		log.Println("get model.AppConfig dir error:", err)
	}

	if err = ioutil.WriteFile(cfgDirPath+"/email_checker/settings.json", configBytes, 0777); err != nil {
		log.Println("save model.AppConfig file error:", err)
	}
}
