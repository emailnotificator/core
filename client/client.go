package client

import (
	"fmt"
	"log"

	"checker_core/model"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// GetConnect return client with connection to concrete email box
func GetConnect(mailBox model.MailBox) (*client.Client, error) {
	imapClient, err := client.DialTLS(fmt.Sprintf("%s:%s", mailBox.Host, mailBox.Port), nil)
	if err != nil {
		log.Println("dial tls error:", err)
		return nil, err
	}

	if err = imapClient.Login(mailBox.Login, mailBox.Password); err != nil {
		log.Printf("login to %s error: %s", mailBox.Login, err)
		return nil, err
	}

	return imapClient, nil
}

// SetFlag set flag for email
func SetFlag(cfg *model.Config, email string, id int, actionFlag string) error {
	// get needed box
	mailBox := model.MailBox{}

	for _, box := range cfg.Boxes {
		if box.Login == email {
			mailBox = box
			break
		}
	}

	if mailBox.Login == "" {
		return fmt.Errorf("not found %s box", email)
	}

	// set flag for email by email id
	if err := setFlag(mailBox, id, []interface{}{actionFlag}); err != nil {
		log.Println("set flag error:", err)
		return err
	}

	return nil
}

// setFlag set flag for email by email id and send in to remote email box
func setFlag(mailBox model.MailBox, id int, flags []interface{}) error {
	delSeq := imap.SeqSet{}
	delSeq.AddNum(uint32(id))
	operation := imap.FormatFlagsOp(imap.AddFlags, true)

	imapClient, err := GetConnect(mailBox)
	if err != nil {
		return err
	}

	// Login
	defer imapClient.Logout()

	// List mailboxes
	// Select INBOX
	if _, err = imapClient.Select("INBOX", false); err != nil {
		log.Println("select INBOX for set flag error:", err)
		return err
	}

	if err = imapClient.UidStore(&delSeq, operation, flags, nil); err != nil {
		fmt.Println("IMAP Message Flag Update Failed", err)
		return err
	}

	return nil
}
