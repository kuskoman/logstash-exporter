GOOS_VALUES := linux darwin windows
GOOS_EXPORTER_BINARIES := $(foreach goos,$(GOOS_VALUES),out/exporter-$(goos))
GOOS_CONTROLLER_BINARIES := $(foreach goos,$(GOOS_VALUES),out/controller-$(goos))
GOOS_BINARIES := $(GOOS_EXPORTER_BINARIES) $(GOOS_CONTROLLER_BINARIES)
GOOS_EXES := $(foreach goos,$(GOOS_VALUES),$(if $(filter windows,$(goos)),out/exporter-$(goos),out/exporter-$(goos))) $(foreach goos,$(GOOS_VALUES),$(if $(filter windows,$(goos)),out/controller-$(goos),out/controller-$(goos)))

GITHUB_REPO := github.com/kuskoman/logstash-exporter
VERSION ?= $(shell git symbolic-ref --short HEAD)
SEMANTIC_VERSION ?= $(shell git describe --tags --abbrev=1 --dirty 2> /dev/null)
GIT_COMMIT := $(shell git rev-parse HEAD)
DOCKER_IMG ?= logstash-exporter

# ****************************** NOTE ****************************** #
# Commands description was made using the following syntax:          #
# https://stackoverflow.com/a/59087509                               #
#                                                                    #
# To write command description use "#:" before command definition    #
# ****************************************************************** #

#: Builds binary executables for all OS (Win, Darwin, Linux)
all: $(GOOS_BINARIES)

VERSIONINFO_PKG := pkg/config
ldflags := -s -w \
	-X '$(GITHUB_REPO)/$(VERSIONINFO_PKG).Version=$(VERSION)' \
	-X '$(GITHUB_REPO)/$(VERSIONINFO_PKG).SemanticVersion=$(SEMANTIC_VERSION)' \
	-X '$(GITHUB_REPO)/$(VERSIONINFO_PKG).GitCommit=$(GIT_COMMIT)' \
	-X '$(GITHUB_REPO)/$(VERSIONINFO_PKG).BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S%Z)'

out/exporter-%:
	CGO_ENABLED=0 GOOS=$* go build -a -installsuffix cgo -ldflags="$(ldflags)" -o out/exporter-$* cmd/exporter/main.go

out/controller-%:
	CGO_ENABLED=0 GOOS=$* go build -a -installsuffix cgo -ldflags="$(ldflags)" -o out/controller-$* cmd/controller/main.go

#: Runs the Go Exporter application
run:
	go run cmd/exporter/main.go

#: Runs the Go Controller application
run-controller:
	go run cmd/controller/main.go

#: Runs the Go Exporter application with watching the configuration file
run-and-watch-config:
	go run cmd/exporter/main.go -watch

#: Builds exporter and controller binaries for Linux
build-linux: out/exporter-linux out/controller-linux
#: Builds exporter and controller binaries for Darwin
build-darwin: out/exporter-darwin out/controller-darwin
#: Builds exporter and controller binaries for Windows
build-windows: out/exporter-windows out/controller-windows
#: Builds exporter and controller binaries for Linux ARM
build-linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -a -installsuffix cgo -ldflags="$(ldflags)" -o out/exporter-linux-arm cmd/exporter/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -a -installsuffix cgo -ldflags="$(ldflags)" -o out/controller-linux-arm cmd/controller/main.go

#: Builds a Docker image for the Go Exporter application
build-docker:
	docker build -t $(DOCKER_IMG) --build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) .

#: Builds a Docker image for the Kubernetes Controller
build-docker-controller:
	docker build -t "$(DOCKER_IMG)-controller" --build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) -f Dockerfile.controller .

#: Builds both exporter and controller Docker images
build-docker-all: build-docker build-docker-controller

# Builds for Linux X86, Apple Silicon/AWS Graviton. Requires docker buildx (Docker 19.03+)
#: Builds a multi-arch Docker image (`amd64` and `arm64`) for the exporter
build-docker-multi:
	docker buildx build --push --platform linux/amd64,linux/arm64 -t $(DOCKER_IMG) --build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) .

#: Builds a multi-arch Docker image (`amd64` and `arm64`) for the controller
build-docker-controller-multi:
	docker buildx build --push --platform linux/amd64,linux/arm64 -t $(DOCKER_IMG)-controller --build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) -f Dockerfile.controller .

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

#: Run Helm unit tests
helm-test:
	helm unittest ./chart

#: Cleans Elasticsearch data. The command may take a very long time to complete
clean-elasticsearch:
	@echo "Cleaning up Elasticsearch indices..."
	@ELASTICSEARCH_PORT=$${ELASTICSEARCH_PORT:-9200} ;\
	indices=$$(curl -s -X GET "http://localhost:$$ELASTICSEARCH_PORT/_cat/indices?h=index" | grep -E 'logstash-*') ;\
	for index in $$indices ; do \
		echo "Deleting documents from index $$index..." ;\
		curl -X POST "http://localhost:$$ELASTICSEARCH_PORT/$$index/_delete_by_query?conflicts=proceed" -H "Content-Type: application/json" -d '{"query": {"match_all": {}}}' ;\
		echo "Completed deleting documents from index $$index." ;\
	done
	@echo "Cleanup completed."

#: Cleans Prometheus data
clean-prometheus:
	@PROMETHEUS_PORT=$${PROMETHEUS_PORT:-9090} ;\
	set -euo pipefail ;\
	echo "Deleting series from Prometheus..." ;\
	ALL_SERIES=$$(curl -Ss http://localhost:$$PROMETHEUS_PORT/api/v1/label/__name__/values | jq -r '.data[]') ;\
	for series in $$ALL_SERIES ; do \
		echo "Deleting series $$series..." ;\
		encodedSeries=$$(printf '%s' "$$series" | jq -sRr @uri) ;\
		curl -X POST "http://localhost:$$PROMETHEUS_PORT/api/v1/admin/tsdb/delete_series?match[]=$$encodedSeries" ;\
		echo "Completed deleting series $$series." ;\
	done

#: Upgrades all dependencies
upgrade-dependencies:
	go get -u ./...
	go mod tidy

#: Migrates configuration from v1 to v2
migrate-v1-to-v2:
	./scripts/migrate_v1_to_v2.sh

#: Update Makefile descriptions in main README.md
update-readme-descriptions:
	./scripts/add_descriptions_to_readme.sh

#: Updates snapshot for test data and runs tests
update-snapshots:
	UPDATE_SNAPS=true go test ./...

#: Shows info about available commands
help:
	@grep -B1 -E "^[a-zA-Z0-9_-]+:([^\=]|$$)" Makefile \
	| grep -v -- -- \
	| sed "N;s/\n/###/" \
	| sed -n "s/^#: \(.*\)###\(.*\):.*/\2###\1/p" \
	| column -t  -s "###"


.DEFAULT_GOAL := run
