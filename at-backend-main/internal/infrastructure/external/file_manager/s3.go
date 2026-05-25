package filemanager

import (
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Client struct {
	client     *s3.Client
	uploader   *manager.Uploader
	presigner  *s3.PresignClient
	bucket     string
	endpoint   string
	accessHost string
}

func NewS3Client(ctx context.Context, key, secret, session, bucket, endpoint, accessHost string) (*s3Client, error) {
	cred := credentials.NewStaticCredentialsProvider(key, secret, session)

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(cred),
		config.WithBaseEndpoint(endpoint),
		config.WithRegion("auto"),
	)

	if err != nil {
		return nil, err
	}

	s3client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	presignCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(cred),
		config.WithBaseEndpoint(accessHost), // ← Важно: внешний хост!
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	presignS3Client := s3.NewFromConfig(presignCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	presigner := s3.NewPresignClient(presignS3Client)

	cl := &s3Client{
		client:     s3client,
		uploader:   manager.NewUploader(s3client),
		presigner:  presigner,
		bucket:     bucket,
		accessHost: accessHost,
		endpoint:   endpoint,
	}

	return cl, nil
}

const opSave = "infrastructure.external.FileManager.Save"

// Save создает директорию, указанную в path, и записывает данные в файл с именем name.
func (fm s3Client) Save(ctx context.Context, filePath string, file io.Reader) error {
	if _, err := fm.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(fm.bucket),
		Key:    aws.String(filePath),
		Body:   file,
	}); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opSave,
		)

		return apperror.ErrFileManager
	}

	return nil
}

const opDelete = "infrastructure.external.FileManager.Delete"

// Delete удаляет указанный файл или директорию. Возвращает ErrNotExists, если файл или директория не существуют.
func (fm s3Client) Delete(ctx context.Context, filePath string) error {
	if _, err := fm.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(fm.bucket),
		Key:    aws.String(filePath),
	}); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opDelete,
		)

		return apperror.ErrFileManager
	}

	return nil
}

const opGetURLToFile = "infrastructure.external.FileManager.GetURLToFile"

// GetURLToFile возвращает ссылку, которая позволяет просмотреть файл
func (fm s3Client) GetURLToFile(ctx context.Context, filePath string) (string, error) {
	pr, err := fm.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(fm.bucket),
		Key:    aws.String(filePath),
	}, func(po *s3.PresignOptions) {
		po.Expires = 1 * time.Minute
	})
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opGetURLToFile,
		)

		return "", apperror.ErrFileManager
	}

	return pr.URL, nil
}
