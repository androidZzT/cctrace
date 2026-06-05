<div align="center">
  <h1>cctrace</h1>
  <p><strong>AI 编程 agent 的实时 profiler。包一层 Claude Code 或 Codex，整段会话的每一个事件都看得见，本地瀑布流时间线。</strong></p>
  <p><em>一个 Go 二进制。JSONL 落盘，web UI 跑本地。无后台进程，无遥测，无云。</em></p>

  <p>
    <a href="https://github.com/androidZzT/cctrace/stargazers"><img src="https://img.shields.io/github/stars/androidZzT/cctrace?style=flat-square" alt="Stars"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/androidZzT/cctrace?style=flat-square" alt="License"></a>
    <img src="https://img.shields.io/badge/go-1.22%2B-00ADD8?style=flat-square&logo=go" alt="Go">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey?style=flat-square" alt="Platform">
    <img src="https://img.shields.io/badge/local--first-no%20telemetry-orange?style=flat-square" alt="Local-first">
  </p>

  <p>
    <a href="#-为什么要-cctrace">为什么</a> &bull;
    <a href="#-功能">功能</a> &bull;
    <a href="#%EF%B8%8F-截图">截图</a> &bull;
    <a href="#-快速开始">快速开始</a> &bull;
    <a href="#-cli-命令">CLI</a> &bull;
    <a href="#-数据与隐私">数据与隐私</a> &bull;
    <a href="#%EF%B8%8F-架构">架构</a> &bull;
    <a href="README.md">English</a>
  </p>
</div>

---

<p align="center">
  <img src="docs/screenshots/demo.gif" alt="cctrace 瀑布流时间线演示" width="900">
</p>

---

## 🤔 为什么要 cctrace？

用 Claude Code 或 Codex 跑一个复杂任务时，背后有很多东西你看不到：

- 哪些工具调用触发了、每个跑了多久
- 进程事件、transcript 事件、ccglass 记录之间的先后顺序
- 模型什么时候在想、什么时候卡住、什么时候重试、什么时候 fan out 子 agent
- 整段 wall-clock 时间到底花在哪了

cctrace 把 agent 进程包一层，从进程活动、transcript 文件、ccglass trace 三个来源拉事件进同一套模型，做关联，整段会话铺在本地瀑布流时间线上。

不起后台进程，不发遥测，JSONL 落盘，web UI 跑在 localhost。

## ✨ 功能

- **包一层任意 AI 编程 agent** — `cctrace claude -- <cmd>` 或 `cctrace codex -- <cmd>` 把整段会话从头录到尾
- **统一 trace 模型** — process / transcript / ccglass 三种记录归到同一套 Event / Session schema
- **启发式关联** — 跨来源链接相关事件，每条关联打置信度分
- **本地瀑布流 UI** — 浏览器里看时间线，拖、缩放、点开任意事件看详情
- **可回放** — 每次会话就是磁盘上的 JSONL 文件，后面用 `cctrace view <session-id>` 重新打开
- **一个 Go 二进制** — 不要 Python、不要 Node、不要 Docker、不要常驻服务

## 🖼️ 截图

一次录制会话的瀑布流时间线——左侧按请求轮次分组，每一个事件（用户、AI、工具调用、工具结果）都铺在同一条时间轴上，顶部是事件密度总览。

![cctrace 瀑布流时间线](docs/screenshots/waterfall-overview.png)

点任意一条工具操作，就能看它的参数、输出、耗时、关联 ID 以及原始事件 JSON。

![cctrace 事件详情](docs/screenshots/event-details.png)

## 🚀 快速开始

```sh
# 从源码安装
go install github.com/androidZzT/cctrace/cmd/cctrace@latest

# 包一层 Claude Code
cctrace claude -- claude

# 包一层 Codex
cctrace codex -- codex

# 回放保存的会话
cctrace view <session-id>
```

包一层 agent 时会立刻打印本地 UI 的 URL，打开它就能看实时瀑布流：

![cctrace 启动并打印本地 UI URL](docs/screenshots/cli-start.png)

## 📖 CLI 命令

| 命令 | 作用 |
|---|---|
| `cctrace claude -- <cmd>` | 包一层 Claude Code 调用，录 process + transcript + ccglass 事件 |
| `cctrace codex -- <cmd>` | 包一层 Codex 调用，同一套录制管线 |
| `cctrace view <session-id>` | 在本地 web UI 里打开保存的会话 |

`cctrace --help` 看完整参数。

## 🔐 数据与隐私

cctrace 默认就是本地优先：

- web UI 默认监听 `127.0.0.1:43179`。
- 会话默认写到 `/tmp/cctrace-sessions`。
- 每个会话目录里有一个 `session.json` 元数据文件和一个 `events.jsonl` 事件流。
- cctrace 不发遥测，也不会把 trace 上传到托管服务。
- 落盘的 JSONL 可能包含命令参数、工具输出、transcript 片段、文件路径、模型元数据，以及 ccglass 来源里的请求/响应耗时或 usage 字段。

请把 trace 文件当成调试日志看待：里面可能包含敏感路径、prompt、命令输出或模型/工具 payload。对外分享前建议先审阅或脱敏。

保存过的会话可以直接回放，不需要重跑 agent：

```sh
cctrace view <session-id>
```

web UI 顶部的导入栏也可以导入已有 trace：

- 包含 `events.jsonl` 的 cctrace session 目录
- Codex rollout JSONL 文件或 rollout 目录包
- Claude transcript / session 文件夹

所以 cctrace 既可以做实时 profiler，也可以当轻量离线 trace viewer 用。

## 🏗️ 架构

三个来源汇进同一套模型，做关联并打置信度分，最后铺到本地时间线上：

![cctrace 数据流：三源 → collectors → 统一模型 → correlate → UI](docs/screenshots/dataflow.png)

```
cmd/cctrace                 CLI 入口
└── internal/
    ├── app                 装配 CLI 解析、采集、持久化、server 启动
    ├── trace               共享的 Event / Session 模型
    ├── store               把会话和事件落到 session 目录的 JSONL
    ├── collectors          process / transcript / ccglass → trace events
    ├── correlate           启发式事件关联 + 置信度打分
    └── server              本地 web UI + 事件 JSON API
```

会话存成 JSONL，可以直接 grep、diff、或者管道给别的工具用，不一定要走 cctrace 自己的 UI。

## 🛠️ 开发

```sh
# 跑全部测试
go test ./...

# 开发模式跑 CLI
go run ./cmd/cctrace claude -- claude

# 冒烟测一条 trace
go run ./cmd/cctrace claude -- sh -c 'printf cctrace-smoke'

# 回放
go run ./cmd/cctrace view <session-id>
```

## 📝 License

[MIT](LICENSE)

## 🙏 致谢

cctrace 的事件来源之一，是读取 [ccglass](https://github.com/jianshuo/ccglass)（[@jianshuo](https://github.com/jianshuo) 出品，MIT）导出的 trace 记录；`cctrace <agent> -- <cmd>` 的命令范式也借鉴了 ccglass「包一层、看清楚」的思路。想看清 agent 到底给模型发了什么，推荐去看 ccglass，和 cctrace 搭着用很合适。

---

<div align="center">
  <sub>用 Go 写的。本地优先。无遥测。</sub>
</div>
