package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sort"

	"checker_core/model"
	"checker_core/util"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
)

func GetEmails(box model.MailBox) []model.Email {
	emails := make([]model.Email, 0)
	imapClient, err := GetConnect(box)

	if err != nil {
		log.Println("get connect error:", err)
		return emails
	}

	// List mailboxes
	// Select INBOX
	mbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		log.Println("select INBOX error:", err)
		return nil
	}

	// Don't forget to logout
	defer imapClient.Logout()

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
		done <- imapClient.Fetch(seqSet, []imap.FetchItem{
			fetchSection.FetchItem(),
			imap.FetchEnvelope,
			imap.FetchFlags,
			imap.FetchInternalDate,
			imap.FetchUid,
		}, messages)
	}()

	for msg := range messages {
		if !util.ContainsString(msg.Flags, `\Seen`) {
			unreadEnvelopes = append(unreadEnvelopes, *msg)
		}
	}
	if err = <-done; err != nil {
		log.Println("done error:", err)
	}

	// delete(unreadEmails, box.Login)
	msgs := make([]model.Email, 0, len(unreadEnvelopes))

	for _, envelope := range unreadEnvelopes {
		section := imap.BodySectionName{}
		bodyReader := envelope.GetBody(&section)
		msg := model.Email{
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
			msg.Body = getBody(bodyReader)
		}

		msgs = append(msgs, msg)
	}

	// sort emails
	sort.Sort(model.ByDate(msgs))
	return msgs
}

func getBody(bodyReader io.Reader) string {
	mr, err := mail.CreateReader(bodyReader)

	if err != nil {
		log.Println("create msg reader error:", err)
		return ""
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
				return ""
			}

			return fmt.Sprintf("%v", string(body))
		case *mail.AttachmentHeader:
			// This is an attachment
			filename, _ := partHeader.Filename()

			log.Printf("Got attachment: %v", filename)
		}
	}

	return ""
}

func CheckEmails(box model.MailBox) {
	emails := GetEmails(box)

	if len(emails) != 0 {
		for _, email := range emails {
			if !containsMsg(model.UnreadEmails[box.Login], email) {
				model.NewEmails = append(model.NewEmails, email.Subject)
			}
		}
	}

	model.UnreadEmails[box.Login] = emails
}

func containsMsg(data []model.Email, msg model.Email) bool {
	for _, item := range data {
		if item.Id == msg.Id {
			return true
		}
	}

	return false
}
