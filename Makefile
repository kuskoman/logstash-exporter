GOOS_VALUES := linux darwin windows
GOOS_BINARIES := $(foreach goos,$(GOOS_VALUES),out/main-$(goos))
GOOS_EXES := $(foreach goos,$(GOOS_VALUES),$(if $(filter windows,$(goos)),out/main-$(goos).exe,out/main-$(goos)))

all: $(GOOS_BINARIES)

out/main-%:
	CGO_ENABLED=0 GOOS=$* go build -a -installsuffix cgo -ldflags="-w -s" -o out/main-$* cmd/exporter/main.go

run:
	go run cmd/exporter/main.go

build-linux: out/main-linux
build-darwin: out/main-darwin
build-windows: out/main-windows

clean:
	rm -f $(GOOS_EXES)

.DEFAULT_GOAL := run
