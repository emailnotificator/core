package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)
import "C"

type Config struct {
	CheckPeriod int       `json:"check_period"`
	Boxes       []MailBox `json:"boxes"`
}

type MailBox struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

const timeFormat = "02 Jan 15:04"

var config *Config
var unreadEmails = map[string][]string{}
var ticker <-chan time.Time
var closeChan chan bool
var lastUpdate string
var newEmails = make([]string, 0)
var isSetup = false
var checkTime = 0

func init() {
	cfgDirPath, err := os.UserConfigDir()

	if err != nil {
		log.Println("")
	}
	if _, err = os.Stat(cfgDirPath + "/email_checker/settings.json"); os.IsNotExist(err) {
		if err = os.MkdirAll(cfgDirPath+"/email_checker", 0777); err != nil {
			log.Println("create cfg dir error:",err)
		}
		if _, err = os.Create(cfgDirPath + "/email_checker/settings.json"); err != nil {
			log.Println("create cfg file error:", err)
		}
	}

	bytes, err := ioutil.ReadFile(cfgDirPath + "/email_checker/settings.json")

	if err != nil {
		log.Println("Read config file error")
	}

	config = new(Config)

	if err = json.Unmarshal(bytes, config); err != nil {
		log.Println("unmarshal config file error: " + err.Error())
	}
	if config.CheckPeriod == 0 {
		config.CheckPeriod = 5
	}
	if config.Boxes == nil {
		config.Boxes = make([]MailBox, 0)
	}
}

func main() {
	Setup()

	for _, box := range config.Boxes {
		checkEmails(box)
	}

	Shutdown()
}

//export Setup
func Setup() {
	log.Println("setup")
	isSetup = true
	closeChan = make(chan bool, 1)
	ticker = time.NewTicker(time.Minute * time.Duration(config.CheckPeriod)).C
	lastUpdate = time.Now().Format(timeFormat)

	go listener()

	for _, box := range config.Boxes {
		checkEmails(box)
	}

	isSetup = false
	checkTime = config.CheckPeriod
}

//export IsSetup
func IsSetup() bool {
	return isSetup
}

//export GetUnread
func GetUnread() *C.char {
	data, _ := json.Marshal(unreadEmails)
	return C.CString(string(data))
}

//export GetUnreadCount
func GetUnreadCount() int {
	cnt := 0

	for _, mails := range unreadEmails {
		cnt += len(mails)
	}

	return cnt
}

//export GetLastUpdate
func GetLastUpdate() *C.char {
	if lastUpdate == "" {
		lastUpdate = time.Now().Format(timeFormat)
	}
	return C.CString(lastUpdate)
}

//export SetConfig
func SetConfig(data *C.char) {
	strData := C.GoString(data)

	if err := json.Unmarshal([]byte(strData), config); err != nil {
		log.Println(err)
	}
	if config.CheckPeriod != checkTime {
		ticker = time.NewTicker(time.Minute * time.Duration(config.CheckPeriod)).C
	}

	configBytes, err := json.MarshalIndent(config, "", "    ")

	if err != nil {
		log.Println("marshal ident error:", err)
	}

	cfgDirPath, err := os.UserConfigDir()

	if err != nil {
		log.Println("get config dir error:", err)
	}
	if err = ioutil.WriteFile(cfgDirPath+"/email_checker/settings.json", configBytes, 0777); err != nil {
		log.Println("save config file error:", err)
	}
}

//export GetConfig
func GetConfig() *C.char {
	data, _ := json.Marshal(config)
	return C.CString(string(data))
}

//export GetNewEmails
func GetNewEmails() *C.char {
	data, _ := json.Marshal(newEmails)
	newEmails = []string{}
	return C.CString(string(data))
}

//export Shutdown
func Shutdown() {
	closeChan <- true
}

//export GetUnreadList
func GetUnreadList() *C.char {
	data  := ""

	for box, emails := range unreadEmails {
		data += box + "\n"

		for i, title := range emails {
			data += fmt.Sprintf("%d. %s\n", i + 1, title)
		}

		data += "\n\n"
	}

	return C.CString(data)
}

func listener() {
	log.Println("start listener")
	for {
		select {
		case <-ticker:
			log.Println("check emails")
			for _, box := range config.Boxes {
				checkEmails(box)
			}
			lastUpdate = time.Now().Format(timeFormat)
		case <-closeChan:
			log.Println("app close")
			return
		}
	}
}

func checkEmails(box MailBox) {
	if _, ok := unreadEmails[box.Login]; !ok {
		unreadEmails[box.Login] = []string{}
	}

	// Connect to server
	imapClient, err := client.DialTLS(fmt.Sprintf("%s:%s", box.Host, box.Port), nil)

	if err != nil {
		log.Fatal(err)
		return
	}

	// Don't forget to logout
	defer imapClient.Logout()

	// Login
	if err = imapClient.Login(box.Login, box.Password); err != nil {
		log.Println(err)
		return
	}

	// List mailboxes
	// Select INBOX
	mbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// Get the last 20 messages
	from := uint32(1)
	to := mbox.Messages

	if mbox.Messages > 50 {
		from = mbox.Messages - 50
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(from, to)

	// get unread
	newEmails = make([]string, 0)
	unreadEnvelopes := make([]imap.Envelope, 0)
	messages := make(chan *imap.Message, 20)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags}, messages)
	}()

	for msg := range messages {
		if !contains(msg.Flags, `\Seen`) {
			unreadEnvelopes = append(unreadEnvelopes, *msg.Envelope)
		}
	}
	if err = <-done; err != nil {
		log.Println(err)
	}
	for _, envelope := range unreadEnvelopes {
		if !contains(unreadEmails[box.Login], envelope.Subject) {
			newEmails = append(newEmails, envelope.Subject)
		}
	}

	delete(unreadEmails, box.Login)
	unreadEmails[box.Login] = []string{}

	for _, envelope := range unreadEnvelopes {
		unreadEmails[box.Login] = append(unreadEmails[box.Login], envelope.Subject)
	}
}

func contains(data []string, str string) bool {
	str = strings.ToLower(str)

	for _, item := range data {
		if strings.ToLower(item) == str {
			return true
		}
	}

	return false
}
