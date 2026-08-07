# Architecture

RepoLens is split into a streaming analysis core, a small local HTTP service,
and a browser UI embedded into the release binary.

## Data flow

1. Validate and resolve the repository and revision.
2. Stream one `git log --numstat` process for historical workload.
3. Enumerate the selected tree without checking it out.
4. Run bounded concurrent blame processes for eligible files.
5. Normalize identities and calculate dimension shares.
6. write a versioned analysis snapshot to the user cache directory.
7. Serve the snapshot to the local UI or export a standalone HTML report.

## Large repositories

- Git output is parsed as a stream.
- Contributor and timeline aggregates are bounded; full diffs are not retained.
- Blame concurrency is configurable and canceled with the parent context.
- Oversized and binary files are rejected before blame.
- Result snapshots are keyed by commit and configuration fingerprint.
- The Fast profile skips blame; Balanced caps files and parallelism; Thorough
  removes most caps.

## Security boundary

The server binds to loopback by default. Repository paths are never returned to
the browser after a job is created. The analyzer invokes Git with arguments
directly and does not use a shell, check out code, load project dependencies, or
execute hooks.

