package external

import "context"

type EmailSender interface {
	SendMessage(ctx context.Context, recipientsEmails []string, subject, mailType string, msg []byte) error
}