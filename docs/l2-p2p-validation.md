# P2P L2 跨网络真打洞验证手册

验证 tcpForward 访客网关模式在**跨 NAT 真实网络**(非 127.0.0.1)下的打洞
能力。CI 全部走 host 候选(同一回环),本手册补齐 srflx 候选与真实 NAT 场景。

## 拓扑

```
                       公网 / VPS
                    ┌─────────────┐
                    │  gt server  │  ← 只做信令与中转兜底,数据不走这里
                    │  + STUN     │     (stunAddr 提供公网 STUN)
                    └──────┬──────┘
             ┌─────────────┴─────────────┐
        NAT-A(家宽/办公网)          NAT-B(另一运营商/热点)
             │                            │
      ┌──────┴──────┐              ┌──────┴──────┐
      │ provider    │              │ gateway     │
      │ gt client   │              │ gt client   │
      │ (id=prov)   │              │ (tcpForward)│
      │ 本地服务:8080│              │ 对外监听:8888│
      └─────────────┘              └─────────────┘
```

- server:任一有公网 IP 的 VPS(云主机即可)。
- provider 与 gateway 必须在**不同的 NAT 后面**(例如一台家宽、一台手机热点;
  同一内网两台机器会退化成 host 候选直连,验证不到打洞)。
- 若无第二公网 STUN,直接用 server 的 `-stunAddr`;要测公网 STUN 也可用
  `stun:stun.l.google.com:19302`。

## 角色命令

三个角色各执行一条命令(替换 `<server_ip>`、`<stun_ip>`):

**server(公网 VPS):**

```bash
gt server -addr 0.0.0.0:7000 -stunAddr 0.0.0.0:3478 \
  -id prov -secret prov-secret
```

**provider(NAT-A 后,承载真实服务,如一个 HTTP 文件服务):**

```bash
# 先起一个本地服务,例如:
python3 -m http.server 8080

gt client -id prov -secret prov-secret \
  -remote <server_ip>:7000 \
  -remoteSTUN stun:<server_ip>:3478 \
  -local http://127.0.0.1:8080 \
  -webrtcThreadMode \
  -logLevel debug
```

**gateway(NAT-B 后,把公网入口映射到 provider):**

```bash
gt client -id gw -secret gw-secret \
  -remote <server_ip>:7000 \
  -remoteSTUN stun:<server_ip>:3478 \
  -tcpForwardAddr 0.0.0.0:8888 \
  -tcpForwardHostPrefix prov \
  -logLevel debug
```

> 注:provider 必须带 `-webrtcThreadMode`(Go 线程模式)。不带该开关走
> proc 模式,PC 运行在 sub-p2p 子进程(Rust)中,重协商已支持(2026-09-03),
> 但 L2 首验建议先用 thread 模式排除变量,proc 模式作为第二轮复验。

## 验证步骤

1. **第三个网络**(手机热点等)或直接在 gateway 机器上发起访问:

   ```bash
   curl -v http://127.0.0.1:8888/
   ```

   期望:HTTP 200 且返回 provider 本地服务的内容。

2. **观察打洞路径**。两台客户端均开 `-logLevel debug` 后,grep 客户端日志:

   ```bash
   grep -E "srflx|typ host|candidate pair|Connected|Completed" client.log
   ```

   - `typ srflx` 候选出现 = STUN 打洞已生效(获取到公网映射地址);
   - 选中候选对(Pair)两端均为 `srflx` = 真·跨 NAT 打洞成功;
   - 若只看到 `relay`/中转字样或根本无 srflx,说明走了回退(见下)。

3. **数据面佐证**。访问多次后看通道收发统计(日志 `close data channel` /
   `data channel on open` 行的 `bytesSent/bytesReceived`),非零即数据经
   P2P 通道流动;同时 provider 侧应看到 `p2p http task started`。

## 判定矩阵

| NAT 组合                     | 预期                                   |
|------------------------------|----------------------------------------|
| 全锥型 × 全锥型(多数家宽)   | srflx 直连成功                         |
| 受限锥型 × 受限锥型          | 成功(先 outbound 打洞后互发)          |
| 任一端对称 NAT               | **预计失败**——无 TURN,无法回退中转,  |
|                              | 属已知限制(decisions.md §12)          |
| 任一端为公网 IP(DMZ/云主机) | 成功(等价直连)                        |

## 已知限制与注意

- 无 TURN:对称 NAT 打不通是预期行为,不要当 bug 报。
- 双端务必用**同一构建版本**(重协商协议 2026-09-03 起才有,旧二进制互连
  会因无重协商而通道打不开)。
- 若观察srflx 候选已交换但通道仍不 open,抓 `-webrtcLogLevel verbose`
  (临时)看 DCEP 与 SCTP 状态,并对照 CI 中 TestTCPForward 的行为差异。
- Windows 客户端注意防火墙放行 gt 进程的 UDP 出入站。
