# 常见问题

## 访问者使用的主机名是什么?

`<前缀>.<中转域名>`,如 `myapp.relay.example.com`。前缀默认是客户端 `id`;可用 `-hostPrefix` / `services[].hostPrefix` 覆盖或追加。共享中转可能开启 `host.withID`,前缀会带命名空间变成 `id-host`(`myapp-web.relay.example.com`),避免用户间冲突。

## 零参数启动 `gt server` 之后呢?

打开日志里打印的管理台地址(服务端默认 `http://127.0.0.1:8000`,客户端默认 `http://127.0.0.1:7000`),在页面里完成全部配置;配置会保存并在下次启动时复用。

## 能不断连接就改配置吗?

可以。编辑配置后执行 `gt -s reload`,工作进程在转发不中断的情况下重新读取配置。整进程重启(`gt -s restart`)与二进制升级同样能保持转发;压测中带载 reload 实测 0 错误。

## `-remoteTCPRandom` 分到了哪个端口?

服务端从该用户的端口池里分配(`users[].tcp[].range`,如 `30000-39999`,受 `tcpNumber` 约束)。客户端启动日志会打印映射地址;看不到就加 `-logLevel debug`。

## P2P 直连建立不起来

按可能性排序:

1. **任一端是对称 NAT**——预期内失败;P2P 没有 TURN 回退,流量会走中转。
2. **UDP 被封**——打洞依赖 UDP;两端防火墙都要放行 `gt` 进程(Windows Defender 也算)。
3. **版本不一致**——两端必须用同一构建版本。
4. 两端都开 `-logLevel debug` 排查:两侧都出现 `typ srflx` 候选说明 STUN 正常;若选中的候选对是 `relay`,看到的就是回退路径。

## 如何保护我的中转服务器?

- 只签发自己发放过的凭证(公网地址上绝不 `allowAnyClient`)
- Web 管理台只监听 `127.0.0.1`,或配置认证(`admin`、`password`、`signingKey`)
- 给每个用户配额:`speed`、`connections`、`tcpNumber`、`host.number`
- 优先提供 `tls://`/`quic://` 监听,凭证不以明文过网

按设计,日志不会打印密钥与流量内容。

## 支持哪些平台?

Windows(x86_64)、macOS(Apple Silicon、Intel)、Linux(x86_64、arm64、RISC-V)的静态单文件,见 [GitHub Releases](https://github.com/zxln007/gt/releases);服务端与客户端均有 Docker 镜像。同一引擎也内置在 Windows/macOS 的 [GT-Desktop](/zh/) 中。

## 与 frp 相比如何?

都做反向隧道。G-Tunnel 的 wrk 压测显示吞吐最高 6 倍、平均延迟约 1/8、满并发零错误,并内置 QUIC 与 P2P——方法与原始输出见[压测数据](/zh/#benchmark)。
