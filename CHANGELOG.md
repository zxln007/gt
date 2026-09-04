# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v2.4.1] - 2026-09-04

### 🐛 P2P 数据面修复战役 —— 跨 NAT tcpForward 恢复可用
* **WebRTC 重新协商支持**（libwebrtc 2025 适配）：新版 libwebrtc 移除了 in-band 通道的隐式流建立，连接后创建的数据通道不再自动打开。tcpForward 网关现对每条 TCP 连接发起重新协商（复用 XP 信令通路 + `WebRTC-OP-ID` 路由到既有 peer task），provider 在既有 PC 上应用连续 offer；通道数不再有上限，替代临时性的预创建通道方案。
* **proc 模式（sub-p2p 子进程）同步打通**：Go 侧将连续 offer 经 stdin/stdout op 通路转发给子进程；Rust 侧移除 config 回显（回显占据 stdout 首位导致首轮握手读错消息）。

### 🔧 包装层深层修复
* **nil network/worker 线程兜底**：2025 版 libwebrtc 将数据通道传输层任务（OnTransportReady、DCEP）投递到 network 线程，线程为 null 时任务永久丢失——PC 信令正常、ICE 可通但通道永不 open。
* **RegisterObserver 异步注册吞 open**：观察者注册被异步投递且不补发状态，本地创建通道在 transport 就绪时立即 kOpen 但回调丢失；C++ 侧注册后检测补发。
* **多 tunnel 信令撞号**：各 tunnel 的 taskIDSeed 独立计数，撞号时旧逻辑会杀掉活着的 peer task 顶替，重协商被误判为首轮协商；改为撞号不杀不占位 + 查找优先路由。
* **信令分帧统一**：所有信令响应消息（SDP/candidate/peer id）统一 2 字节大端长度前缀，消除裸 `{"id":N}` 与 candidate JSON 的解析撞形。

### ✅ 质量基建
* **test.yml 放宽到全量 core 测试**（27 个，零 skip）：P2P 重协商全链路（TestTCPForward/TestP2PSetOffer）、中转、TLS/SNI、QUIC、鉴权等全部进入 CI。
* 修复 3 个从未在 CI 跑过的遗留测试：TestSNI 摆脱外网依赖（本地自签 TLS 回声）、TestTCPNumberAndTCPRange 端口避开临时端口区与全局 range 校验、TestReconnectLimit 改确定性限流断言。
* 消除并行负载下的挂死路径（t.Fatal Goexit × 隧道写阻塞），PC 级等待补超时。
* 新增 L2 跨 NAT 真打洞验证手册（`docs/l2-p2p-validation.md`）。

---

## [v2.4.0] - 2026-08-30

### 🚀 SaaS Control Plane Integration
* **authAPI Quota Contract**: relay now resolves per-user quotas (host/TCP numbers, speed, connections, monthly traffic) from the control plane on every tunnel authentication, with a TTL cache.
* **Per-User Traffic Metering**: hot-path atomic byte counters (upload/download) on all data planes (HTTP/TCP/SNI), accumulated server-side across sessions and flushed to the `usageAPI` every 30s with retry-on-failure and shutdown flush.
* **Monthly Traffic Enforcement**: the control plane rejects authentication (`result:false`) once a user's month-to-date traffic reaches the plan limit; tunnels are cut within the auth cache TTL.

### 🌟 Features
* **httpMUXHeader now configurable** (`NETWORK_HTTPMUXHEADER`, defaults to `Host`) — browser-friendly host routing out of the box.
* **Third-level tunnel domains**: users get `prefix.app.gtunnel.dev` addresses; relay host parsing takes the first label of any depth suffix.
* **Unified CI pipeline**: container + release workflows merged into a single `build.yml` — one 5-platform build feeds both Docker images and GitHub release binaries; rc/alpha/beta tags publish as prereleases without stealing `latest`.
* **Docker image semver tags**: images are tagged `server-<version>` / `client-<version>` on releases, with `latest` pointing at the newest stable tag (not the dev branch).

### 🏗️ Architecture
* **Desktop client extracted** to a dedicated repository (`gt-desktop`) to decouple SaaS account integration and release cadence from the data plane.

### 🐛 Fixes
* Entrypoint no longer hardcodes `httpMUXHeader: EID` (browsers got EOF on tunnel domains).
* Let's Encrypt wildcard cert (`*.app.gtunnel.dev` + `relay.gtunnel.dev` SAN) mounted for relay TLS.
* Release workflow resolves WebRTC/abseil headers via shallow `gclient sync` (prebuilt archives ship only libs).
* macOS Intel (macos-13) runner retired from the matrix; Intel Macs run the aarch64 build under Rosetta 2.

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
