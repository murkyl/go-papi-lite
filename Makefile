GOARCH = amd64
UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: fmt lint build

build:
	GOOS=$(OS) GOARCH="$(GOARCH)" go build

fmt:
	go fmt $$(go list ./...)
  
lint:
	golint

readme:
	goreadme -credit=false -title "go-papi-lite" > README.md

.PHONY: build fmt lint
