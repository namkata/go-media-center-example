package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	// Load configuration from environment
	endpoint := os.Getenv("AWS_ENDPOINT")
	region := os.Getenv("AWS_REGION")
	bucket := os.Getenv("AWS_BUCKET_NAME")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if endpoint == "" || region == "" || bucket == "" || accessKey == "" || secretKey == "" {
		log.Fatal("Missing required environment variables")
	}

	// Create custom endpoint resolver
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, reg string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	// Configure AWS SDK
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Test bucket listing
	fmt.Println("Listing buckets...")
	result, err := client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		log.Fatalf("Failed to list buckets: %v", err)
	}

	for _, b := range result.Buckets {
		fmt.Printf("Bucket: %s, Created: %s\n", *b.Name, b.CreationDate)
	}

	// Test file upload
	testContent := []byte("Hello, LocalStack S3!")
	testKey := "test.txt"

	fmt.Printf("\nUploading test file to bucket %s...\n", bucket)
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(testKey),
		Body:   bytes.NewReader(testContent),
	})
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}

	// Test file download
	fmt.Println("Downloading test file...")
	getResult, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(testKey),
	})
	if err != nil {
		log.Fatalf("Failed to download file: %v", err)
	}
	defer getResult.Body.Close()

	// Read and verify content
	buf := new(bytes.Buffer)
	buf.ReadFrom(getResult.Body)
	content := buf.String()

	fmt.Printf("Downloaded content: %s\n", content)
	if content != string(testContent) {
		log.Fatal("Content mismatch!")
	}

	fmt.Println("\nS3 test completed successfully!")
}
