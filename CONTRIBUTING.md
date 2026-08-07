# Contributing

Contributions are welcome. Keep analysis behavior deterministic, document any
metric changes, and add tests for parsers and scoring rules.

Before opening a pull request:

```powershell
go test ./...
cd web
npm ci
npm run check
npm run build
```

Use focused commits. Changes to contribution formulas must update
`docs/METHODOLOGY.md` and include migration notes when stored report data changes.

