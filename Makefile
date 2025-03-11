
Now, let's create a Makefile:

```makefile:%2FUsers%2Ftrannam%2FProducts%2Fgo-media-center-example%2FMakefile
.PHONY: run build test migrate lint clean

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

all: build

run:
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
	psql -U postgres -d media_center -f database/migrations/*.sql
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

clean:
	rm -rf bin/
	rm -rf tmp/

deps:
	$(GOMOD) download
	$(GOMOD) tidy

.env:
	cp .env.example .env