# Media Center Example

A Go-based media center application that demonstrates file storage handling using both SeaweedFS and AWS S3 (with LocalStack support for development).

## Features

- File upload and management
- Support for multiple storage backends:
  - SeaweedFS (distributed file system)
  - AWS S3 (with LocalStack support for local development)
- Image processing capabilities
- Video metadata extraction
- Folder organization
- Tag management
- User authentication
- RESTful API

## Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- PostgreSQL
- FFmpeg (for video processing)
- AWS CLI (for LocalStack interaction)

## Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/go-media-center-example
   cd go-media-center-example
   ```

2. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

3. Start the development environment:
   ```bash
   make dev-setup
   ```

4. Run the application:
   ```bash
   make run
   ```

## Storage Configuration

### LocalStack S3 (Development)

The project uses LocalStack to simulate AWS S3 locally during development.

1. Start LocalStack and create the test bucket:
   ```bash
   make localstack-start
   ```

2. Verify the setup:
   ```bash
   make localstack-status
   make localstack-list-buckets
   ```

3. Run the S3 test script:
   ```bash
   go run scripts/test_s3.go
   ```

### SeaweedFS

Alternatively, you can use SeaweedFS as your storage backend:

1. Start SeaweedFS:
   ```bash
   make seaweed-start
   ```

2. Check the status:
   ```bash
   make seaweed-status
   ```

3. View logs:
   ```bash
   make seaweed-logs
   ```

## Environment Variables

Key configuration options in `.env`:

```env
# Storage Configuration
STORAGE_PROVIDER=s3  # Options: seaweedfs, s3
MAX_UPLOAD_SIZE=104857600  # 100MB in bytes

# AWS S3/LocalStack Configuration
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_BUCKET_NAME=media-center-bucket
AWS_PUBLIC_URL=http://localhost:4566
AWS_ENDPOINT=http://localhost:4566
AWS_FORCE_PATH_STYLE=true

# SeaweedFS Configuration
SEAWEED_CONTAINER=media-center-seaweedfs
SEAWEED_VOLUME=media-center-seaweedfs-data
SEAWEED_MASTER_PORT=9333
SEAWEED_VOLUME_PORT=8080
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register a new user
- `POST /api/v1/auth/login` - Login and get JWT token

### Media Management
- `POST /api/v1/media/upload` - Upload media file
- `GET /api/v1/media/list` - List all media files
- `GET /api/v1/media/:id` - Get media details
- `PUT /api/v1/media/:id` - Update media metadata
- `DELETE /api/v1/media/:id` - Delete media file

### Folders
- `POST /api/v1/folders` - Create folder
- `GET /api/v1/folders` - List folders
- `PUT /api/v1/folders/:id` - Update folder
- `DELETE /api/v1/folders/:id` - Delete folder

## Development Commands

```bash
# Build the application
make build

# Run tests
make test

# Run linter
make lint

# Create a new migration
make migrate-create

# Apply migrations
make migrate

# Clean up
make clean
```

## File Upload Specifications

- Maximum file size: 100MB (configurable)
- Supported image formats: JPG, PNG, GIF
- Supported video formats: MP4, MOV, AVI
- Automatic metadata extraction for both images and videos
- Image processing capabilities (resize, crop)
- Multipart upload support for large files

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [SeaweedFS](https://github.com/chrislusf/seaweedfs) for distributed file system
- [LocalStack](https://localstack.cloud/) for AWS service emulation
- [FFmpeg](https://ffmpeg.org/) for video processing