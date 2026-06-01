# Windows MSVC Rust Link Failure With Go c-archive

## 背景

GT 的 Windows release 构建链路包含三段产物：

- `core/release/windows/gt.lib`：由 Go `go build -buildmode=c-archive` 生成。
- `core/release/windows/msquic.lib`：预编译 MsQuic 静态库。
- `core/release/windows/webrtc.lib`：预编译 WebRTC 静态库。

最终 `cargo build --target x86_64-pc-windows-msvc -r` 会通过 MSVC `link.exe` 链接 Rust CLI 和这些原生静态库。

## 现象演进

最早的失败表现为：

```text
libgt-*.rlib(...o) : error LNK2019: unresolved external symbol MsQuicOpenVersion referenced in function Init
libgt-*.rlib(...o) : error LNK2019: unresolved external symbol MsQuicClose referenced in function Init
fatal error LNK1120: 2 unresolved externals
```

这看起来像 `msquic.lib` 没有被链接，或者预编译 MsQuic 产物不完整。但后续日志证明，Rust 链接命令已经带上了：

```text
-C link-arg=core/release/windows/gt.lib
-C link-arg=core/release/windows/webrtc.lib
-C link-arg=core/release/windows/msquic.lib
```

后续进一步暴露出真实错误：

```text
gt.lib(go.o) : fatal error LNK1223: invalid or corrupt file: file contains invalid .pdata contributions
```

这说明链接器不是单纯找不到 MsQuic，而是在处理 Go c-archive 里的 `go.o` 时提前失败，导致后续外部符号解析表现被污染。

## 误判和排除项

排查过程中重点排除了这些方向：

- 不是单纯缺少 `msquic.lib`：链接参数里已经包含 `msquic.lib`。
- 不是只靠重新编译 MsQuic 能解决：重新生成的 Windows static MsQuic 产物大小和结构没有本质变化。
- 不是 WebRTC 同步问题本身：WebRTC 头文件同步问题会导致 Go CGO 编译失败，但与 `gt.lib(go.o)` 的 LNK1223 是不同阶段的问题。
- 不是 Rust crate 依赖问题：失败点在 MSVC link 处理原生 COFF archive。

## 真实原因

MSVC `link.exe` 要求 COFF `.pdata` section 里的 `RUNTIME_FUNCTION` 记录按函数起始地址排序。Go 在 Windows 上生成 `-buildmode=c-archive` 时，会把运行时对象打入 archive，其中 `go.o` 的 `.pdata` 记录可能不是 MSVC 期望的顺序。

直接按 `.pdata` 的 12 字节原始内容排序不可靠，因为 Go object 中 `.pdata` 的函数地址字段主要依赖 relocation。原始 DWORD 在文件里可能都是占位值，真正的排序依据应该来自每条 `.pdata` entry 的第一个 relocation 指向的 COFF symbol。

因此，关键不是 “MsQuic 缺两个符号”，而是：

```text
Rust/MSVC link -> 读取 gt.lib -> 读取 go.o -> 校验 .pdata -> LNK1223 -> 链接流程中断
```

## 最终修复

新增脚本：

```text
core/scripts/sort_go_pdata.py
```

脚本做的事情：

- 解析 MSVC COFF archive。
- 找到 archive member `go.o`。
- 解析 `go.o` 的 COFF section table、symbol table 和 string table。
- 找到所有 `.pdata` section。
- 按每个 `.pdata` entry 的 begin-address relocation 指向的 symbol 排序。
- 同步重写 `.pdata` relocation 的 `VirtualAddress`，保证 relocation 仍然指向移动后的 entry。
- 对无法用 symbol 判断的 entry 回退到原始 bytes 排序。

`core/Makefile` 在 Windows `go build -buildmode=c-archive` 之后调用该脚本：

```makefile
python scripts/sort_go_pdata.py release/$(RUST_TARGET_DIR)/$(GO_OUT_FILE_NAME)
```

该调用已覆盖 `build_lib` 和 `release_lib` 两条路径。

## 诊断日志

为避免后续再次盲猜，脚本现在会打印 `.pdata` 修复审计日志，例如：

```text
[sort_go_pdata] .pdata: entries=... relocations=... symbol_keyed=... raw_fallback=...
[sort_go_pdata] .pdata: before_first=... before_last=...
[sort_go_pdata] .pdata: after_first=... after_last=...
[sort_go_pdata] .pdata: moved_entries=... order_changed=...
[sort_go_pdata] .pdata: move old=... new=... key=... source=symbol#...
patched release/windows/gt.lib: sorted 1 .pdata section(s) in go.o
```

判断修复是否生效时，应重点看：

- `Build GT core library (Windows)` 阶段是否运行了 `python scripts/sort_go_pdata.py release/windows/gt.lib`。
- 日志里是否出现 `patched ... sorted ... .pdata section(s) in go.o`。
- `Compile GT single target (Windows)` 阶段是否不再出现 `gt.lib(go.o) : fatal error LNK1223`。

## 维护注意事项

- 如果未来 Go 修复了 Windows c-archive `.pdata` 排序问题，脚本可能变成 no-op，但保留日志仍有价值。
- 如果 Go 或 MSVC 改变 COFF object/relocation 结构，应优先检查脚本的 symbol-keyed 排序日志，而不是先怀疑 MsQuic。
- 如果重新出现 `MsQuicOpenVersion` / `MsQuicClose` unresolved，需要先确认是否同时存在 LNK1223；只有没有 LNK1223 时，才回到 MsQuic export/import 和 link order 方向排查。

参考：

- [Microsoft LNK1223 documentation](https://learn.microsoft.com/cpp/error-messages/tool-errors/linker-tools-error-lnk1223)
- [Go change discussion about sorting .pdata for c-archive](https://groups.google.com/g/golang-codereviews/c/53W_efGksj0)
