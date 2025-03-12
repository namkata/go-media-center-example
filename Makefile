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
	@echo "Starting LocalStack container..."
	@docker run -d \
		--name $(LOCALSTACK_CONTAINER) \
		-p $(LOCALSTACK_PORT):4566 \
		-p 4510-4559:4510-4559 \
		-e SERVICES=s3 \
		-e DEFAULT_REGION=$(AWS_REGION) \
		-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		localstack/localstack:$(LOCALSTACK_VERSION)
	@echo "Waiting for LocalStack to be ready..."
	@sleep 10
	@make localstack-create-bucket

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
clean: localstack-stop
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

seaweed-clean: seaweed-stop
	@echo "Cleaning up SeaweedFS..."
	@if [ $$(docker ps -aq -f name=$(SEAWEED_CONTAINER)) ]; then \
		docker rm $(SEAWEED_CONTAINER); \
	fi
	@if [ $$(docker volume ls -q -f name=$(SEAWEED_VOLUME)) ]; then \
		docker volume rm $(SEAWEED_VOLUME); \
	fi

# Existing commands
run: seaweed-start
	$(GORUN) $(MAIN_PATH)

build:
	$(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME) $(MAIN_PATH)

test:
	$(GOTEST) -v ./...

migrate:
	@echo "Running database migrations..."
	@if [ ! -d "database/migrations" ]; then \
		echo "Creating migrations directory..."; \
		mkdir -p database/migrations; \
	fi
	PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f database/migrations/*.sql
	@echo "Migrations completed successfully"

migrate-create:
	@read -p "Enter migration name: " name; \
	timestamp=`date +%Y%m%d%H%M%S`; \
	filename="database/migrations/$${timestamp}_$${name}.sql"; \
	touch $$filename; \
	echo "Created migration file: $$filename"

migrate-rollback:
	@echo "Rolling back last migration..."
	psql -U postgres -d media_center -c "SELECT rollback_migration();"

lint:
	golangci-lint run

clean: seaweed-clean
	rm -rf bin/
	rm -rf tmp/

deps:
	$(GOMOD) download
	$(GOMOD) tidy

.env:
	cp .env.example .env