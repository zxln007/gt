# Copyright (c) 2022 Institute of Software, Chinese Academy of Sciences (ISCAS)
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DATE=$(shell date '+%F %T')
BRANCH=$(shell git branch --show-current)
COMMIT=$(shell git rev-parse HEAD | cut -c1-7)
NAME=gt
EXE=$(shell go env GOEXE)
VERSION=$(NAME) - $(DATE) - $(BRANCH) $(COMMIT)
ifdef STATIC_LINK
	export GO_STATIC_LINK_FLAG=-extldflags=-static
endif
ifdef RACE_CHECK
	export GO_RACE=-race
endif
UPDATE_SUBMODULE_COMMAND=git submodule update --init --recursive
ifdef WITH_OFFICIAL_WEBRTC
	UPDATE_SUBMODULE_COMMAND=echo 'skiped update_submodule'
endif
RELEASE_OPTIONS=$(GO_RACE) -tags release -trimpath -ldflags "$(GO_STATIC_LINK_FLAG) -s -w -X 'github.com/isrc-cas/gt/predef.Version=$(VERSION)'"
DEBUG_OPTIONS=$(GO_RACE) -trimpath -ldflags "$(GO_STATIC_LINK_FLAG) -X 'github.com/isrc-cas/gt/predef.Version=$(VERSION)'"
SOURCES=$(shell find . -path ./dep -prune -o -type f -name "*.go")
FRONTEND_DIR=web/front
SOURCES_FRONT = $(shell find $(FRONTEND_DIR) -type d \( -name 'node_modules' -o -name 'dist' \) -prune -o -type f \( -name '*.ts' -o -name '*.vue' -o -name '*.scss' -o -name '*.json' -o -name '*.cjs' -o -name '*.config.ts' -o -name '*.html' \) -print)
SERVER_FRONT_DIR=server/web
CLIENT_FRONT_DIR=client/web
TARGET?=$(shell gcc -dumpmachine)
TARGET_OS=$(shell echo $(TARGET) | awk -F '-' '{print $$2}')
TARGET_CPU=$(shell echo $(TARGET) | awk -F '-' '{print $$1}')
export GOOS?=$(shell go env GOOS)
export GOARCH?=$(shell go env GOARCH)
export CC=$(TARGET)-gcc -w
export CXX=$(TARGET)-g++ -w
OBJCOPY=$(TARGET)-objcopy
ifeq ($(TARGET_OS), native)
	TARGET_OS=
endif
ifeq ($(TARGET_OS), linux)
	ifeq ($(TARGET_CPU), riscv64)
		RUST_TARGET?=$(shell echo $(TARGET) | awk -F '-' '{print "riscv64gc-unknown-linux-" $$3}')
	else
		RUST_TARGET?=$(shell echo $(TARGET) | awk -F '-' '{print $$1 "-unknown-linux-" $$3}')
	endif
	DEF_WEBRTC_COND?=-DWEBRTC_POSIX
	DEF_LDFLAGS?=-ldl
endif
ifeq ($(TARGET_OS), apple)
	export CC=clang
	export CXX=clang++
	TARGET_OS=mac
	OBJCOPY=llvm-objcopy
	ifeq ($(TARGET_CPU), arm64)
		RUST_TARGET?=aarch64-apple-darwin
	else
		ifndef RUST_TARGET
			RUST_TARGET:=$(TARGET_CPU)-apple-darwin
		endif
	endif
	DEF_WEBRTC_COND?=-DWEBRTC_MAC -DWEBRTC_POSIX
	DEF_LDFLAGS?=-framework Cocoa
endif
ifeq ($(TARGET_CPU), native)
	TARGET_CPU=
endif
ifeq ($(TARGET_CPU), aarch64)
    TARGET_CPU=arm64
endif
ifeq ($(TARGET_CPU), x86_64)
    TARGET_CPU=x64
endif
ifeq ($(TARGET_CPU), i386)
    TARGET_CPU=x86
