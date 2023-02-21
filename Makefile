NAME = gdut-drcom-go
PARAMS = -v -trimpath -ldflags "-s -w -buildid="
MAIN = ./cmd/gdut-drcom-go

build:
	go build $(PARAMS) $(MAIN)
