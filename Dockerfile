FROM golang:1.20.7-alpine3.17 as build

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


FROM scratch as release

COPY --from=build /app/main /app/main

EXPOSE 9198

ENTRYPOINT ["/app/main"]
