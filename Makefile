# Default target
.DEFAULT_GOAL := all

# Build target
build:
	go build ./pkg/...

# Test target
test: docker-up
	go test -v ./...

# Clean target
clean:
	rm -rf bin

# Clean Redis data
clean-redis:
	docker-compose exec -T redis redis-cli ping || docker-compose up -d redis
	docker-compose exec -T redis redis-cli FLUSHALL

# Docker targets
docker-up: docker-down
	docker-compose up -d

docker-down: clean-redis
	docker-compose down

docker-logs:
	docker-compose logs -f

all: build test # Build and test the project
