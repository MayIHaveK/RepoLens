# RepoLens

RepoLens is a local-first Git contribution analyzer for teams that need a
clear, reproducible view of engineering work without uploading private source
code. It separates historical workload, current code ownership, retained code,
and collaboration signals instead of reducing contribution to commit counts.

## Highlights

- Streams Git history instead of loading every diff into memory.
- Computes current ownership with a bounded blame worker pool.
- Excludes generated, vendored, binary, and oversized files by configurable rules.
- Merges contributor aliases and recognizes co-authored commits.
- Exports a self-contained offline HTML report with privacy controls.
- Keeps analysis local. Source snippets and file names are never part of reports.
- Provides Fast, Balanced, and Thorough performance profiles.

## Quick start

Requirements: Git 2.40+, Go 1.24+, and Node.js 20+ for frontend development.

```powershell
cd web
npm install
npm run build
cd ..
go run ./cmd/repolens serve
```

Open `http://127.0.0.1:41739`. The production build is a single executable with
the frontend embedded.

For frontend development, run the API and Vite separately:

```powershell
go run ./cmd/repolens serve --no-open
cd web
npm run dev
```

## CLI

```text
repolens serve [--address 127.0.0.1:41739] [--no-open]
repolens analyze <repository> [--ref HEAD] [--output report.json]
repolens export <analysis.json> <report.html>
repolens version
```

## Privacy model

Analysis is read-only and local. RepoLens never checks out files or executes
repository code. Export configuration controls repository name, commit messages,
directory names, contributor identities, avatars, and commit hashes. Email
addresses, source snippets, absolute paths, credentials, and structured
file-name fields are never exported. Commit messages are user-authored text and
may themselves mention a path or file name; disable commit messages for a
strict public report.

## Contribution model

The default composite score combines normalized shares:

- Effective workload: 35%
- Current ownership: 35%
- Retained contribution: 20%
- Collaboration: 10%

Every dimension is also shown separately. Weights are user configurable and
renormalized when an optional data source is unavailable.

See [docs/METHODOLOGY.md](docs/METHODOLOGY.md) for the precise definitions and
limitations.

## Project status

RepoLens is in active early development. The local Git analysis, dashboard,
configuration, and offline export form the first usable release. GitHub metadata
collection is represented in the configuration and report model; OAuth and API
sync are planned for the next milestone.

## License

MIT
