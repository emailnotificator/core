package client

import "C"
import (
	"log"

	"checker_core/model"

	"github.com/emersion/go-imap"
)

// DeleteEmail set flag 'deleted' for email in remote box and delete email from application email list
func DeleteEmail(cfg *model.Config, email string, id int) {
	if err := SetFlag(cfg, email, id, imap.DeletedFlag); err != nil {
		log.Println("set flag error:", err)
		return
	}

	// delete from memory
	deleteMessage(email, id)
}

// DeleteEmail set flag 'seen' for email in remote box and delete email from application email list
func SetSeen(cfg *model.Config, email string, id int) {
	if err := SetFlag(cfg, email, id, imap.SeenFlag); err != nil {
		log.Println("set flag error:", err)
		return
	}

	// delete from memory
	// todo: delete or not seen emails set on config
	deleteMessage(email, id)
}

// deleteMessage delete email from application email list
func deleteMessage(login string, id int) {
	actualMsgs := make([]model.Email, 0, len(model.UnreadMails.EmailMap[login]))

	for _, message := range model.UnreadMails.EmailMap[login] {
		if message.Id != id {
			actualMsgs = append(actualMsgs, message)
		}
	}

	delete(model.UnreadMails.EmailMap, login)
	model.UnreadMails.EmailMap[login] = actualMsgs
}
