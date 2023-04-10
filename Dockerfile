FROM golang:1.20.3-alpine3.17 as build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main cmd/exporter/main.go


FROM scratch as release

COPY --from=build /app/main /app/main

EXPOSE 9198

ENTRYPOINT ["/app/main"]
