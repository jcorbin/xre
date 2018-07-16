PACKAGE=github.com/jcorbin/xre

all: test xre

clean:
	rm xre cover.out

check:
	@echo rev $(shell git rev-parse HEAD)
	@make lint test

lint:
	golint $(PACKAGE)/...
	go vet $(PACKAGE)/...
	errcheck $(PACKAGE)/...

test:
	go test . -coverprofile cover.out

.PHONY: xre
xre:
	go build -o xre ./cmd
