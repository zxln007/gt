# 使用实例

常见场景的现成配方。所有示例假定:

- 中转地址为 `relay.example.com`——可以是自建服务器,也可以是托管中转(准确地址从 [console.gtunnel.dev](https://console.gtunnel.dev) 获取)
- 客户端凭证由中转运营方发放(托管中转在控制台一键签发)

请替换成你自己的值。完整选项见[配置参考](config.md)。

## 把本地 Web 应用发布到中转域名下

最经典的反向隧道:NAT 后机器上的 Web 服务,通过中转获得一个可访问的 URL。

```bash
gt client -local http://127.0.0.1:3000 \
  -remote tcp://relay.example.com:7000 \
  -id myapp -secret s3cret
```

访问者打开 **http://myapp.relay.example.com:7000**——客户端的 `id`(`myapp`)就是服务端路由用的主机名前缀。把 `*.relay.example.com` 解析到中转 IP,即可支持任意多个客户端发布。

!> 服务端只检查第一个包的 `Host` 头来选定转发目标,后续数据原样直通。这是中转有意为之的隐私/性能设计。

前缀想和 `id` 不一样?加 `-hostPrefix staging`,或者从一个客户端发布多个名字——见[一个客户端发布多个服务](#一个客户端发布多个服务)。

## 访问 NAT 后面的 SSH

把本地 SSH 服务发布到中转的 TCP 端口,任何地方都能 `ssh` 回家:

```bash
gt client -local tcp://127.0.0.1:22 \
  -remote tcp://relay.example.com:7000 \
  -id home -secret s3cret \
  -remoteTCPPort 6022
```

在任意机器上:

```bash
ssh -p 6022 user@relay.example.com
```

适用于一切 TCP 服务——把 `tcp://127.0.0.1:22` 换成 `tcp://127.0.0.1:445`(SMB)、`tcp://127.0.0.1:5432`(PostgreSQL)、`tcp://127.0.0.1:25565`(Minecraft)等即可。

**随机端口变体。** 无法预先固定端口时,让服务端从端口池里自动分配:用 `-remoteTCPRandom` 代替 `-remoteTCPPort`。服务端需配置端口范围(如 `-tcpRange 30000-39999`),客户端启动日志会打印分配到的端口。

## 一个客户端发布多个服务

一个客户端进程可通过配置文件同时发布多个服务:

```yaml
# client.yaml
id: home
secret: s3cret
remote: tcp://relay.example.com:7000

services:
  - hostPrefix: web          # -> http://web.relay.example.com:7000
    local: http://127.0.0.1:8080
  - hostPrefix: nas          # -> http://nas.relay.example.com:7000
    local: http://127.0.0.1:5000
  - local: tcp://127.0.0.1:22   # -> 中转随机 TCP 端口(见日志)
    remoteTCPRandom: true
```

```bash
gt client -c ./client.yaml
```

`services` 的每一项都可设置 `hostPrefix`、`local`、`remoteTCPPort`、`remoteTCPRandom`、`localTimeout`、`useLocalAsHTTPHost`(把本地地址作为 HTTP `Host` 传给后端——后端按虚拟主机区分站点时有用)。

?> 共享中转通常会开启 `host.withID: true`,前缀会带命名空间变成 `id-host`,如 `home-web.relay.example.com`。以中转运营方的说明为准。

## 客户端与服务端之间走 QUIC

弱网/高延迟链路(移动网络、跨运营商 VPS 互联)下,客户端↔中转这一段可以走 QUIC。服务端需要为该监听准备 TLS 证书:

```bash
gt server -addr 7000 -quicAddr 443 \
  -certFile /etc/gt/tls.crt -keyFile /etc/gt/tls.key \
  -id id1 -secret secret1
```

```bash
gt client -local http://127.0.0.1:8080 \
  -remote quic://relay.example.com:443 \
  -remoteCertInsecure \
  -id id1 -secret secret1
```

`-remoteCertInsecure` 表示接受服务端自签证书;生产环境建议去掉它,改用 `-remoteCert /path/to/tls.crt` 固定证书。

wrk 压测中 QUIC 传输保持了 12.8 万 req/s,并把短请求平均延迟压到 1.84ms——方法与原始数据见主页[压测数据](/zh/#benchmark)。链路丢包严重时,两端都加 `-bbr` 启用 BBR 拥塞控制。

## 客户端与服务端之间走 TLS

同样的思路走流式 TLS——UDP 被封的环境适用:

```bash
gt server -addr 7000 -tlsAddr 443 \
  -certFile /etc/gt/tls.crt -keyFile /etc/gt/tls.key \
  -id id1 -secret secret1
```

```bash
gt client -local http://127.0.0.1:8080 \
  -remote tls://relay.example.com:443 \
  -id id1 -secret secret1
```

?> 这段 TLS 保护的是隧道本身。若还想让*访问者*到你域名这一段也端到端加密,可在服务端用 `-sniAddr` 做 SNI 原始转发(不碰载荷内容),或在发布 URL 前面套一层常规反代(Caddy/nginx)。

## P2P 直连

分处不同 NAT 后的两个客户端可以经 WebRTC 直接互通——中转只负责撮合握手(打洞失败时回退为中转流量)。示例拓扑:NAT-A 后的文件服务器("provider"),NAT-B 后的网关把它暴露在本地端口。

**中转服务端**(公网 VPS,同时提供 STUN):

```bash
gt server -addr 0.0.0.0:7000 -stunAddr 0.0.0.0:3478 \
  -id prov -secret prov-secret
```

**Provider**(承载真实服务,位于 NAT-A 后):

```bash
python3 -m http.server 8080   # 任意本地服务

gt client -id prov -secret prov-secret \
  -remote <server_ip>:7000 \
  -remoteSTUN stun:<server_ip>:3478 \
  -local http://127.0.0.1:8080 \
  -webrtcThreadMode
```

**Gateway**(位于 NAT-B 后,把 provider 暴露到本地):

```bash
gt client -id gw -secret gw-secret \
  -remote <server_ip>:7000 \
  -remoteSTUN stun:<server_ip>:3478 \
  -tcpForwardAddr 0.0.0.0:8888 \
  -tcpForwardHostPrefix prov
```

网关所在局域网的任何人打开 `http://<网关IP>:8888/`,即可看到 provider 的内容。两端都开 `-logLevel debug` 观察:两侧都出现 `typ srflx` 候选,说明 P2P 直达通道已建立。

NAT 兼容性速查:

| NAT 组合 | 结果 |
|---|---|
| 全锥型 × 全锥型(多数家宽) | 直连成功 |
| 受限锥型 × 受限锥型 | 成功 |
| 任一端为对称 NAT | 失败——P2P 无 TURN 回退,流量走中转 |
| 任一端有公网 IP | 成功 |

!> 记得在防火墙放行 `gt` 进程的 **UDP** 出入站,打洞依赖它。

## Docker 部署自建中转

`server-dev` 镜像启动时根据环境变量生成配置:

```bash
docker run -d --name gt-server --restart unless-stopped \
  -p 80:80 -p 443:443 -p 3478:3478/udp \
  -v /opt/gt/crt:/opt/crt:ro \
  -e NETWORK_ADDR=80 \
  -e NETWORK_TLSADDR=443 \
  -e NETWORK_STUNADDR=3478 \
  -e NETWORK_WEB_ADDR=0.0.0.0:8000 \
  ghcr.io/ao-space/gt:server-dev
```

- `-v .../crt` 把证书挂载为 `/opt/crt/tls.crt` + `tls.key`(TLS 与 QUIC 监听共用)
- 打开 `http://<vps>:8000` 进入 Web 管理台,签发客户端凭证
- 通过 `-remoteTCPPort` 发放的中转 TCP 端口也要在容器上发布(`-p 6022:6022`,或一段范围)

不想用容器?最小 systemd 单元:

```ini
# /etc/systemd/system/gt-server.service
[Unit]
Description=G-Tunnel relay
After=network-online.target

[Service]
ExecStart=/usr/local/bin/gt server -c /etc/gt/server.yaml
Restart=always

[Install]
WantedBy=multi-user.target
```

配套的带配额 `server.yaml` 见[多用户中转](#多用户中转与配额);全部选项见[配置参考](config.md#服务端选项)。

## 多用户中转与配额

在服务端配置中注册用户,各自持有凭证与限额:

```yaml
# server.yaml
addr: "7000"
webAddr: "127.0.0.1:8000"

users:
  alice:
    secret: alices-secret
    tcp:
      - range: 30000-39999   # 允许 -remoteTCPPort/-remoteTCPRandom 取用的端口池
    tcpNumber: 4             # 最多可开端口数
    speed: 10485760          # 字节/秒
    connections: 10          # 最大隧道连接数
  bob:
    secret: bobs-secret
    host:
      number: 2              # 最多主机前缀服务数
      withID: true           # 前缀自动带命名空间:bob-<前缀>
```

```bash
gt server -c ./server.yaml
```

不想写文件,直接在命令行重复 `-id`/`-secret` 对即可:

```bash
gt server -addr 7000 -id alice -secret alices-secret -id bob -secret bobs-secret
```

!> `allowAnyClient: true` 会把中转变成对任何人开放的开式代理——公网服务器切勿开启。

改了用户配置?不中断现有连接热加载:

```bash
gt -s reload
```

reload 会重新读取配置并保持转发不断流——压测中边跑流量边 reload,0 错误。
