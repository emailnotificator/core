package main

import (
	"encoding/json"
	"log"
	"sort"
	"time"

	"checker_core/client"
	"checker_core/model"
	"checker_core/util"
)
import "C"

const timeFormat = "02 Jan 15:04"

var closeChan chan bool
var lastUpdate string
var isSetup = false

func init() {
	if err := model.InitConfig(); err != nil {
		log.Fatal("init config error:", err)
	}
}

func main() {
}

//export Setup
func Setup() {
	log.Println("setup")
	isSetup = true
	closeChan = make(chan bool, 1)
	model.Ticker = time.NewTicker(time.Minute * time.Duration(model.AppConfig.CheckPeriod)).C
	lastUpdate = time.Now().Format(timeFormat)

	go listener()

	for _, box := range model.AppConfig.Boxes {
		client.CheckEmails(box)
	}

	isSetup = false
	model.CheckTime = model.AppConfig.CheckPeriod
}

//export IsSetup
func IsSetup() bool {
	return isSetup
}

//export GetUnread
func GetUnread() *C.char {
	unread := make([]model.Email, 0, GetUnreadCount())

	for _, value := range model.UnreadEmails {
		unread = append(unread, value...)
	}

	sort.Sort(model.ByDate(unread))
	data, _ := json.Marshal(unread)

	return C.CString(string(data))
}

//export GetUnreadCount
func GetUnreadCount() int {
	cnt := 0

	for _, mails := range model.UnreadEmails {
		cnt += len(mails)
	}

	return cnt
}

//export GetLastUpdate
func GetLastUpdate() *C.char {
	return C.CString(lastUpdate)
}

//export SetConfig
func SetConfig(data *C.char) {
	go util.UpdateConfig(C.GoString(data))
}

//export GetConfig
func GetConfig() *C.char {
	data, _ := json.Marshal(model.AppConfig)
	return C.CString(string(data))
}

//export GetNewEmails
func GetNewEmails() *C.char {
	data, _ := json.Marshal(model.NewEmails)
	model.NewEmails = []string{}
	return C.CString(string(data))
}

//export Shutdown
func Shutdown() {
	closeChan <- true
}

//export DeleteEmail
func DeleteEmail(email *C.char, id int) {
	em := C.GoString(email)

	client.DeleteEmail(model.AppConfig, em, id)
}

//export SetSeen
func SetSeen(email *C.char, id int) {
	em := C.GoString(email)

	client.SetSeen(model.AppConfig, em, id)
}

func listener() {
	log.Println("start listener")
	for {
		select {
		case <-model.Ticker:
			log.Println("check emails")
			model.NewEmails = make([]string, 0)

			for _, box := range model.AppConfig.Boxes {
				client.CheckEmails(box)
			}

			lastUpdate = time.Now().Format(timeFormat)
		case <-closeChan:
			log.Println("app close")
			return
		}
	}
}