endif
PWD=$(shell pwd)
MSQUIC_MODE?=build
MSQUIC_BUILD_ROOT?=$(PWD)/dep/_msquic
MSQUIC_PREBUILT_ROOT?=$(PWD)/prebuilt/msquic/$(TARGET)
MSQUIC_ARTIFACT_LIB_NAME?=libmsquic.a
ifeq ($(MSQUIC_MODE),prebuilt)
	MSQUIC_INCLUDE_DIR:=$(MSQUIC_PREBUILT_ROOT)/include
	MSQUIC_LIB_FILE:=$(MSQUIC_PREBUILT_ROOT)/lib/$(MSQUIC_ARTIFACT_LIB_NAME)
	MSQUIC_METADATA_FILE:=$(MSQUIC_PREBUILT_ROOT)/metadata.json
else
	MSQUIC_INCLUDE_DIR:=$(MSQUIC_BUILD_ROOT)/src/inc
	MSQUIC_LIB_FILE:=$(MSQUIC_BUILD_ROOT)/out/$(TARGET)/bin/Release/libmsquic.a
endif

WEBRTC_MODE?=build
WEBRTC_BUILD_ROOT?=$(PWD)/dep/_google-webrtc/src
WEBRTC_PREBUILT_ROOT?=$(PWD)/prebuilt/webrtc/$(TARGET)
WEBRTC_ARTIFACT_LIB_NAME?=libwebrtc.a
ifeq ($(WEBRTC_MODE),prebuilt)
	WEBRTC_LIB_FILE:=$(WEBRTC_PREBUILT_ROOT)/lib/$(WEBRTC_ARTIFACT_LIB_NAME)
	WEBRTC_METADATA_FILE:=$(WEBRTC_PREBUILT_ROOT)/metadata.json
else
	WEBRTC_LIB_FILE:=$(WEBRTC_BUILD_ROOT)/out/release-$(TARGET)/obj/libwebrtc.a
endif
export CGO_CXXFLAGS=-I$(PWD)/dep/_google-webrtc/src \
	-I$(PWD)/dep/_google-webrtc/src/third_party/abseil-cpp \
	-I$(MSQUIC_INCLUDE_DIR) \
	-std=c++20 $(DEF_WEBRTC_COND) -DQUIC_API_ENABLE_PREVIEW_FEATURES
export CGO_LDFLAGS= $(DEF_LDFLAGS) \
	-pthread
export CGO_ENABLED=1

.PHONY: all build docker_build_linux_arm64 fmt build_client docker_build_linux_arm64_client gofumpt build_server docker_build_linux_arm64_server golangci-lint check_webrtc_dependencies docker_release_linux_amd64 release clean docker_release_linux_amd64_client release_client compile_webrtc docker_release_linux_amd64_server release_server docker_create_image docker_build_linux_amd64 docker_release_linux_arm64 revive docker_build_linux_amd64_client docker_release_linux_arm64_client test docker_build_linux_amd64_server docker_release_linux_arm64_server set_safe_directories update_submodule build_web_server build_web_client release_web_server release_web_client check_npm front_release duplicate_dist_server clean_duplication_client clean_web clean_dist clean_duplication clean_duplication_server clean_duplication_client check_msquic_dependencies resolve_msquic validate_msquic build_msquic prepare_msquic compile_msquic resolve_webrtc validate_webrtc prepare_webrtc build_webrtc

all: gofumpt golangci-lint test release

prepare: gofumpt golangci-lint test

gofumpt:
	gofumpt --version || go install mvdan.cc/gofumpt@latest
	gofumpt -l -w $(shell find . -name '*.go' | grep -Ev '^\./bufio|^\./client/std|^\./logger/file-rotatelogs|^\./dep')

test: compile_webrtc prepare_msquic
	$(eval CGO_CXXFLAGS+=-O0 -g -ggdb)
	go test -race -cover -count 1 ./...

test_without_static: compile_webrtc prepare_msquic
	$(eval CGO_CXXFLAGS+=-O0 -g -ggdb)
	go test -tags nostatic -race -cover -count 1 ./...

golangci-lint:
	golangci-lint --version || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run \
		--skip-dirs client/std \
		--skip-dirs dep \
		--skip-dirs bufio \
		--skip-dirs logger/file-rotatelogs \
		--skip-dirs build \
		--skip-dirs release \
		--exclude 'SA6002: argument should be pointer-like to avoid allocations' \
		--exclude 'S1000: should use a simple channel send/receive instead of `select` with a single case'

