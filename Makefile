
NAME := bizyair
VERSION    := v0.0.1
GOOS	   ?= $(shell go env GOOS)
OUTPUT_BIN ?= execs/${GOOS}/${NAME}-${VERSION}
PACKAGE    := github.com/siliconflow/${NAME}-cli
CGO_ENABLED?=0
GO_FLAGS   ?=
GIT_REV    ?= $(shell git rev-parse --short HEAD)
GO_TAGS	   ?= netgo
SOURCE_DATE_EPOCH ?= $(shell date +%s)
ifeq ($(shell uname), Darwin)
DATE       ?= $(shell TZ=GMT date -j -f "%s" ${SOURCE_DATE_EPOCH} +"%Y-%m-%dT%H:%M:%SZ")
else
DATE       ?= $(shell date -u -d @${SOURCE_DATE_EPOCH} +"%Y-%m-%dT%H:%M:%SZ")
endif

deps:
	go mod tidy

clean:
	rm -rf execs/*

build: deps
	@CGO_ENABLED=${CGO_ENABLED} go build ${GO_FLAGS} \
	-ldflags "-w -s -X '${PACKAGE}/meta.Version=${VERSION}' -X '${PACKAGE}/meta.Commit=${GIT_REV}' -X '${PACKAGE}/meta.BuildDate=${DATE}'" \
	-a -tags=${GO_TAGS} -o execs/${NAME} main.go

install: build
	cp execs/${NAME} /usr/local/bin/${NAME}

build_windows:
	@CGO_ENABLED=${CGO_ENABLED} GOOS=windows GOARCH=amd64 go build ${GO_FLAGS} \
	-ldflags "-w -s -X ${PACKAGE}/meta.Version=${VERSION} -X ${PACKAGE}/meta.Commit=${GIT_REV} -X ${PACKAGE}/meta.BuildDate=${DATE}" \
	-a -tags=${GO_TAGS} -o execs/windows/${NAME}-${VERSION}.exe main.go

build_linux:
	@CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=amd64 go build ${GO_FLAGS} \
	-ldflags "-w -s -X ${PACKAGE}/meta.Version=${VERSION} -X ${PACKAGE}/meta.Commit=${GIT_REV} -X ${PACKAGE}/meta.BuildDate=${DATE}" \
	-a -tags=${GO_TAGS} -o execs/linux/${NAME}-${VERSION} main.go

build_mac:
	@CGO_ENABLED=${CGO_ENABLED} GOOS=darwin GOARCH=amd64 go build ${GO_FLAGS} \
	-ldflags "-w -s -X ${PACKAGE}/meta.Version=${VERSION} -X ${PACKAGE}/meta.Commit=${GIT_REV} -X ${PACKAGE}/meta.BuildDate=${DATE}" \
	-a -tags=${GO_TAGS} -o execs/mac/${NAME}-${VERSION} main.go
build_mac_arm64:
	@CGO_ENABLED=${CGO_ENABLED} GOOS=darwin GOARCH=arm64 go build ${GO_FLAGS} \
	-ldflags "-w -s -X ${PACKAGE}/meta.Version=${VERSION} -X ${PACKAGE}/meta.Commit=${GIT_REV} -X ${PACKAGE}/meta.BuildDate=${DATE}" \
	-a -tags=${GO_TAGS} -o execs/mac/${NAME}-${VERSION} main.go

build_linux_arm64:
	@CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=arm64 go build ${GO_FLAGS} \
	-ldflags "-w -s -X ${PACKAGE}/meta.Version=${VERSION} -X ${PACKAGE}/meta.Commit=${GIT_REV} -X ${PACKAGE}/meta.BuildDate=${DATE}" \
	-a -tags=${GO_TAGS} -o execs/linux_arm64/${NAME}-${VERSION} main.go