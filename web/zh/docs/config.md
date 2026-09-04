# 配置参考

`gt` 从两处读取配置,后者覆盖前者:

1. YAML 配置文件——`-c ./config.yaml`,或 Web 管理台写入的默认路径
2. 命令行参数

`-c` 指向**目录**可批量启动多个角色——每个文件需用顶层 `type: client`、`type: server` 声明角色(含 `services:` 键的文件视为客户端):

```bash
gt -c ./conf.d
```

对运行中的实例热更新配置,不断连接:

```bash
gt -s reload    # 另有:restart、stop
```

reload 重新读取配置,工作进程切换期间转发不断流。同一机制让二进制升级也能在带载下零停机完成。

## 客户端选项

```yaml
# client.yaml
id: myapp
secret: s3cret
remote: tcp://relay.example.com:7000

services:
  - hostPrefix: web
    local: http://127.0.0.1:8080
```

| 选项(yaml / 命令行) | 默认值 | 说明 |
|---|---|---|
| `id` / `-id` | — | 客户端 id,兼任默认主机名前缀。长度 1–200 |
| `secret` / `-secret` | — | 与 id 配对的密钥。长度 1–200 |
| `remote` / `-remote` | `tcp://` | 中转地址,可重复。协议:`tcp://`、`tls://`、`quic://`。给出多个时客户端探测网络后自动选择可用协议 |
| `reconnectDelay` | `5s` | 重连间隔 |
| `remoteConnections` | `3`(1–10) | 中转连接池上限 |
| `remoteIdleConnections` | `1` | 保持预热的空闲连接数 |
| `remoteTimeout` | `45s` | 中转连接超时 |
| `remoteSTUN` / `-remoteSTUN` | — | WebRTC P2P 使用的 STUN 服务器,如 `stun:1.2.3.4:3478` |
| `remoteCert` / `-remoteCert` | — | 中转证书路径(固定证书校验) |
| `remoteCertInsecure` | `false` | 接受中转自签证书 |
| `webAddr` / `-webAddr` | 关闭 | Web 管理台地址,如 `127.0.0.1:7000` |
| `logLevel` / `-logLevel` | `info` | `trace`…`panic`,或 `disable` |
| `logFile` + `logFileMaxSize`(512MB)+ `logFileMaxCount`(7) | 关闭 | 滚动文件日志 |
| `bbr` | `false` | QUIC 使用 BBR 拥塞控制(两端都开) |

`services:` 下每项的字段(命令行对应 `-hostPrefix`、`-local`、`-remoteTCPPort`、`-remoteTCPRandom`、`-useLocalAsHTTPHost`、`-localTimeout`):

| 字段 | 说明 |
|---|---|
| `hostPrefix` | 该服务应答的主机名前缀 |
| `local` | 本地服务 URL:`http://`、`https://` 或 `tcp://host:port` |
| `remoteTCPPort` | 中转为该服务开启的固定 TCP 端口 |
| `remoteTCPRandom` | 由中转从其配置的端口池自动分配 |
| `localTimeout` | 本地连接超时 |
| `useLocalAsHTTPHost` | 向后端传递本地地址作为 HTTP `Host` |

WebRTC/P2P 相关:`webrtcConnections`(P2P 连接上限,默认 30)、`webrtcConnectionIdleTimeout`(5m)、`webrtcMinPort`/`webrtcMaxPort`(UDP 端口范围)、`webrtcThreadMode`。TCP 转发(网关)相关:`tcpForwardAddr`、`tcpForwardHostPrefix`、`tcpForwardConnections`。

## 服务端选项

```yaml
# server.yaml
addr: "7000"
webAddr: "127.0.0.1:8000"
```

| 选项(yaml) | 默认值 | 说明 |
|---|---|---|
| `addr` / `-addr` | — | 中转监听,如 `80`、`:80`、`0.0.0.0:80` |
| `tlsAddr` + `certFile` + `keyFile` | 关闭 | TLS 监听,供 `tls://` 客户端(`tlsVersion` 最低 `tls1.2`) |
| `quicAddr` + `certFile` + `keyFile` | 关闭 | QUIC 监听,供 `quic://` 客户端 |
| `stunAddr` / `-stunAddr` | 关闭 | STUN 监听(UDP 3478),供 WebRTC P2P |
| `sniAddr` | 关闭 | 按 SNI 路由的原始 TLS 转发——端到端加密直通 |
| `apiAddr` | 关闭 | 内部 API 监听 |
| `webAddr` / `-webAddr` | 关闭 | Web 管理台,如 `0.0.0.0:8000`(可加 `admin`/`password`/`signingKey` 认证) |
| `users:` 或重复 `-id`/`-secret` | — | 客户端凭证;见[多用户实例](examples.md#多用户中转与配额) |
| `users[].tcp[].range` | — | 用户可开的端口池,如 `30000-39999` |
| `users[].tcpNumber` | — | 每用户最多可开端口数 |
| `users[].speed` | 不限 | 每用户限速,字节/秒 |
| `users[].connections` | `10` | 每用户最大隧道连接数 |
| `host.number` | `0` | 每用户最多主机前缀服务数 |
| `host.regex` | — | 允许的前缀规则 |
| `host.withID` | `false` | 前缀带命名空间:`id-host` |
| `timeout` | `90s` | 空闲连接超时(`timeoutOnUnidirectionalTraffic` 更严格) |
| `httpMUXHeader` | `Host` | 主机名前缀路由所用的请求头 |
| `allowAnyClient` | `false` | 接受任意 id/secret——**公网切勿开启** |
| `bbr` | `false` | QUIC 使用 BBR 拥塞控制 |
| `logLevel`、`logFile`、`logFileMaxSize`、`logFileMaxCount` | `info` | 日志,同客户端 |

`authAPI` / `usageAPI` 允许外部服务校验凭证并统计每用户流量——托管控制台正是这样集成的。