update_submodule: set_safe_directories
	$(UPDATE_SUBMODULE_COMMAND)
	@cd .. && git apply libcs/dep/patch/*.patch && echo "patches applied" || echo "no patch applied"

set_safe_directories:
	@echo "Setting safe directories..."
	@dirs="\
			/go/src/github.com/isrc-cas/gt \
			/go/src/github.com/isrc-cas/gt/dep/_google-webrtc \
			/go/src/github.com/isrc-cas/gt/dep/_msquic \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/clog \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/googletest \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl/boringssl \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3 \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/gost-engine \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl/krb5 \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/gost-engine/libprov \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl/pyca-cryptography \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/krb5 \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl/wycheproof \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/oqs-provider \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/pyca-cryptography \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/python-ecdsa \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/tlsfuzzer \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/tlslite-ng \
			/go/src/github.com/isrc-cas/gt/dep/_msquic/submodules/openssl3/wycheproof ";\
	for dir in $$dirs; do \
		if git config --global --get-regexp '^safe\.directory$$' | grep -q "^safe.directory $$dir$$"; then \
			echo "$$dir is already in safe.directory"; \
		else \
			git config --global --add safe.directory $$dir; \
			echo "Added $$dir to safe.directory"; \
		fi; \
	done

docker_create_image: update_submodule
	docker images | grep -cim1 -E "^gtbuild\s+?v1" || docker build -t gtbuild:v1 .

docker_build_linux_amd64: docker_create_image
	$(eval MAKE_ENV=TARGET=x86_64-linux-gnu GOOS=linux GOARCH=amd64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make build'
docker_release_linux_amd64: docker_create_image
	$(eval MAKE_ENV=TARGET=x86_64-linux-gnu GOOS=linux GOARCH=amd64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make release'
docker_build_linux_amd64_client: docker_create_image
	$(eval MAKE_ENV=TARGET=x86_64-linux-gnu GOOS=linux GOARCH=amd64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make build_client'
docker_release_linux_amd64_client: docker_create_image
	$(eval MAKE_ENV=TARGET=x86_64-linux-gnu GOOS=linux GOARCH=amd64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make release_client'
docker_build_linux_amd64_server: docker_create_image
	$(eval MAKE_ENV=TARGET=x86_64-linux-gnu GOOS=linux GOARCH=amd64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make build_server'
docker_release_linux_amd64_server: docker_create_image
	$(eval MAKE_ENV=TARGET=x86_64-linux-gnu GOOS=linux GOARCH=amd64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make release_server'

docker_build_linux_arm64: docker_create_image
	$(eval MAKE_ENV=TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make build'
docker_release_linux_arm64: docker_create_image
	$(eval MAKE_ENV=TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make release'
docker_build_linux_arm64_client: docker_create_image
	$(eval MAKE_ENV=TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make build_client'
docker_release_linux_arm64_client: docker_create_image
	$(eval MAKE_ENV=TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make release_client'
docker_release_linux_arm64_lib_client: docker_create_image
	$(eval MAKE_ENV=TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make release_lib_client'
docker_build_linux_arm64_server: docker_create_image
	$(eval MAKE_ENV=TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make build_server'
docker_release_linux_arm64_server: docker_create_image
	$(eval MAKE_ENV=TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 STATIC_LINK=$(STATIC_LINK) RACE_CHECK=$(RACE_CHECK) WITH_OFFICIAL_WEBRTC=$(WITH_OFFICIAL_WEBRTC))
	docker run --rm -v $(PWD):/go/src/github.com/isrc-cas/gt -w /go/src/github.com/isrc-cas/gt gtbuild:v1 sh -c '$(MAKE_ENV) make release_server'

build: build_server build_client
release: release_server release_client
build_client: $(SOURCES) Makefile prepare_msquic compile_webrtc compile_p2p
	$(eval CGO_CXXFLAGS+=-O0 -g -ggdb)
	$(eval CGO_LDFLAGS+=$(PWD)/dep/p2p/target/$(RUST_TARGET)/release/libp2p.a)
	$(eval NAME=$(GOOS)-$(GOARCH)-client)
	go build $(DEBUG_OPTIONS) -o build/$(NAME)$(EXE) ./cmd/client
release_client: $(SOURCES) Makefile prepare_msquic compile_webrtc compile_p2p release_web_client
	$(eval CGO_CXXFLAGS+=-O3)
	$(eval CGO_LDFLAGS+=$(PWD)/dep/p2p/target/$(RUST_TARGET)/release/libp2p.a)
	$(eval NAME=$(GOOS)-$(GOARCH)-client)
	go build $(RELEASE_OPTIONS) -o release/$(NAME)$(EXE) ./cmd/client
build_server: $(SOURCES) Makefile prepare_msquic compile_webrtc
	$(eval CGO_CXXFLAGS+=-O0 -g -ggdb)
	$(eval NAME=$(GOOS)-$(GOARCH)-server)
	go build $(DEBUG_OPTIONS) -o build/$(NAME)$(EXE) ./cmd/server
release_server: $(SOURCES) Makefile prepare_msquic compile_webrtc release_web_server
	$(eval CGO_CXXFLAGS+=-O3)
	$(eval NAME=$(GOOS)-$(GOARCH)-server)
	go build $(RELEASE_OPTIONS) -o release/$(NAME)$(EXE) ./cmd/server
build_lib: $(SOURCES) Makefile prepare_msquic compile_webrtc
	mkdir -p release/$(RUST_TARGET)
	cp $(WEBRTC_LIB_FILE) release/$(RUST_TARGET)/
	cp $(MSQUIC_LIB_FILE) release/$(RUST_TARGET)/
	go build $(DEBUG_OPTIONS) -buildmode=c-archive -o release/$(RUST_TARGET)/libcs.a ./lib/export
release_lib: $(SOURCES) Makefile prepare_msquic compile_webrtc release_web_client release_web_server
	mkdir -p release/$(RUST_TARGET)
	cp $(WEBRTC_LIB_FILE) release/$(RUST_TARGET)/
	cp $(MSQUIC_LIB_FILE) release/$(RUST_TARGET)/
	go build $(RELEASE_OPTIONS) -buildmode=c-archive -o release/$(RUST_TARGET)/libcs.a ./lib/export

release_web_server: $(SOURCES_FRONT) Makefile check_npm front_release duplicate_dist_server
release_web_client: $(SOURCES_FRONT) Makefile check_npm front_release duplicate_dist_client

check_npm:
	npm --version || curl -qL https://www.npmjs.com/install.sh | sh

front_build: $(SOURCES_FRONT)
	(cd $(FRONTEND_DIR) && npm install && npm run "build:test")

front_release: $(SOURCES_FRONT)
	(cd $(FRONTEND_DIR) && npm install && npm run "build:pro")

duplicate_dist_server:
	cp -r $(FRONTEND_DIR)/dist $(SERVER_FRONT_DIR)/dist

duplicate_dist_client:
	cp -r $(FRONTEND_DIR)/dist $(CLIENT_FRONT_DIR)/dist

clean: clean_web clean_p2p
	rm -rf build/* release/*
	rm -rf dep/_google-webrtc/src/out/*

clean_web: clean_dist
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -f $(FRONTEND_DIR)/package-lock.json

clean_dist: clean_duplication
	rm -rf $(FRONTEND_DIR)/dist

clean_duplication: clean_duplication_server clean_duplication_client

clean_duplication_server:
	rm -rf $(SERVER_FRONT_DIR)/dist
clean_duplication_client:
	rm -rf $(CLIENT_FRONT_DIR)/dist


check_webrtc_dependencies:
	sh -c "command -v gn"
	sh -c "command -v ninja"
	sh -c "command -v $(CC)"
	sh -c "command -v $(CXX)"

ifeq ($(TARGET_OS), linux)
build_webrtc: check_webrtc_dependencies update_submodule
	cd ./dep/_google-webrtc/src && gn gen out/release-$(TARGET) --args=" \
        clang_use_chrome_plugins=false \
        enable_google_benchmarks=false \
        enable_libaom=false \
        is_clang=false \
        is_component_build=false \
        is_debug=false \
        libyuv_disable_jpeg=true \
        libyuv_include_tests=false \
        libyuv_use_sme=false \
        rtc_build_examples=false \
        rtc_build_tools=false \
        rtc_enable_grpc=false \
        rtc_enable_protobuf=false \
        rtc_include_builtin_audio_codecs=false \
        rtc_include_builtin_video_codecs=false \
        rtc_include_dav1d_in_internal_decoder_factory=false \
        rtc_include_ilbc=false \
        rtc_include_internal_audio_device=false \
        rtc_include_tests=false \
        rtc_use_h264=false \
        rtc_use_x11=false \
        target_cpu=\"$(TARGET_CPU)\" \
        target_os=\"$(TARGET_OS)\" \
        treat_warnings_as_errors=false \
        use_custom_libcxx=false \
        use_custom_libcxx_for_host=false \
        use_gold=false \
        use_lld=false \
        use_rtti=true \
        use_sysroot=false"
	sed -e '/^rule cc$$/{n;s| [^ ]*gcc | $(CC) |g;}' \
		-e '/^rule cxx$$/{n;s| [^ ]*g++ | $(CXX) |g;}' \
		-e '/^rule asm$$/{n;s| [^ ]*gcc | $(CC) |g;}' \
		-e '/^rule solink$$/{n;s| [^ ]*g++ | $(CXX) |g;}' \
		-e '/^rule link$$/{n;s| [^ ]*g++ | $(CXX) |g;}' ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja > ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja.tmp
	cat ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja.tmp > ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja
	rm ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja.tmp
	sed -e 's/ar = ar/ar = $(TARGET)-ar/g' \
		-e 's/ar = "ar"/ar = "$(TARGET)-ar"/g' \
		-e 's/"ar"/$(TARGET)-ar/g' ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja > ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja.tmp
	cat ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja.tmp > ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja
	rm ./dep/_google-webrtc/src/out/release-$(TARGET)/toolchain.ninja.tmp
	ln -sf $$(command -v ar) ./dep/_google-webrtc/src/out/release-$(TARGET)/ar
	ln -sf $$(command -v $(TARGET)-ar || command -v ar) ./dep/_google-webrtc/src/out/release-$(TARGET)/$(TARGET)-ar
	ninja -C ./dep/_google-webrtc/src/out/release-$(TARGET)
endif
ifeq ($(TARGET_OS), mac)
build_webrtc: check_webrtc_dependencies update_submodule
	python3 ./dep/_google-webrtc/src/tools/clang/scripts/update.py
	cd ./dep/_google-webrtc/src && gn gen out/release-$(TARGET) --args=" \
        clang_use_chrome_plugins=false \
        use_xcode_clang=false \
        enable_google_benchmarks=false \
        enable_libaom=false \
        is_component_build=false \
        is_debug=false \
        libyuv_disable_jpeg=true \
        libyuv_include_tests=false \
        rtc_build_examples=false \
        rtc_build_tools=false \
        rtc_enable_grpc=false \
        rtc_enable_protobuf=false \
        rtc_include_builtin_audio_codecs=false \
        rtc_include_builtin_video_codecs=false \
        rtc_include_dav1d_in_internal_decoder_factory=false \
        rtc_include_ilbc=false \
        rtc_include_internal_audio_device=false \
        rtc_include_tests=false \
        rtc_use_h264=false \
        rtc_use_x11=false \
        target_cpu=\"$(TARGET_CPU)\" \
        target_os=\"$(TARGET_OS)\" \
        treat_warnings_as_errors=false \
        use_custom_libcxx=false \
        use_gold=false \
        use_lld=true \
        use_rtti=true \
        use_sysroot=false"
	ninja -C ./dep/_google-webrtc/src/out/release-$(TARGET)
endif

check_msquic_dependencies:
	sh -c "command -v cmake"

resolve_msquic:
ifeq ($(MSQUIC_MODE),prebuilt)
	@echo "Using prebuilt msquic from $(MSQUIC_PREBUILT_ROOT)"
else
	@echo "Using build-mode msquic from $(MSQUIC_BUILD_ROOT)"
endif

validate_msquic:
	test -d "$(MSQUIC_INCLUDE_DIR)"
	test -f "$(MSQUIC_LIB_FILE)"
ifeq ($(MSQUIC_MODE),prebuilt)
	test -f "$(MSQUIC_METADATA_FILE)"
	grep -Eq '"target"[[:space:]]*:[[:space:]]*"$(TARGET)"' "$(MSQUIC_METADATA_FILE)"
	grep -Eq '"library_type"[[:space:]]*:[[:space:]]*"static"' "$(MSQUIC_METADATA_FILE)"
ifeq ($(TARGET_OS),linux)
	grep -Eq '"rename_symbols_applied"[[:space:]]*:[[:space:]]*true' "$(MSQUIC_METADATA_FILE)"
endif
endif

prepare_msquic: resolve_msquic
ifeq ($(MSQUIC_MODE),prebuilt)
	$(MAKE) validate_msquic MSQUIC_MODE=$(MSQUIC_MODE) TARGET=$(TARGET)
else
	$(MAKE) build_msquic MSQUIC_MODE=$(MSQUIC_MODE) TARGET=$(TARGET)
	$(MAKE) validate_msquic MSQUIC_MODE=$(MSQUIC_MODE) TARGET=$(TARGET)
endif

compile_msquic: prepare_msquic

resolve_webrtc:
ifeq ($(WEBRTC_MODE),prebuilt)
	@echo "Using prebuilt webrtc from $(WEBRTC_PREBUILT_ROOT)"
else
	@echo "Using build-mode webrtc from $(WEBRTC_BUILD_ROOT)"
endif

validate_webrtc:
	test -f "$(WEBRTC_LIB_FILE)"
ifeq ($(WEBRTC_MODE),prebuilt)
	test -f "$(WEBRTC_METADATA_FILE)"
	grep -Eq '"target"[[:space:]]*:[[:space:]]*"$(TARGET)"' "$(WEBRTC_METADATA_FILE)"
	grep -Eq '"library_type"[[:space:]]*:[[:space:]]*"static"' "$(WEBRTC_METADATA_FILE)"
endif

prepare_webrtc: resolve_webrtc
ifeq ($(WEBRTC_MODE),prebuilt)
	$(MAKE) validate_webrtc WEBRTC_MODE=$(WEBRTC_MODE) TARGET=$(TARGET)
else
	$(MAKE) build_webrtc WEBRTC_MODE=$(WEBRTC_MODE) TARGET=$(TARGET)
	$(MAKE) validate_webrtc WEBRTC_MODE=$(WEBRTC_MODE) TARGET=$(TARGET)
endif

compile_webrtc: prepare_webrtc

ifeq ($(TARGET_OS), linux)
build_msquic: check_msquic_dependencies update_submodule
	mkdir -p ./dep/_msquic/out/$(TARGET)
	cmake -B./dep/_msquic/out/$(TARGET) -S./dep/_msquic -DQUIC_BUILD_SHARED=OFF -DCMAKE_TARGET_ARCHITECTURE=$(TARGET_CPU)
	make -C./dep/_msquic/out/$(TARGET) -j$(shell nproc)
	@renameSymbols=$$(objdump -t ./dep/_msquic/out/$(TARGET)/bin/Release/libmsquic.a | awk -v RS= '/_YB80VJ/{next}1' | grep -E 'g +(F|O) ' | grep -Evi ' (ms){0,1}quic' | awk '{print " --redefine-sym " $$NF "=" $$NF "_YB80VJ"}') && \
		$(TARGET)-objcopy $$renameSymbols ./dep/_msquic/out/$(TARGET)/bin/Release/libmsquic.a
endif
ifeq ($(TARGET_OS), mac)
build_msquic: check_msquic_dependencies update_submodule
	mkdir -p ./dep/_msquic/out/$(TARGET)
	cmake -B./dep/_msquic/out/$(TARGET) -S./dep/_msquic -DQUIC_BUILD_SHARED=OFF -DCMAKE_TARGET_ARCHITECTURE=$(TARGET_CPU)
	make -C./dep/_msquic/out/$(TARGET) -j$(shell sysctl -n hw.physicalcpu)
	renameSymbols=$$(cat ./dep/patch/redefine-sym.mac) && \
		$(OBJCOPY) $$renameSymbols ./dep/_msquic/out/$(TARGET)/bin/Release/libmsquic.a
endif

compile_p2p:
	cd ./dep/p2p && \
    . "$$HOME/.cargo/env" && \
	cargo build -r --target $(RUST_TARGET)

clean_p2p:
	cd ./dep/p2p && \
    . "$$HOME/.cargo/env" && \
	cargo clean
