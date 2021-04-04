package main

import (
	"encoding/json"
	"log"
	"time"

	"checker_core/client"
	"checker_core/model"
	"checker_core/util"
)
import "C"

const timeFormat = "02 Jan 15:04"

var (
	closeChan  chan bool
	lastUpdate string
	isSetup    = false
)

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
	return toCString(model.UnreadMails.All())
}

//export GetUnreadCount
func GetUnreadCount() int {
	return model.UnreadMails.Count()
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
	return toCString(model.AppConfig)
}

//export GetNewEmails
func GetNewEmails() *C.char {
	cString := toCString(model.NewEmails)
	model.NewEmails = []string{} // clear new email because frontend now know about it

	return cString
}

//export Shutdown
func Shutdown() {
	closeChan <- true
}

//export DeleteEmail
func DeleteEmail(email *C.char, id int) {
	item := C.GoString(email)
	client.DeleteEmail(model.AppConfig, item, id)
}

//export SetSeen
func SetSeen(email *C.char, id int) {
	item := C.GoString(email)
	client.SetSeen(model.AppConfig, item, id)
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
			log.Println("app shutdown")
			close(closeChan)
			return
		}
	}
}

func toCString(data interface{}) *C.char {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Println("marshal to CString error:", err)
		data = []byte("")
	}

	return C.CString(string(jsonBytes))
}
