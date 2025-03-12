package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/linxGnu/goseaweedfs"
)

// StorageProvider represents the type of storage being used
type StorageProvider string

const (
	SeaweedFS StorageProvider = "seaweedfs"
	S3        StorageProvider = "s3"
)

// Storage is the interface that wraps the basic storage operations
type Storage interface {
	Upload(file *multipart.FileHeader) (string, error)
	Delete(filename string) error
	GetURL(filename string) string
	GetInternalURL(filename string) string
}

// SeaweedFSStorage implements Storage interface using SeaweedFS
type SeaweedFSStorage struct {
	client      *goseaweedfs.Seaweed
	masterURL   string
	internalURL string
	publicURL   string
}

// S3Storage implements Storage interface using AWS S3
type S3Storage struct {
	client    *s3.Client
	bucket    string
	region    string
	publicURL string
}

// NewStorage creates a new storage instance based on the provider
func NewStorage(provider StorageProvider, config map[string]string) (Storage, error) {
	switch provider {
	case SeaweedFS:
		return NewSeaweedFSStorage(config)
	case S3:
		return NewS3Storage(config)
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", provider)
	}
}

// NewSeaweedFSStorage creates a new SeaweedFS storage instance
func NewSeaweedFSStorage(config map[string]string) (*SeaweedFSStorage, error) {
	client, err := goseaweedfs.NewSeaweed(
		config["master_url"],
		[]string{},
		int64(10), // timeout in seconds
		&http.Client{Timeout: 10 * time.Second},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SeaweedFS client: %v", err)
	}

	return &SeaweedFSStorage{
		client:      client,
		masterURL:   config["master_url"],
		internalURL: config["internal_url"],
		publicURL:   config["public_url"],
	}, nil
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(config map[string]string) (*S3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(config["region"]),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	client := s3.NewFromConfig(awsCfg)
	return &S3Storage{
		client:    client,
		bucket:    config["bucket"],
		region:    config["region"],
		publicURL: config["public_url"],
	}, nil
}

// Upload implements Storage interface for SeaweedFSStorage
func (s *SeaweedFSStorage) Upload(file *multipart.FileHeader) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	filePart, err := s.client.Upload(
		f,
		file.Filename,
		file.Size,
		"",
		"",
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload to SeaweedFS: %v", err)
	}

	return filePart.FileID, nil
}

// GetURL implements Storage interface for SeaweedFSStorage
func (s *SeaweedFSStorage) GetURL(fid string) string {
	// Get file extension from metadata if available
	ext := filepath.Ext(fid)
	if ext == "" {
		ext = ".bin" // default extension if none is found
	}

	// Generate a clean filename using the fid
	cleanName := fmt.Sprintf("%s%s", filepath.Base(fid), ext)
	return fmt.Sprintf("%s/media/files/%s", s.publicURL, cleanName)
}

// GetInternalURL implements Storage interface for SeaweedFSStorage
func (s *SeaweedFSStorage) GetInternalURL(fid string) string {
	// For internal access, use the volume server's URL directly
	return fmt.Sprintf("%s/%s", s.internalURL, fid)
}

// Delete implements Storage interface for SeaweedFSStorage
func (s *SeaweedFSStorage) Delete(fid string) error {
	err := s.client.DeleteFile(fid, nil)
	if err != nil {
		return fmt.Errorf("failed to delete from SeaweedFS: %v", err)
	}
	return nil
}

// Upload implements Storage interface for S3Storage
func (s *S3Storage) Upload(file *multipart.FileHeader) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	filename := filepath.Join("uploads", filepath.Base(file.Filename))

	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &filename,
		Body:   f,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	return filename, nil
}

// GetURL implements Storage interface for S3Storage
func (s *S3Storage) GetURL(filename string) string {
	cleanName := filepath.Base(filename)
	return fmt.Sprintf("%s/media/files/%s", s.publicURL, cleanName)
}

// GetInternalURL implements Storage interface for S3Storage
func (s *S3Storage) GetInternalURL(filename string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, filename)
}

// Delete implements Storage interface for S3Storage
func (s *S3Storage) Delete(filename string) error {
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &filename,
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %v", err)
	}
	return nil
}
