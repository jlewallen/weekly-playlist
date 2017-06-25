package main

import (
	"bytes"
	"html/template"
	"log"
	"net/smtp"
	"strconv"
)

type EmailUser struct {
	Username    string
	Password    string
	EmailServer string
	Port        int
}

type SmtpTemplateData struct {
	From    string
	To      string
	Subject string
	Body    string
}

func SendEmail(info string) {
	var err error

	data := &SmtpTemplateData{
		"jcl.automated@gmail.com",
		"jlewalle@gmail.com",
		"weekly-playlist: Report",
		info,
	}

	body, err := ParseTemplate("report.html", data)

	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	subject := "Subject: " + data.Subject + "\n"
	msg := []byte(subject + mime + "\n" + body)

	emailUser := &EmailUser{smtpUsername, smtpPassword, "smtp.gmail.com", 587}
	auth := smtp.PlainAuth("", emailUser.Username, emailUser.Password, emailUser.EmailServer)
	server := emailUser.EmailServer + ":" + strconv.Itoa(emailUser.Port)

	err = smtp.SendMail(server, auth, emailUser.Username, []string{"jlewalle@gmail.com"}, msg)
	if err != nil {
		log.Print("ERROR: Unable to send email", err)
	}
}

func ParseTemplate(templateFileName string, data interface{}) (body string, err error) {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return
	}
	body = buf.String()
	return
}
