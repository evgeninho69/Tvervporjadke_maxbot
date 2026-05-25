package email

import (
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
	"sync"
)

type client struct {
	Host           string
	Address        string
	SenderEmail    string
	SenderPassword string
	tlsConfig      *tls.Config
	auth           smtp.Auth
}

var instance client
var once sync.Once

func NewClient(Host, Port, SenderEmail, SenderPassword string) (client, error) {
	var clientErr error
	once.Do(func() {
		tlsconfig := &tls.Config{
			ServerName: Host,
		}

		auth := smtp.PlainAuth("", SenderEmail, SenderPassword, Host)

		instance = client{
			Host:           Host,
			Address:        Host + ":" + Port,
			SenderEmail:    SenderEmail,
			SenderPassword: SenderPassword,
			auth:           auth,
			tlsConfig:      tlsconfig,
		}
	})

	return instance, clientErr
}

const opSendMessage = "infrastructure.external.EmailSender.SendMessage"

func (c client) SendMessage(ctx context.Context, recipientsEmails []string, subject, mailType string, msg []byte) error {
	encodedSubject := mime.BEncoding.Encode("utf-8", subject)

	header := fmt.Appendf(nil,
		"From: %s\nSubject: %s\nMIME-version: 1.0;\nContent-Type: %s; charset=\"UTF-8\";\n\n",
		c.SenderEmail,
		encodedSubject,
		mailType,
	)

	msg = append(header, msg...)

	logs.Info(
		"email prepared",
		common.GetCorrelationID(ctx),
		opSendMessage,
	)

	conn, err := tls.Dial("tcp", c.Address, c.tlsConfig)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	if err := client.Auth(c.auth); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	if err = client.Mail(c.SenderEmail); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	if err = client.Rcpt(recipientsEmails[0]); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	writer, err := client.Data()
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	if _, err = fmt.Fprint(writer, string(msg)); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	if err = writer.Close(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	if err = client.Quit(); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSendMessage,
		)

		return apperror.ErrEmailSender
	}

	logs.Info(
		"email sent",
		common.GetCorrelationID(ctx),
		opSendMessage,
	)

	return nil
}
