# Include environment variables from .env file
ifneq (,$(wildcard .env))
    include .env
    export
endif

Now, let's create a Makefile:

.PHONY: localstack-start localstack-stop localstack-create-bucket localstack-list-buckets localstack-status dev-setup run build test migrate lint clean seaweed-start seaweed-stop seaweed-status

# Application
APP_NAME=media-center
MAIN_PATH=cmd/api/main.go

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

# SeaweedFS configuration (with environment variable fallbacks)
SEAWEED_CONTAINER?=$(APP_NAME)-seaweedfs
SEAWEED_VOLUME?=$(APP_NAME)-seaweedfs-data
SEAWEED_MASTER_PORT?=9333
SEAWEED_VOLUME_PORT?=8080
SEAWEED_DATA_DIR?=/data
SEAWEED_REPLICAS?=1

all: build
# LocalStack management
localstack-start:
	@echo "Starting LocalStack..."
	@if [ ! $$(docker ps -q -f name=$(LOCALSTACK_CONTAINER)) ]; then \
		if [ ! $$(docker ps -aq -f status=exited -f name=$(LOCALSTACK_CONTAINER)) ]; then \
			docker run -d \
				--name $(LOCALSTACK_CONTAINER) \
				-p $(LOCALSTACK_PORT):4566 \
				-p 4510-4559:4510-4559 \
				-e SERVICES=s3 \
				-e DEFAULT_REGION=$(AWS_REGION) \
				-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
				-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
				localstack/localstack:$(LOCALSTACK_VERSION); \
			echo "Waiting for LocalStack to be ready..."; \
			sleep 10; \
		else \
			docker start $(LOCALSTACK_CONTAINER); \
			echo "Waiting for LocalStack to be ready..."; \
			sleep 10; \
		fi \
	else \
		echo "LocalStack is already running"; \
	fi
	@make localstack-create-bucket
	@echo "LocalStack started with:"
	@echo "Endpoint URL: http://localhost:$(LOCALSTACK_PORT)"

localstack-stop:
	@echo "Stopping LocalStack container..."
	@docker stop $(LOCALSTACK_CONTAINER) || true
	@docker rm $(LOCALSTACK_CONTAINER) || true

localstack-create-bucket:
	@echo "Creating S3 bucket..."
	@aws --endpoint-url=http://localhost:$(LOCALSTACK_PORT) \
		s3 mb s3://$(AWS_BUCKET_NAME) \
		--region $(AWS_REGION) || true
	@aws --endpoint-url=http://localhost:$(LOCALSTACK_PORT) \
		s3api put-bucket-acl \
		--bucket $(AWS_BUCKET_NAME) \
		--acl public-read || true

localstack-list-buckets:
	@aws --endpoint-url=http://localhost:$(LOCALSTACK_PORT) \
		s3 ls

localstack-status:
	@echo "Checking LocalStack container status..."
	@docker ps -f name=$(LOCALSTACK_CONTAINER) --format "{{.Status}}" || echo "Container not running"

# Development setup
dev-setup: localstack-start
	@echo "Development environment setup complete"

# Clean up
clean: localstack-stop seaweed-clean
	@echo "Cleaning up build artifacts..."
	@rm -rf bin/
	@rm -rf tmp/
	@echo "Clean up complete"

# SeaweedFS commands
seaweed-start:
	@echo "Starting SeaweedFS..."
	@if [ ! $$(docker ps -q -f name=$(SEAWEED_CONTAINER)) ]; then \
		if [ ! $$(docker ps -aq -f status=exited -f name=$(SEAWEED_CONTAINER)) ]; then \
			docker volume create $(SEAWEED_VOLUME); \
			docker run -d --name $(SEAWEED_CONTAINER) \
				-p $(SEAWEED_MASTER_PORT):9333 \
				-p $(SEAWEED_VOLUME_PORT):8080 \
				-v $(SEAWEED_VOLUME):$(SEAWEED_DATA_DIR) \
				-e SEAWEEDFS_MASTER_PORT=$(SEAWEED_MASTER_PORT) \
				-e SEAWEEDFS_VOLUME_PORT=$(SEAWEED_VOLUME_PORT) \
				-e SEAWEEDFS_VOLUME_MAX=$(SEAWEED_VOLUME_MAX) \
				chrislusf/seaweedfs server \
				-dir=$(SEAWEED_DATA_DIR) \
				-master.port=$(SEAWEED_MASTER_PORT) \
				-volume.port=$(SEAWEED_VOLUME_PORT) \
				-volume.max=$(SEAWEED_VOLUME_MAX) \
		else \
			docker start $(SEAWEED_CONTAINER); \
		fi \
	else \
		echo "SeaweedFS is already running"; \
	fi
	@echo "SeaweedFS started with:"
	@echo "Master URL: http://localhost:$(SEAWEED_MASTER_PORT)"
	@echo "Volume URL: http://localhost:$(SEAWEED_VOLUME_PORT)"
	@echo "Data Directory: $(SEAWEED_DATA_DIR)"

