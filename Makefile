GIT_REV=$(shell git describe --abbrev=8 --always --tags)
VER=$(GIT_REV)-$(shell date +%Y%m%d-%H:%M:%S)

build:
	@echo "  >>>  Building binary files..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.version=$(VER)" -o bin/stubrouter-linux cmd/main.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.version=$(VER)" -o bin/stubrouter-macos cmd/main.go

dist: build
	@echo "  >>>  Make distribution..."
	@tar cvf dist.tar config.toml web -C bin stubrouter-linux stubrouter-macos

clean:
	@rm -rf bin dist.tar


.PHONY: build dist clean
