ifeq ($(OS),Windows_NT)
    detected_os := Windows
else
    detected_os := $(shell sh -c 'uname 2>/dev/null || echo Unknown')
endif

ifeq ($(detected_os),Windows)
    make_param := windows
endif
ifeq ($(detected_os),Darwin)
    make_param := darwin
endif
ifeq ($(detected_os),Linux)
    make_param := linux
endif

all:
	make $(make_param)

linux:
	cd ./core && TARGET=aarch64-linux-gnu GOOS=linux GOARCH=arm64 make release_lib
	cargo build --target aarch64-unknown-linux-gnu -r
	cd ./core && TARGET=x86_64-linux-gnu GOOS=linux GOARCH=amd64 make release_lib
	cargo build --target x86_64-unknown-linux-gnu -r
	cd ./core && TARGET=riscv64-linux-gnu GOOS=linux GOARCH=riscv64 make release_lib
	cargo build --target riscv64gc-unknown-linux-gnu -r
	mkdir -p release
	cp target/x86_64-unknown-linux-gnu/release/gt release/gt-linux-x86_64
	cp target/aarch64-unknown-linux-gnu/release/gt release/gt-linux-aarch64
	cp target/riscv64gc-unknown-linux-gnu/release/gt release/gt-linux-riscv64

darwin:
	cd ./core && TARGET=x86_64-apple-darwin GOOS=darwin GOARCH=amd64 arch -arch x86_64 make release_lib
	cargo build --target x86_64-apple-darwin -r
	cd ./core &&  TARGET=aarch64-apple-darwin GOOS=darwin GOARCH=arm64 arch -arch arm64 make release_lib
	cargo build --target aarch64-apple-darwin -r
	mkdir -p release
	cp target/x86_64-apple-darwin/release/gt release/gt-macos-x86_64
	cp target/aarch64-apple-darwin/release/gt release/gt-macos-aarch64

release_target:
	cd ./core && TARGET=$(TARGET) GOOS=$(GOOS) GOARCH=$(GOARCH) RUST_TARGET=$(RUST_TARGET) MSQUIC_MODE=$(MSQUIC_MODE) WEBRTC_MODE=$(WEBRTC_MODE) make release_lib
	cargo build --target $(RUST_TARGET) -r
	mkdir -p release
ifeq ($(GOOS),windows)
	cp target/$(RUST_TARGET)/release/gt.exe release/gt-$(BINARY_SUFFIX).exe
else
	cp target/$(RUST_TARGET)/release/gt release/gt-$(BINARY_SUFFIX)
endif

