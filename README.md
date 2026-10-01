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
- 项目画像独立统计文本总行、有效源代码行、纯注释行与空白行，按语言、文件和各层目录汇总。
- 本地文件明细支持搜索、分类、排序、分页和变更热点查看；静态关系支持 Go 包、JS/TS 导入、页面资源与 tsconfig 引用。
- 配置分析检查 JSON/JSONC 结构、重复键和配置引用；YAML/TOML 等提供顶层结构检查，不保存配置值。
- 分析设置自动保存在浏览器中，可导入、导出 JSON；命令行可通过 `--config` 使用同一配置。

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
repolens analyze <仓库路径> [--ref HEAD] [--profile balanced] [--config settings.json] [--since 2026-01-01] [--until 2026-01-31] [--output report.json]
repolens export <分析结果.json> <报告.html>
repolens version
```

## 隐私模型

RepoLens 只读且完全在本地分析。它不会检出文件，也不会执行仓库中的代码。导出设置可以控制是否包含仓库名称、提交信息、目录名称、贡献者身份、头像和提交哈希。

报告始终不会导出邮箱地址、源代码片段、本机绝对路径、凭据和结构化文件名字段。提交信息属于用户编写的文本，本身可能提到路径或文件名；公开报告需要严格隐藏这些信息时，请关闭“提交信息”。

上述导出保护针对离线 HTML 报告。本机缓存和命令行分析 JSON 保存完整分析配置、身份别名及文件关系，适合留在本机使用；离线 HTML 会移除完整配置、文件明细、配置键、依赖边和循环组，只保留统计汇总及用户允许的目录名称。

## 项目分析范围

代码总量读取目标提交的 Git Blob，不包含未提交修改和未跟踪文件。总行包含通过过滤及大小限制的 UTF-8 文本；“有效源代码行”只汇总源代码与测试的非空、非纯注释行。注释使用词法估算，混合代码与注释的行归入代码；文档和未支持格式只统计非空内容，不等同于 AST 语句数量。

历史工作量按作者时间筛选，开始／结束日期按 UTC 解释，结束日期包含当天。日期筛选不会改变目标文件树，因此此时停用近似留存指标，避免将区间新增与全量当前归属相除。界面展示最终生效限制和实际评分权重。

Go 引用基于导入语法，JS/TS 等基于静态字符串规则。当前不推断路径别名、动态表达式、调用关系或运行时配置合并。目录统计包含子目录，各层数据不能相加；循环组表示强连通关系，不代表调用顺序。文件历史以当前路径汇总，重命名前的旧路径工作量仍属于历史汇总，不自动迁移到新路径。

## 贡献模型

默认综合贡献由以下归一化占比组成：

- 有效工作量：35%
- 当前所有权：35%
- 留存贡献：20%
- 协作贡献：10%

每个维度都会独立展示。用户可以调整所有权重；某项可选数据不可用时，其余权重会自动重新归一化。

代码类型权重、性能限制、机器人名称规则、生成文件与第三方代码规则、贡献者身份别名和导出隐私均可在界面中配置。

## 大型仓库

“快速”模式会跳过逐行所有权分析，但仍生成项目画像。“平衡”模式使用有上限的并发、文件大小限制和所有权文件数量限制。“完整”模式将所有权文件上限提高至 100 万；画像文件上限与文件大小限制仍生效。归属达到上限时按路径哈希确定性选取文件，并显示部分覆盖警告。

相同提交、最终配置、程序／算法版本、Git 版本及身份映射输入的重复分析会使用本地缓存。服务全局一次运行一个分析，最多接受 8 个活动任务；分析期间可停止，刷新页面可恢复当前服务中的任务。

精确定义和限制请参阅[贡献计算方法](docs/METHODOLOGY.md)。

## 项目状态

RepoLens 目前处于早期活跃开发阶段。本地 Git 分析、项目画像、配置结构检查、静态文件关系及离线导出已实现。GitHub 元数据字段仍为预留，OAuth 和 API 同步计划在后续里程碑实现。当前尚未提供完整调用图、复杂度评估或版本基线对比。

## 许可证

本项目采用 MIT 许可证。
