# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v2.3.0] - 2026-05-28

### 🚀 Architectural Breakthrough: Monorepo Transition
* **Physical Directory Restructuring**: Transitioned the repository into a clean, highly cohesive **Monorepo** structure. Reorganized files into logical sub-modules:
  - `/core` (Go Kernel, renamed from `libcs/`): Handles core protocol engines, tunnel routing, and CGO bindings.
  - `/cli` (Rust CLI, renamed from `bin/`): Zero-dependency high-performance CLI shell statically linking the core.
  - `/desktop` (Wails GUI): Native Wails desktop graphical application to control local Client tunnels.
  - `/admin` (Web Dashboard, moved from `core/web/front/` to workspace root): Cloud Server web management administration dashboard.
* **Workspace Bridging**: Integrated Go Workspaces (`go.work`) and Cargo Workspaces, enabling zero-friction cross-module local dependency resolution and live-reload compilation.
* **Client/Server Management Separation**: Separated management domains completely. `gt-admin` (TS/Vue 3 UI) is now embedded exclusively in the `gt-server` backend binary, keeping the `gt-client` executable ultra-lightweight, while client management is fully delegated to `gt-desktop`.

### 🌟 New Features & Enhancements
* **Wails v3 Desktop Client**: Initialized a cross-platform desktop GUI application using **Wails v3** with support for system trays, native frames, and multi-OS builds.
* **Server Quota TTL Cache**: Added a high-performance in-memory TTL cache for AuthAPI quota responses, reducing external auth load.
* **Quota Pre-Validation**: Integrated quota validation directly into `handleTunnel` inside the server to abort unauthorized connections early.

### ⚡ CI/CD & Build Infrastructure
* **Parallel Decoupled Matrix Pipelines**: Rewrote GitHub Actions to run platform compilations (`x86_64`, `aarch64`, `riscv64`) fully in parallel, preventing monolithic build blockages.
* **Self-Healing Prebuild Caching**: Implemented automatic prebuild dependency validation. The CI probes for precompiled `msquic` and `webrtc` archives from releases; if missing (cache miss), it compiles, packages, and automatically uploads them back to the release for future instant runs.
* **Decoupled Containerization Flow**: Decoupled binary compilation from Docker building. Multi-arch binaries are uploaded as pipeline artifacts and compiled into multi-platform server/client Docker images in a standalone, parallel GHCR packaging job.
* **C++20 CGO Standard Upgrade**: Upgraded CGO compiler flags from C++17 to C++20 to align with modern WebRTC specifications.

### 🔧 Compatibility & Compilations Fixes
* **GCC 13 Cross-Compilation Support**:
  - Automatically disabled SVE/SVE2 compilation in `libaom` and `libvpx` under GCC target limitations.
  - Patched std::pair comparison heterogeneous typing error in GCC 13.
  - Disabled ARM64 SME instructions assembly inside `libyuv` GN parameters for GCC toolchains.
* **Toolchain Ninja Repair**: Refined `toolchain.ninja` compiler and `ar` linkage replacement strategies using POSIX boundaries to bypass cross-compilation errors.
* **Windows Build Stabilization**:
  - Solved Windows shell double-quote stripping bugs in `gclient config` inside PowerShell.
  - Manually simulated `LASTCHANGE` and `LASTCHANGE.committime` file generation to bypass timestamp calculation script failures in Windows build host.
* **CGO Compilation Headers Sync**: Configured unconditional WebRTC checkout in CI pipelines, ensuring CGO compilation headers (like Abseil functional wrappers) are always present for GCC/G++ linkage during static-library builds.

---

[v2.3.0]: https://github.com/isrc-cas/gt/compare/244454f...v2.3.0
