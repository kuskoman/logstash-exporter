FROM golang:1.25.3-alpine3.21 AS build

ARG VERSION \
    GIT_COMMIT \
    GITHUB_REPO="github.com/kuskoman/logstash-exporter"

WORKDIR /app

RUN grep "nobody:x:65534" /etc/passwd > /app/user

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s \
    -X ${GITHUB_REPO}/pkg/config.Version=${VERSION} \
    -X ${GITHUB_REPO}/pkg/config.GitCommit=${GIT_COMMIT} \
    -X ${GITHUB_REPO}/pkg/config.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o exporter cmd/exporter/main.go

FROM scratch AS release

COPY --from=build /app/user /etc/passwd
COPY --from=build /app/exporter /app/exporter

EXPOSE 9198
USER 65534
ENTRYPOINT ["/app/exporter"]

ENV EXPORTER_CONFIG_LOCATION=/app/config.yml
