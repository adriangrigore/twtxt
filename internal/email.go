package internal

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	log "github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

var (
	ErrSendingEmail = errors.New("error: unable to send email")

	passwordResetEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .Username }},

You have requested to have your password on {{ .Pod }} reset for your account.

**IMPORTANT:** If this was __NOT__ you, please ignore this email and contract support!

To reset your password, please visit the following link:

{{ .BaseURL}}/newPassword?token={{ .Token }}

Thank you!

Kind regards

{{ .Pod}} Support
`))

	supportRequestEmailTemplate = template.Must(template.New("email").Parse(`Hello {{ .AdminUser }},

{{ .Name }} <{{ .Email }} from {{ .Pod }} has sent the following support request:

Subject: {{ .Subject }}

{{ .Message }}

Kind regards

Thank you!
`))
)

type PasswordResetEmailContext struct {
	Pod     string
	BaseURL string

	Token    string
	Username string
}

type SupportRequestEmailContext struct {
	Pod       string
	AdminUser string

	Name    string
	Email   string
	Subject string
	Message string
}

func SendEmail(conf *Config, recipients []string, replyTo, subject string, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", conf.SMTPFrom)
	m.SetHeader("To", recipients...)
	m.SetHeader("Reply-To", replyTo)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(conf.SMTPHost, conf.SMTPPort, conf.SMTPUser, conf.SMTPPass)

	err := d.DialAndSend(m)
	if err != nil {
		log.WithError(err).Error("SendEmail() failed")
		return ErrSendingEmail
	}

	return nil
}

func SendPasswordResetEmail(conf *Config, user *User, tokenString string) error {
	recipients := []string{user.Email}
	subject := fmt.Sprintf(
		"[%s]: Password Reset Request for %s",
		conf.Name, user.Username,
	)
	ctx := PasswordResetEmailContext{
		Pod:     conf.Name,
		BaseURL: conf.BaseURL,

		Token:    tokenString,
		Username: user.Username,
	}

	buf := &bytes.Buffer{}
	if err := passwordResetEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, conf.SMTPFrom, subject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending new token to %s", recipients[0])
		return err
	}

	return nil
}

func SendSupportRequestEmail(conf *Config, name, email, subject, message string) error {
	recipients := []string{conf.AdminEmail, email}
	subject = fmt.Sprintf(
		"[%s Support Request]: %s",
		conf.Name, subject,
	)
	ctx := SupportRequestEmailContext{
		Pod:       conf.Name,
		AdminUser: conf.AdminUser,

		Name:    name,
		Email:   email,
		Subject: subject,
		Message: message,
	}

	buf := &bytes.Buffer{}
	if err := supportRequestEmailTemplate.Execute(buf, ctx); err != nil {
		log.WithError(err).Error("error rendering email template")
		return err
	}

	if err := SendEmail(conf, recipients, conf.SMTPFrom, subject, buf.String()); err != nil {
		log.WithError(err).Errorf("error sending support request to %s", recipients[0])
		return err
	}

	return nil
}
