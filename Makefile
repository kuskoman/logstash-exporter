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

# Builds for Linux X86, Apple Silicon/AWS Graviton. Requires docker buildx (Docker 19.03+)
build-docker-multi:
	docker buildx build --platform linux/amd64,linux/arm64 -t logstash-exporter .

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
	./scripts/verify_metrics.sh

pull:
	docker-compose pull

logs:
	docker-compose logs -f

minify:
	upx -9 $(GOOS_EXES)

install-helm-readme:
	./scripts/install_helm_readme_generator.sh

helm-readme:
	./scripts/generate_helm_readme.sh

.DEFAULT_GOAL := run
