package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
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

type Email struct {
	Id      int       `json:"id"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	MailBox string    `json:"mail_box"`
	From    string    `json:"from"`
	Date    time.Time `json:"date"`
}

type ByDate []Email

const timeFormat = "02 Jan 15:04"

var config *Config

var unreadEmails = map[string][]Email{}
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
			log.Println("create cfg dir error:", err)
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
	unread := make([]Email, 0, GetUnreadCount())

	for _, value := range unreadEmails {
		unread = append(unread, value...)
	}

	sort.Sort(ByDate(unread))
	data, _ := json.Marshal(unread)
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

//export DeleteEmail
func DeleteEmail(email *C.char, id int) {
	em := C.GoString(email)
	mailBox := MailBox{}

	for _, box := range config.Boxes {
		if box.Login == em {
			mailBox = box
		}
	}
	if err := setFlag(mailBox, id, []interface{}{imap.DeletedFlag}); err != nil {
		log.Println("set flag error:", err)
		return
	}

	// delete from memory
	deleteMessage(em, id)
}

//export SetSeen
func SetSeen(email *C.char, id int) {
	em := C.GoString(email)
	mailBox := MailBox{}

	for _, box := range config.Boxes {
		if box.Login == em {
			mailBox = box
		}
	}
	if err := setFlag(mailBox, id, []interface{}{imap.SeenFlag}); err != nil {
		log.Println("set flag error:", err)
		return
	}

	// delete from memory
	deleteMessage(em, id)
}

func setFlag(mailBox MailBox, id int, flags []interface{}) error {
	delSeq := imap.SeqSet{}
	delSeq.AddNum(uint32(id))
	operation := imap.FormatFlagsOp(imap.AddFlags, true)
	imapClient, err := client.DialTLS(fmt.Sprintf("%s:%s", mailBox.Host, mailBox.Port), nil)

	if err != nil {
		return err
	}

	// Login
	defer imapClient.Logout()

	if err = imapClient.Login(mailBox.Login, mailBox.Password); err != nil {
		return err
	}

	// List mailboxes
	// Select INBOX
	if _, err = imapClient.Select("INBOX", false); err != nil {
		return err
	}
	if err = imapClient.UidStore(&delSeq, operation, flags, nil); err != nil {
		fmt.Println("IMAP Message Flag Update Failed", err)
		return err
	}

	return nil
}

func listener() {
	log.Println("start listener")
	for {
		select {
		case <-ticker:
			log.Println("check emails")
			newEmails = make([]string, 0)

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
	// Connect to server
	imapClient, err := client.DialTLS(fmt.Sprintf("%s:%s", box.Host, box.Port), nil)

	if err != nil {
		log.Println(err)
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

	// Get the last 50 messages
	from := uint32(1)
	to := mbox.Messages

	if mbox.Messages > 50 {
		from = mbox.Messages - 50
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(from, to)

	// get unread
	unreadEnvelopes := make([]imap.Message, 0)
	messages := make(chan *imap.Message, 20)
	done := make(chan error, 1)
	fetchSection := imap.BodySectionName{
		Peek: true,
	}

	go func() {
		//done <- imapClient.Fetch(seqSet, []imap.FetchItem{imap.FetchFull}, messages)
		done <- imapClient.Fetch(seqSet, []imap.FetchItem{
			fetchSection.FetchItem(),
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchInternalDate,
			imap.FetchUid,
		}, messages)
	}()

	for msg := range messages {
		if !contains(msg.Flags, `\Seen`) {
			unreadEnvelopes = append(unreadEnvelopes, *msg)
		}
	}
	if err = <-done; err != nil {
		log.Println("done error:", err)
	}

	delete(unreadEmails, box.Login)
	msgs := make([]Email, 0, len(unreadEnvelopes))

	for _, envelope := range unreadEnvelopes {
		var section imap.BodySectionName
		bodyReader := envelope.GetBody(&section)
		msg := Email{
			Id:      int(envelope.Uid),
			Subject: envelope.Envelope.Subject,
			Date:    envelope.Envelope.Date,
			MailBox: box.Login,
		}

		if envelope.Envelope.From != nil && len(envelope.Envelope.From) > 0 {
			from := envelope.Envelope.From[0]
			msg.From = from.PersonalName + " | " + from.Address()
		}
		if bodyReader != nil {
			// read mail body
			mr, err := mail.CreateReader(bodyReader)

			if err != nil {
				log.Println("create msg reader error:", err)
			}
			for {
				part, err := mr.NextPart()

				if err == io.EOF {
					break
				} else if err != nil {
					log.Println("read next part error:", err)
					continue
				}
				switch partHeader := part.Header.(type) {
				case *mail.InlineHeader:
					// This is the message's text (can be plain-text or HTML)
					body, err := ioutil.ReadAll(part.Body)

					if err != nil {
						log.Println("read mail body error:", err)
					}

					msg.Body = fmt.Sprintf("%v", string(body))
				case *mail.AttachmentHeader:
					// This is an attachment
					filename, _ := partHeader.Filename()

					log.Printf("Got attachment: %v", filename)
				}
			}
		}

		msgs = append(msgs, msg)
	}

	// sort emails
	sort.Sort(ByDate(msgs))
	unreadEmails[box.Login] = msgs
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

func containsMsg(data []Email, msg imap.Message) bool {
	uid := int(msg.Uid)

	for _, item := range data {
		if item.Id == uid {
			return true
		}
	}

	return false
}

func deleteMessage(login string, id int) {
	actualMsgs := make([]Email, 0)

	for _, message := range unreadEmails[login] {
		if message.Id != id {
			actualMsgs = append(actualMsgs, message)
		}
	}

	delete(unreadEmails, login)
	unreadEmails[login] = actualMsgs
}

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Less(i, j int) bool { return a[i].Date.After(a[j].Date) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
