GOOS_VALUES := linux darwin windows
GOOS_BINARIES := $(foreach goos,$(GOOS_VALUES),out/main-$(goos))
GOOS_EXES := $(foreach goos,$(GOOS_VALUES),$(if $(filter windows,$(goos)),out/main-$(goos),out/main-$(goos)))

all: $(GOOS_BINARIES)

out/main-%:
	CGO_ENABLED=0 GOOS=$* go build -a -installsuffix cgo -ldflags="-w -s" -o out/main-$* cmd/exporter/main.go

run:
	go run cmd/exporter/main.go

build-linux: out/main-linux
build-darwin: out/main-darwin
build-windows: out/main-windows

build-docker:
	docker build -t logstash-exporter .

clean:
	rm -f $(GOOS_EXES)

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

compose:
	docker-compose up -d --build

wait-for-compose:
	docker-compose up -d --wait

compose-down:
	docker-compose down

verify-metrics:
	./scripts/verify-metrics.sh

pull:
	docker-compose pull

logs:
	docker-compose logs -f

minify:
	upx -9 $(GOOS_EXES)

.DEFAULT_GOAL := run
