GOOS_VALUES := linux darwin windows
GOOS_BINARIES := $(foreach goos,$(GOOS_VALUES),out/main-$(goos))
GOOS_EXES := $(foreach goos,$(GOOS_VALUES),$(if $(filter windows,$(goos)),out/main-$(goos),out/main-$(goos)))

GITHUB_REPO := github.com/kuskoman/logstash-exporter
VERSION ?= $(shell git symbolic-ref --short HEAD)
GIT_COMMIT := $(shell git rev-parse HEAD)
DOCKER_IMG ?= "logstash-exporter"

# ****************************** NOTE ****************************** #
# Commands description was made using the following syntax:          #
# https://stackoverflow.com/a/59087509                               #
#                                                                    #
# To write command description use "#:" before command definition    #
# ****************************************************************** #

#: Builds binary executables for all OS (Win, Darwin, Linux)
all: $(GOOS_BINARIES)

VERSIONINFO_PKG := config
ldflags := -s -w \
	-X '$(GITHUB_REPO)/$(VERSIONINFO_PKG).Version=$(VERSION)' \
	-X '$(GITHUB_REPO)/$(VERSIONINFO_PKG).GitCommit=$(GIT_COMMIT)' \
	-X '$(GITHUB_REPO)/$(VERSIONINFO_PKG).BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S%Z)'

out/main-%:
	CGO_ENABLED=0 GOOS=$* go build -a -installsuffix cgo -ldflags="$(ldflags)" -o out/main-$* cmd/exporter/main.go

#: Runs the Go Exporter application
run:
	go run cmd/exporter/main.go

#: Builds a binary executable for Linux
build-linux: out/main-linux
#: Builds a binary executable for Darwin
build-darwin: out/main-darwin
#: Builds a binary executable for Windows
build-windows: out/main-windows

#: Builds a Docker image for the Go Exporter application
build-docker:
	docker build -t $(DOCKER_IMG) --build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) .

# Builds for Linux X86, Apple Silicon/AWS Graviton. Requires docker buildx (Docker 19.03+)
#: Builds a multi-arch Docker image (`amd64` and `arm64`)
build-docker-multi:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMG) --push .

#: Deletes all binary executables in the out directory
clean:
	rm -f $(GOOS_EXES)

#: Runs all tests
test:
	go test -race -v ./...

#: Displays test coverage report
test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

#: Starts a Docker-compose configuration
compose:
	docker-compose up -d --build

#: Starts a Docker-compose configuration until it's ready
wait-for-compose:
	docker-compose up -d --wait

#: Stops a Docker-compose configuration
compose-down:
	docker-compose down

#: Verifies the metrics from the Go Exporter application
verify-metrics:
	./scripts/verify_metrics.sh

#: Pulls the Docker image from the registry
pull:
	docker-compose pull

#: Shows logs from the Docker-compose configuration
logs:
	docker-compose logs -f

#: Minifies the binary executables
minify:
	upx -9 $(GOOS_EXES)

#: Installs readme-generator-for-helm tool
install-helm-readme:
	./scripts/install_helm_readme_generator.sh

#: Generates Helm chart README.md file
helm-readme:
	./scripts/generate_helm_readme.sh

#: Cleans Elasticsearch data, works only with default ES port. The command may take a very long time to complete
clean-elasticsearch:
	@indices=$(shell curl -s -X GET "http://localhost:9200/_cat/indices" | awk '{print $$3}') ;\
	for index in $$indices ; do \
		echo "Deleting all documents from index $$index" ;\
		curl -X POST "http://localhost:9200/$$index/_delete_by_query?conflicts=proceed" -H "Content-Type: application/json" -d '{"query": {"match_all": {}}}' ;\
		echo "" ;\
	done

#: Upgrades all dependencies
upgrade-dependencies:
	go get -u ./...

#: Shows info about available commands
help:
	@grep -B1 -E "^[a-zA-Z0-9_-]+\:([^\=]|$$)" Makefile \
	| grep -v -- -- \
	| sed 'N;s/\n/###/' \
	| sed -n 's/^#: \(.*\)###\(.*\):.*/\2###\1/p' \
	| column -t  -s '###'


.DEFAULT_GOAL := run
