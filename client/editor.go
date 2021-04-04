package client

import "C"
import (
	"log"

	"checker_core/model"

	"github.com/emersion/go-imap"
)

func DeleteEmail(cfg *model.Config, email string, id int) {
	if err := SetFlag(cfg, email, id, imap.DeletedFlag); err != nil {
		log.Println("set flag error:", err)
		return
	}

	// delete from memory
	deleteMessage(email, id)
}

func SetSeen(cfg *model.Config, email string, id int) {
	if err := SetFlag(cfg, email, id, imap.SeenFlag); err != nil {
		log.Println("set flag error:", err)
		return
	}

	// delete from memory
	deleteMessage(email, id)
}

func deleteMessage(login string, id int) {
	actualMsgs := make([]model.Email, 0)

	for _, message := range model.UnreadEmails[login] {
		if message.Id != id {
			actualMsgs = append(actualMsgs, message)
		}
	}

	delete(model.UnreadEmails, login)
	model.UnreadEmails[login] = actualMsgs
}
