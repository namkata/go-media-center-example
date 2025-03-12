package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/linxGnu/goseaweedfs"
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

// Storage is the interface that wraps the basic storage operations
type Storage interface {
	Upload(file *multipart.FileHeader) (string, error)
	MultipartUpload(file *multipart.FileHeader) (string, error)
	Delete(filename string) error
	GetURL(filename string) string
	GetInternalURL(filename string) string
	GetPresignedURL(fileID string, expiration time.Duration) (string, error)
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
	var options []func(*awsconfig.LoadOptions) error

	// Add region
	options = append(options, awsconfig.WithRegion(config["region"]))

	// Add credentials if provided
	if config["access_key_id"] != "" && config["secret_access_key"] != "" {
		options = append(options, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				config["access_key_id"],
				config["secret_access_key"],
				"",
			),
		))
	}

	// Add custom endpoint if provided
	if config["endpoint"] != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               config["endpoint"],
				SigningRegion:     config["region"],
				HostnameImmutable: true,
			}, nil
		})
		options = append(options, awsconfig.WithEndpointResolverWithOptions(customResolver))
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create S3 client with options
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if config["force_path_style"] == "true" {
			o.UsePathStyle = true
		}
	})

	return &S3Storage{
		client:    s3Client,
		bucket:    config["bucket"],
		region:    config["region"],
		publicURL: config["public_url"],
	}, nil
}

// Upload implements Storage interface for SeaweedFSStorage
func (s *SeaweedFSStorage) Upload(file *multipart.FileHeader) (string, error) {
	// Use multipart upload for large files
	if file.Size > MultipartThreshold {
		return s.MultipartUpload(file)
	}

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

// MultipartUpload implements multipart upload for SeaweedFSStorage
func (s *SeaweedFSStorage) MultipartUpload(file *multipart.FileHeader) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	// Create a buffer for reading chunks
	buffer := make([]byte, DefaultChunkSize)
	chunks := make([]string, 0)

	for {
		n, err := f.Read(buffer)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read file chunk: %v", err)
		}
		if n == 0 {
			break
		}

		// Upload chunk
		chunk := bytes.NewReader(buffer[:n])
		filePart, err := s.client.Upload(
			chunk,
			fmt.Sprintf("%s.part%d", file.Filename, len(chunks)),
			int64(n),
			"",
			"",
		)
		if err != nil {
			// Cleanup uploaded chunks on error
			for _, chunkID := range chunks {
				s.client.DeleteFile(chunkID, nil)
			}
			return "", fmt.Errorf("failed to upload chunk: %v", err)
		}
		chunks = append(chunks, filePart.FileID)
	}

	// For SeaweedFS, we'll store the chunk IDs in the metadata
	// The first chunk's ID will be the main file ID
	if len(chunks) == 0 {
		return "", fmt.Errorf("no chunks uploaded")
	}

	return chunks[0], nil
}

// GetURL implements Storage interface for SeaweedFSStorage
func (s *SeaweedFSStorage) GetURL(fid string) string {
	// Get file extension from metadata if available
	ext := filepath.Ext(fid)

	// Generate a clean filename using the fid
	cleanName := filepath.Base(fid)
	if ext != "" {
		cleanName = fmt.Sprintf("%s%s", cleanName, ext)
	}
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
	// Use multipart upload for large files
	if file.Size > MultipartThreshold {
		return s.MultipartUpload(file)
	}

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

// MultipartUpload implements multipart upload for S3Storage
func (s *S3Storage) MultipartUpload(file *multipart.FileHeader) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	filename := filepath.Join("uploads", filepath.Base(file.Filename))

	// Create multipart upload
	createResp, err := s.client.CreateMultipartUpload(context.TODO(), &s3.CreateMultipartUploadInput{
		Bucket: &s.bucket,
		Key:    &filename,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create multipart upload: %v", err)
	}

	// Upload parts
	var completedParts []types.CompletedPart
	partNumber := int32(1)
	buffer := make([]byte, DefaultChunkSize)

	for {
		n, err := f.Read(buffer)
		if err != nil && err != io.EOF {
			// Abort multipart upload on error
			_, abortErr := s.client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
				Bucket:   &s.bucket,
				Key:      &filename,
				UploadId: createResp.UploadId,
			})
			if abortErr != nil {
				return "", fmt.Errorf("failed to abort multipart upload: %v (original error: %v)", abortErr, err)
			}
			return "", fmt.Errorf("failed to read file chunk: %v", err)
		}
		if n == 0 {
			break
		}

		// Upload part
		currentPartNumber := partNumber
		partInput := &s3.UploadPartInput{
			Bucket:     &s.bucket,
			Key:        &filename,
			PartNumber: &currentPartNumber,
			UploadId:   createResp.UploadId,
			Body:       bytes.NewReader(buffer[:n]),
		}

		partResp, err := s.client.UploadPart(context.TODO(), partInput)
		if err != nil {
			// Abort multipart upload on error
			_, abortErr := s.client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
				Bucket:   &s.bucket,
				Key:      &filename,
				UploadId: createResp.UploadId,
			})
			if abortErr != nil {
				return "", fmt.Errorf("failed to abort multipart upload: %v (original error: %v)", abortErr, err)
			}
			return "", fmt.Errorf("failed to upload part: %v", err)
		}

		// Create a copy of partNumber for the CompletedPart
		pnum := partNumber
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       partResp.ETag,
			PartNumber: &pnum,
		})
		partNumber++
	}

	// Complete multipart upload
	_, err = s.client.CompleteMultipartUpload(context.TODO(), &s3.CompleteMultipartUploadInput{
		Bucket:   &s.bucket,
		Key:      &filename,
		UploadId: createResp.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %v", err)
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

// Update GetPresignedURL for SeaweedFSStorage
func (s *SeaweedFSStorage) GetPresignedURL(fileID string, expiration time.Duration) (string, error) {
    // Generate a token with expiration time
    expirationTime := time.Now().Add(expiration).Unix()
    token := fmt.Sprintf("exp=%d", expirationTime)
    
    // Construct URL with token
    return fmt.Sprintf("%s/%s?%s", s.publicURL, fileID, token), nil
}

// Update GetPresignedURL for S3Storage
func (s *S3Storage) GetPresignedURL(fileID string, expiration time.Duration) (string, error) {
    presignClient := s3.NewPresignClient(s.client)
    
    request, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
        Bucket:          &s.bucket,
        Key:            &fileID,
        ResponseExpires: aws.Time(time.Now().Add(expiration)),
    }, func(opts *s3.PresignOptions) {
        opts.Expires = expiration
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned URL: %v", err)
    }
    
    return request.URL, nil
}
