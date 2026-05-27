# GT Desktop Runtime Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Wails desktop app stop embedding the GT client runtime directly and instead control a local worker subprocess through a stable desktop-side runtime layer.

**Architecture:** The Wails process keeps UI, config persistence, and process orchestration. Instead of embedding the GT client runtime, it launches the existing `gt` binary in client mode with a generated runtime config that enables the built-in localhost web/API control plane. Frontend-facing Wails methods stay stable so the UI can continue using `LoadConfig`, `SaveConfig`, `StartTunnel`, `StopTunnel`, `GetStatus`, and log events without a rewrite.

**Tech Stack:** Go, Wails v3, net/http, os/exec, YAML, localhost control API, existing GT client runtime.

---

## File Map

- Modify: `D:\Work\tools\gt\desktop\app.go`
  - Replace direct `client.Client` ownership with a desktop runtime abstraction.
- Modify: `D:\Work\tools\gt\desktop\config.go`
  - Replace `client.Config` dependency with a desktop-local YAML model compatible with GT config files.
- Modify: `D:\Work\tools\gt\desktop\logger.go`
  - Keep UI log buffering, add a way for the worker runtime to append plain log lines from subprocess pipes.
- Modify: `D:\Work\tools\gt\desktop\main.go`
  - Wire the new runtime implementation into `GTApp`.
- Create: `D:\Work\tools\gt\desktop\model.go`
  - Hold the desktop-local config and service structs used by Wails bindings and YAML persistence.
- Create: `D:\Work\tools\gt\desktop\runtime.go`
  - Define the runtime interface and shared status/result structs.
- Create: `D:\Work\tools\gt\desktop\process_runtime.go`
  - Locate and launch the existing `gt` binary, generate runtime config overlays, collect stdout/stderr logs, and call the built-in localhost API.
- Modify: `D:\Work\tools\gt\desktop\frontend\bindings\...`
  - Regenerate Wails bindings after desktop-side config types move out of `github.com/isrc-cas/gt/client`.

## Phase Scope

This implementation intentionally stops at the first stable architecture cut:

- Wails no longer imports `github.com/isrc-cas/gt/client`.
- Desktop config read/write stays local and compatible with GT YAML.
- Start/stop/status go through a worker subprocess.
- Worker logs stream back into the existing Wails event bus.

Deferred to a later phase:

- Packaging the `gt` binary alongside desktop release artifacts in the Windows/macOS/Linux taskfiles.
- Replacing localhost HTTP with named pipes or Unix sockets.
- Adding a desktop-specific typed control API instead of reusing the current GT web admin endpoints.

## Tasks

### Task 1: Introduce desktop-local config and runtime interfaces

**Files:**
- Create: `D:\Work\tools\gt\desktop\model.go`
- Create: `D:\Work\tools\gt\desktop\runtime.go`
- Modify: `D:\Work\tools\gt\desktop\config.go`

- [ ] Define desktop-local config structs that mirror the YAML fields the current UI edits.
- [ ] Update config persistence to use those desktop-local structs only.
- [ ] Add a runtime interface for start, stop, status, and cleanup.

### Task 2: Move heavy runtime into the existing GT binary subprocess

**Files:**
- Create: `D:\Work\tools\gt\desktop\process_runtime.go`

- [ ] Implement a subprocess runtime that starts `gt client --config <generated-temp-config>`.
- [ ] Reuse the built-in localhost `/api/login`, `/api/health`, `/api/config/running`, and `/api/server/stop` endpoints for orchestration.
- [ ] Capture subprocess stdout/stderr in the desktop process and feed them into the existing log writer.

### Task 3: Rewire the Wails service without changing the frontend contract

**Files:**
- Modify: `D:\Work\tools\gt\desktop\app.go`
- Modify: `D:\Work\tools\gt\desktop\main.go`
- Modify: `D:\Work\tools\gt\desktop\logger.go`

- [ ] Replace direct `client.Client` ownership in `GTApp` with the new runtime abstraction.
- [ ] Keep public Wails methods and return shapes stable so `frontend/main.js` does not need a structural rewrite.
- [ ] Ensure shutdown and repeated start/stop are serialized.

### Task 4: Refresh bindings and verify the new split

**Files:**
- Modify: `D:\Work\tools\gt\desktop\frontend\bindings\...`

- [ ] Regenerate Wails bindings after moving config/status types.
- [ ] Build the worker package directly.
- [ ] Build the desktop root package directly.
- [ ] Attempt `wails3 build DEV=true` or `wails3 dev` and record any remaining blockers outside this refactor.
