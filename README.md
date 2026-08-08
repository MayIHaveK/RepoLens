# RepoLens

[![CI](https://github.com/MayIHaveK/RepoLens/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/MayIHaveK/RepoLens/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/MayIHaveK/RepoLens)](https://github.com/MayIHaveK/RepoLens/releases)
[![License](https://img.shields.io/github/license/MayIHaveK/RepoLens)](LICENSE)

RepoLens 是一款本地优先的 Git 贡献分析工具。它适合需要清晰、可复现地了解工程投入，又不希望上传私有源代码的团队。项目分别展示历史工作量、当前代码所有权、留存贡献和协作贡献，不会简单地用提交次数衡量贡献。

## 主要特性

- 流式读取 Git 历史，不会一次性把所有差异加载到内存。
- 使用有上限的并发 `blame` 任务计算当前代码所有权。
- 可以安全停止长时间分析，并同时终止正在运行的 Git 子进程。
- 通过可配置规则排除生成文件、第三方代码、二进制文件和超大文件。
- 支持贡献者身份别名和 `Co-authored-by` 联合署名。
- 按隐私设置导出单文件离线 HTML 报告。
- 所有分析均在本机完成；报告不会包含源代码片段和结构化文件名。
- 提供“快速”“平衡”“完整”三种性能配置。

## 快速开始

Windows 用户可以从[最新版本](https://github.com/MayIHaveK/RepoLens/releases/latest)下载 `repolens-windows-amd64.exe`，然后运行：

```powershell
.\repolens-windows-amd64.exe serve
```

RepoLens 会在 `http://127.0.0.1:41739` 打开，并且默认只监听本机回环地址。

从源代码构建需要安装 Git 2.40+、Go 1.26+ 和 Node.js 24+：

```powershell
cd web
npm install
npm run build
cd ..
go run ./cmd/repolens serve
```

生产构建会把前端嵌入单个可执行文件。在 Windows 上，`scripts/build.ps1` 会安装前端依赖、检查前后端代码、运行测试，并生成 `bin/repolens.exe`。

前端开发时，可以分别启动 API 和 Vite：

```powershell
go run ./cmd/repolens serve --no-open
cd web
npm run dev
```

## 命令行

```text
repolens serve [--address 127.0.0.1:41739] [--no-open]
repolens analyze <仓库路径> [--ref HEAD] [--output report.json]
repolens export <分析结果.json> <报告.html>
repolens version
```

## 隐私模型

RepoLens 只读且完全在本地分析。它不会检出文件，也不会执行仓库中的代码。导出设置可以控制是否包含仓库名称、提交信息、目录名称、贡献者身份、头像和提交哈希。

报告始终不会导出邮箱地址、源代码片段、本机绝对路径、凭据和结构化文件名字段。提交信息属于用户编写的文本，本身可能提到路径或文件名；公开报告需要严格隐藏这些信息时，请关闭“提交信息”。

## 贡献模型

默认综合贡献由以下归一化占比组成：

- 有效工作量：35%
- 当前所有权：35%
- 留存贡献：20%
- 协作贡献：10%

每个维度都会独立展示。用户可以调整所有权重；某项可选数据不可用时，其余权重会自动重新归一化。

代码类型权重、性能限制、机器人名称规则、生成文件与第三方代码规则、贡献者身份别名和导出隐私均可在界面中配置。

## 大型仓库

“快速”模式会跳过逐行所有权分析，适合迅速查看历史工作量。“平衡”模式使用有上限的并发、文件大小限制和所有权文件数量限制。“完整”模式会移除所有权文件数量限制。

相同提交和配置指纹的重复分析会直接使用本地缓存。分析运行期间可以从界面停止，不会遗留 Git 子进程。

精确定义和限制请参阅[贡献计算方法](docs/METHODOLOGY.md)。

## 项目状态

RepoLens 目前处于早期活跃开发阶段。本地 Git 分析、可视化面板、配置和离线导出已经组成首个可用版本。GitHub 元数据已经预留在配置与报告模型中，OAuth 和 API 同步计划在下一个里程碑实现。

## 许可证

本项目采用 MIT 许可证。
