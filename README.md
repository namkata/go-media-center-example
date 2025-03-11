Here's an updated version of the README.md with more comprehensive information:

```markdown:/Users/trannam/Products/go-media-center-example/README.md
# Go Media Center

A media management system built with Go, Gin, and PostgreSQL.

## Features

- User authentication with JWT
- File upload and management
- Folder organization
- Media tagging and searching
- Image processing and optimization
- RESTful API
- Pagination and filtering
- Export functionality (CSV, JSON)

## Prerequisites

- Go 1.23 or higher
- PostgreSQL 14 or higher
- Make (optional, for using Makefile commands)

## Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/go-media-center-example.git
cd go-media-center-example
```

2. Create environment file:
```bash
make .env
```

3. Update the `.env` file with your configuration:
```plaintext
DB_HOST=localhost
DB_PORT=5432
DB_NAME=media_center
DB_USER=postgres
DB_PASSWORD=your_password

JWT_SECRET=your_secret_key
STORAGE_PATH=./storage
MAX_UPLOAD_SIZE=10485760
```

4. Install dependencies:
```bash
make deps
```

5. Run migrations:
```bash
make migrate
```

6. Build and run:
```bash
make build
make run
```

## API Endpoints

### Authentication
- POST `/api/auth/register` - Register new user
- POST `/api/auth/login` - Login user

### Media
- POST `/api/media` - Upload media
- GET `/api/media` - List media
- GET `/api/media/:id` - Get media details
- PUT `/api/media/:id` - Update media
- DELETE `/api/media/:id` - Delete media
- POST `/api/media/batch` - Batch operations

### Folders
- POST `/api/folders` - Create folder
- GET `/api/folders` - List folders
- PUT `/api/folders/:id` - Update folder
- DELETE `/api/folders/:id` - Delete folder

### Export
- GET `/api/export/csv` - Export media list as CSV
- GET `/api/export/json` - Export media list as JSON

## Development

### Available Make Commands
- `make run` - Run the application
- `make build` - Build the application
- `make test` - Run tests
- `make migrate` - Run database migrations
- `make migrate-create` - Create new migration
- `make lint` - Run linter
- `make clean` - Clean build artifacts
- `make deps` - Install dependencies

### Project Structure
```
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── api/
│   ├── config/
│   ├── database/
│   ├── models/
│   └── utils/
├── database/
│   └── migrations/
├── storage/
├── .env
├── .env.example
├── go.mod
├── go.sum
└── Makefile
```

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request
```