package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/linxGnu/goseaweedfs"

	"go-media-center-example/internal/config"
)

// StorageProvider represents the type of storage being used
type StorageProvider string

const (
	SeaweedFS StorageProvider = "seaweedfs"
	S3        StorageProvider = "s3"
	// Default chunk size for multipart uploads (5MB)
	DefaultChunkSize = 5 * 1024 * 1024
	// Threshold for using multipart upload (10MB)
	MultipartThreshold = 10 * 1024 * 1024
)

// Storage defines the interface for storage providers
type Storage interface {
	Upload(reader io.Reader, filename string) (string, error)
	Download(path string) (io.ReadCloser, error)
	Delete(path string) error
	GetPublicURL(path string) string
	GetInternalURL(path string) string
	UploadBytes(data []byte, filename string) (string, error)
	GetPresignedURL(fileID string, expiration time.Duration) (string, error)
}

// S3Storage implements the Storage interface for AWS S3
type S3Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// Upload uploads a file to S3
func (s *S3Storage) Upload(reader io.Reader, filename string) (string, error) {
	key := filepath.Clean(filename)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	_, err = s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}
	return key, nil
}

// Download downloads a file from S3
func (s *S3Storage) Download(path string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %v", err)
	}
	return result.Body, nil
}

// Delete deletes a file from S3
func (s *S3Storage) Delete(path string) error {
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %v", err)
	}
	return nil
}

// GetPublicURL returns the public URL for a file in S3
func (s *S3Storage) GetPublicURL(path string) string {
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", s.publicURL, path)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, path)
}

// GetInternalURL returns the internal URL for a file in S3
func (s *S3Storage) GetInternalURL(path string) string {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, path)
}

// UploadBytes uploads bytes to S3
func (s *S3Storage) UploadBytes(data []byte, filename string) (string, error) {
	key := filepath.Clean(filename)
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload bytes to S3: %v", err)
	}
	return key, nil
}

// GetPresignedURL generates a presigned URL for S3
func (s *S3Storage) GetPresignedURL(fileID string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)
	request, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket:          aws.String(s.bucket),
		Key:             aws.String(fileID),
		ResponseExpires: aws.Time(time.Now().Add(expiration)),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}
	return request.URL, nil
}

// SeaweedFSStorage implements the Storage interface for SeaweedFS
type SeaweedFSStorage struct {
	client      *goseaweedfs.Filer
	internalURL string
	publicURL   string
}

// Upload implements Storage interface for SeaweedFSStorage
func (s *SeaweedFSStorage) Upload(reader io.Reader, filename string) (string, error) {
	// Read the entire file into memory since SeaweedFS client doesn't support streaming
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Upload the file
	filePart, err := s.client.Upload(
		bytes.NewReader(data),
		int64(len(data)), // size
		filename,         // path
		"default",        // collection
		"",               // ttl
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload to SeaweedFS: %v", err)
	}

	return filePart.FileID, nil
}

// Download downloads a file from SeaweedFS
func (s *SeaweedFSStorage) Download(path string) (io.ReadCloser, error) {
	reader, _, err := s.client.Get(path, url.Values{}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from SeaweedFS: %v", err)
	}
	return io.NopCloser(bytes.NewReader(reader)), nil
}

// Delete deletes a file from SeaweedFS
func (s *SeaweedFSStorage) Delete(path string) error {
	if err := s.client.Delete(path, url.Values{}); err != nil {
		return fmt.Errorf("failed to delete file from SeaweedFS: %v", err)
	}
	return nil
}

// GetPublicURL returns the public URL for a file in SeaweedFS
func (s *SeaweedFSStorage) GetPublicURL(path string) string {
	return fmt.Sprintf("%s/%s", s.publicURL, path)
}

// GetInternalURL returns the internal URL for a file in SeaweedFS
func (s *SeaweedFSStorage) GetInternalURL(path string) string {
	return fmt.Sprintf("%s/%s", s.internalURL, path)
}

// UploadBytes uploads bytes to SeaweedFS
func (s *SeaweedFSStorage) UploadBytes(data []byte, filename string) (string, error) {
	path := filepath.Clean(filename)
	collection := "default"
	ttl := ""

	if _, err := s.client.Upload(bytes.NewReader(data), -1, path, collection, ttl); err != nil {
		return "", fmt.Errorf("failed to upload bytes to SeaweedFS: %v", err)
	}
	return path, nil
}

// GetPresignedURL generates a presigned URL for SeaweedFS
func (s *SeaweedFSStorage) GetPresignedURL(fileID string, expiration time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiration).Unix()
	token := fmt.Sprintf("exp=%d", expirationTime)
	return fmt.Sprintf("%s/%s?%s", s.publicURL, fileID, token), nil
}

var (
	provider Storage
	once     sync.Once
)

// GetProvider returns the configured storage provider
func GetProvider() Storage {
	once.Do(func() {
		var err error
		cfg := config.GetConfig()
		var storageConfig map[string]string

		switch cfg.Storage.Provider {
		case "s3":
			storageConfig = map[string]string{
				"region":            cfg.Storage.S3.Region,
				"access_key_id":     cfg.Storage.S3.AccessKeyID,
				"secret_access_key": cfg.Storage.S3.SecretAccessKey,
				"bucket":            cfg.Storage.S3.BucketName,
				"endpoint":          cfg.Storage.S3.Endpoint,
				"force_path_style":  "true",
				"public_url":        cfg.Storage.S3.PublicURL,
			}
			provider, err = NewS3Storage(storageConfig)
		case "seaweedfs":
			storageConfig = map[string]string{
				"master_url":   cfg.Storage.SeaweedFS.MasterURL,
				"internal_url": fmt.Sprintf("http://localhost:%d", cfg.Storage.SeaweedFS.VolumePort),
				"public_url":   fmt.Sprintf("http://localhost:%s", cfg.Server.Port),
			}
			provider, err = NewSeaweedFSStorage(storageConfig)
		default:
			panic(fmt.Sprintf("Unsupported storage provider: %s", cfg.Storage.Provider))
		}
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize storage provider: %v", err))
		}
	})
	return provider
}

// NewStorage creates a new storage provider instance
func NewStorage(provider StorageProvider, config map[string]string) (Storage, error) {
	switch provider {
	case S3:
		return NewS3Storage(config)
	case SeaweedFS:
		return NewSeaweedFSStorage(config)
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", provider)
	}
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(config map[string]string) (Storage, error) {
	cfg := aws.Config{
		Region: config["region"],
		Credentials: credentials.NewStaticCredentialsProvider(
			config["access_key_id"],
			config["secret_access_key"],
			"",
		),
	}

	if endpoint := config["endpoint"]; endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     config["region"],
				HostnameImmutable: true,
			}, nil
		})
		cfg.EndpointResolverWithOptions = customResolver
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = config["force_path_style"] == "true"
	})

	return &S3Storage{
		client:    client,
		bucket:    config["bucket"],
		publicURL: config["public_url"],
	}, nil
}

// NewSeaweedFSStorage creates a new SeaweedFS storage instance
func NewSeaweedFSStorage(config map[string]string) (Storage, error) {
	client, err := goseaweedfs.NewFiler(config["master_url"], nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SeaweedFS client: %v", err)
	}

	return &SeaweedFSStorage{
		client:      client,
		internalURL: config["internal_url"],
		publicURL:   config["public_url"],
	}, nil
}