seaweed-stop:
	@echo "Stopping SeaweedFS..."
	@if [ $$(docker ps -q -f name=$(SEAWEED_CONTAINER)) ]; then \
		docker stop $(SEAWEED_CONTAINER); \
	else \
		echo "SeaweedFS is not running"; \
	fi

seaweed-status:
	@if [ $$(docker ps -q -f name=$(SEAWEED_CONTAINER)) ]; then \
		echo "SeaweedFS is running"; \
		echo "Master URL: http://localhost:$(SEAWEED_MASTER_PORT)"; \
		echo "Volume URL: http://localhost:$(SEAWEED_VOLUME_PORT)"; \
		docker logs $(SEAWEED_CONTAINER) --tail 10; \
	else \
		echo "SeaweedFS is not running"; \
	fi

seaweed-logs:
	@if [ $$(docker ps -q -f name=$(SEAWEED_CONTAINER)) ]; then \
		docker logs $(SEAWEED_CONTAINER) --follow; \
	else \
		echo "SeaweedFS is not running"; \
	fi

seaweed-clean:
	@echo "Cleaning up SeaweedFS..."
	@if [ $$(docker ps -aq -f name=$(SEAWEED_CONTAINER)) ]; then \
		docker rm -f $(SEAWEED_CONTAINER) || true; \
	fi
	@if [ $$(docker volume ls -q -f name=$(SEAWEED_VOLUME)) ]; then \
		docker volume rm $(SEAWEED_VOLUME) || true; \
	fi

# Existing commands
run: localstack-start seaweed-start
	@echo "Starting application..."
	$(GORUN) $(MAIN_PATH)

build:
	$(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME) $(MAIN_PATH)

test:
	$(GOTEST) -v ./...

migrate:
	@echo "Running database migrations..."
	@chmod +x scripts/migrate.sh
	@./scripts/migrate.sh up

migrate-down:
	@echo "Rolling back database migrations..."
	@chmod +x scripts/migrate.sh
	@./scripts/migrate.sh down

migrate-reset:
	@echo "Resetting database..."
	@chmod +x scripts/migrate.sh
	@./scripts/migrate.sh reset

migrate-create:
	@read -p "Enter migration name: " name; \
	timestamp=`date +%Y%m%d%H%M%S`; \
	up_file="database/migrations/$${timestamp}_$${name}.sql"; \
	down_file="database/migrations/$${timestamp}_$${name}_down.sql"; \
	touch $$up_file $$down_file; \
	echo "Created migration files:"; \
	echo "  Up: $$up_file"; \
	echo "  Down: $$down_file"

lint:
	golangci-lint run

# Check if gtimeout is available, otherwise don't use timeout
TIMEOUT_CMD := $(shell which gtimeout 2>/dev/null || echo "")
ifneq ($(TIMEOUT_CMD),)
    TIMEOUT = $(TIMEOUT_CMD) 300
else
    TIMEOUT = 
endif

install-deps:
	@echo "Installing required Go packages..."
	@$(TIMEOUT) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest || (echo "Failed to install golangci-lint"; exit 1)
	@$(TIMEOUT) go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest || (echo "Failed to install migrate"; exit 1)
	@$(TIMEOUT) go install github.com/swaggo/swag/cmd/swag@latest || (echo "Failed to install swag"; exit 1)
	@$(TIMEOUT) go install github.com/go-delve/delve/cmd/dlv@latest || (echo "Failed to install delve"; exit 1)
	@echo "All tools installed successfully"

deps:
	@echo "Downloading project dependencies..."
	@$(TIMEOUT) $(GOMOD) download || (echo "Failed to download dependencies"; exit 1)
	@$(TIMEOUT) $(GOMOD) tidy || (echo "Failed to tidy dependencies"; exit 1)
	@echo "Dependencies updated successfully"

.env:
	cp .env.example .env
