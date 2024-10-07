FROM golang:1.23.1-alpine3.19 as build

ARG VERSION \
    GIT_COMMIT \
    GITHUB_REPO="github.com/kuskoman/logstash-exporter"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s \
    -X ${GITHUB_REPO}/config.Version=${VERSION} \
    -X ${GITHUB_REPO}/config.GitCommit=${GIT_COMMIT} \
    -X ${GITHUB_REPO}/config.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o main cmd/exporter/main.go

RUN grep "nobody:x:65534" /etc/passwd > /app/user

FROM scratch as release

COPY --from=build /app/main /app/main
COPY --from=build /app/user /etc/passwd

EXPOSE 9198
USER 65534
ENTRYPOINT ["/app/main"]
