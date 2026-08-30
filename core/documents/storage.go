package documents

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type DocumentStorage struct {
	s3Client        *s3.Client
	s3PresignClient *s3.PresignClient
}

func NewDocumentStorage(
	s3Client *s3.Client) *DocumentStorage {

	presignClient := s3.NewPresignClient(s3Client)
	return &DocumentStorage{
		s3Client:        s3Client,
		s3PresignClient: presignClient,
	}
}

func (s *DocumentStorage) GetObjectURL(ctx context.Context,
	key *string,
	expiresIn time.Duration) (string, error) {

	if key == nil {
		return "", nil
	}

	res, err := s.s3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String("torb"),
		Key:    key,
	}, func(po *s3.PresignOptions) {
		po.Expires = expiresIn
	})
	if err != nil {
		return "", fmt.Errorf("s3 PresignGetObject: %w", err)
	}

	return res.URL, nil
}

func (s *DocumentStorage) StoreObject(ctx context.Context,
	key, fileType string,
	body io.Reader) error {
	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String("torb"),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(fileType),
	})
	if err != nil {
		return err
	}

	return nil
}
