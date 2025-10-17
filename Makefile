
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
	@echo "✅ 构建完成: execs/${NAME}"
	@echo "💡 提示: WebP工具会在首次使用图片转换功能时自动下载到 execs/.bin/webp"

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

# 构建所有平台
build_all: build_windows build_linux build_mac build_mac_arm64 build_linux_arm64
	@echo "✅ 所有平台构建完成"
	@echo "💡 提示: WebP工具会在每个平台首次运行时自动下载"
	@echo "💡 如需预打包WebP工具，请在对应平台上运行一次程序，然后将 execs/<platform>/.bin 目录一起打包"

# 打包发布（可选：包含WebP工具）
# 使用方法：
#   1. 运行 make build_<platform>
#   2. 在对应平台上运行一次程序（触发WebP工具下载）
#   3. 运行 make package_<platform> 打包
package_windows:
	@echo "打包 Windows 版本..."
	@mkdir -p dist
	@cd execs/windows && tar -czf ../../dist/${NAME}-${VERSION}-windows-amd64.tar.gz ${NAME}-${VERSION}.exe .bin 2>/dev/null || tar -czf ../../dist/${NAME}-${VERSION}-windows-amd64.tar.gz ${NAME}-${VERSION}.exe
	@echo "✅ Windows 包已创建: dist/${NAME}-${VERSION}-windows-amd64.tar.gz"

package_linux:
	@echo "打包 Linux 版本..."
	@mkdir -p dist
	@cd execs/linux && tar -czf ../../dist/${NAME}-${VERSION}-linux-amd64.tar.gz ${NAME}-${VERSION} .bin 2>/dev/null || tar -czf ../../dist/${NAME}-${VERSION}-linux-amd64.tar.gz ${NAME}-${VERSION}
	@echo "✅ Linux 包已创建: dist/${NAME}-${VERSION}-linux-amd64.tar.gz"

package_mac:
	@echo "打包 macOS 版本..."
	@mkdir -p dist
	@cd execs/mac && tar -czf ../../dist/${NAME}-${VERSION}-darwin-amd64.tar.gz ${NAME}-${VERSION} .bin 2>/dev/null || tar -czf ../../dist/${NAME}-${VERSION}-darwin-amd64.tar.gz ${NAME}-${VERSION}
	@echo "✅ macOS 包已创建: dist/${NAME}-${VERSION}-darwin-amd64.tar.gz"

.PHONY: deps clean build install build_windows build_linux build_mac build_mac_arm64 build_linux_arm64 build_all package_windows package_linux package_mac