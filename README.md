# RepoLens

[![CI](https://github.com/MayIHaveK/RepoLens/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/MayIHaveK/RepoLens/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/MayIHaveK/RepoLens)](https://github.com/MayIHaveK/RepoLens/releases)
[![License](https://img.shields.io/github/license/MayIHaveK/RepoLens)](LICENSE)

RepoLens is a local-first Git contribution analyzer for teams that need a
clear, reproducible view of engineering work without uploading private source
code. It separates historical workload, current code ownership, retained code,
and collaboration signals instead of reducing contribution to commit counts.

## Highlights

- Streams Git history instead of loading every diff into memory.
- Computes current ownership with a bounded blame worker pool.
- Cancels long-running analysis cleanly, including active Git subprocesses.
- Excludes generated, vendored, binary, and oversized files by configurable rules.
- Merges contributor aliases and recognizes co-authored commits.
- Exports a self-contained offline HTML report with privacy controls.
- Keeps analysis local. Source snippets and structured file names are never part of reports.
- Provides Fast, Balanced, and Thorough performance profiles.

## Quick start

For Windows, download `repolens-windows-amd64.exe` from the
[latest release](https://github.com/MayIHaveK/RepoLens/releases/latest), then run:

```powershell
.\repolens-windows-amd64.exe serve
```

RepoLens opens at `http://127.0.0.1:41739` and listens only on loopback.

To build from source, install Git 2.40+, Go 1.26+, and Node.js 24+:

```powershell
cd web
npm install
npm run build
cd ..
go run ./cmd/repolens serve
```

The production build is a single executable with the frontend embedded. On
Windows, `scripts/build.ps1` performs a clean frontend install, checks both
codebases, runs tests, and creates `bin/repolens.exe`.

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

Category weights, performance limits, bot patterns, generated and vendored
rules, contributor aliases, and export privacy are all configurable in the UI.

## Large repositories

The Fast profile skips line ownership for quick history-only results. Balanced
uses bounded concurrency, file-size limits, and an ownership-file cap. Thorough
removes the ownership-file cap. Repeated analysis at the same commit and
configuration fingerprint uses the local result cache. Active analysis can be
stopped from the UI without leaving Git subprocesses running.

See [docs/METHODOLOGY.md](docs/METHODOLOGY.md) for the precise definitions and
limitations.

## Project status

RepoLens is in active early development. The local Git analysis, dashboard,
configuration, and offline export form the first usable release. GitHub metadata
collection is represented in the configuration and report model; OAuth and API
sync are planned for the next milestone.

## License

MIT
