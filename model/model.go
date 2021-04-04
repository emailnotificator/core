package model

import "time"

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

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Less(i, j int) bool { return a[i].Date.After(a[j].Date) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
