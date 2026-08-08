# 参与贡献

欢迎为 RepoLens 贡献代码。请保持分析结果确定且可复现，记录贡献计算方式的任何变化，并为解析器和评分规则补充测试。

提交 Pull Request 前请运行：

```powershell
go test ./...
cd web
npm ci
npm run check
npm run build
```

请保持每个提交的改动目标明确。修改贡献计算公式时，必须同步更新 `docs/METHODOLOGY.md`；如果已保存报告的数据结构发生变化，还需要提供迁移说明。
