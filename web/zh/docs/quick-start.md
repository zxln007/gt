# 快速开始

从零到发布一个服务。命令以 Linux/macOS shell 为例,Windows 下使用对应 `.exe` 即可。

## 1. 安装

一行命令安装(Linux / macOS):

```bash
curl -fsSL https://gtunnel.dev/install.sh | sh
```

脚本会自动识别操作系统与 CPU 架构,从 [GitHub Releases](https://github.com/zxln007/gt/releases) 下载对应二进制,安装到 `/usr/local/bin`(无 root 时为 `~/.local/bin`)。用 `gt --version` 验证。

其他方式:

- **直接下载**——Windows / macOS / Linux(x86_64、arm64、RISC-V)静态单文件,见[下载页](/zh/)。
- **桌面版**——同一引擎的图形界面封装,不想敲命令选它。
- **Docker**——服务端可用 `ghcr.io/ao-space/gt:server-dev` 等镜像,见 [Docker 部署](examples.md#docker-部署自建中转)。

## 2. 五分钟体验(零配置)

两种角色都支持零参数启动,并打开本地 Web 管理台编辑配置:

```bash
gt server
```

日志会打印管理台地址,默认 `http://127.0.0.1:8000`。打开后可在表单里配置监听、用户与服务,无需手写 YAML。客户端同理,端口 `7000`:

```bash
gt client
```

!> 在 Web 管理台编辑的配置会保存到默认配置路径,重启后依然生效。推荐用管理台搭建配置;下文的 YAML 就是它生成的产物。

## 3. 发布第一个服务

习惯终端?两个终端,三条命令。

**终端 1——启动中转服务端**(`-id`/`-secret` 注册一个客户端凭证):

```bash
gt server -addr 12080 -id id1 -secret secret1
```

**终端 2——连接客户端**,声明要发布的本地服务:

```bash
gt client -local http://127.0.0.1:8080 -remote tcp://127.0.0.1:12080 -id id1 -secret secret1
```

**验证。** 服务端按主机名前缀路由 HTTP(客户端的 `id` 即前缀),让 curl 带上匹配的 `Host` 头:

```bash
curl -H "Host: id1.example.com" http://127.0.0.1:12080/
```

应该能看到 `127.0.0.1:8080` 本地服务的响应。在真实服务器上配好域名后,访问者直接打开 `http://id1.你的域名:12080/` 即可。

## 4. 接下来

- 实战场景(NAT 后 SSH、一客户端多服务、P2P):[使用实例](examples.md)
- 全部参数与 YAML 选项:[配置参考](config.md)
- 出问题了:[常见问题](faq.md)
