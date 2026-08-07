# Contribution methodology

RepoLens reports evidence from version-control history. It does not claim to
measure developer value, difficulty, quality, mentoring, or product impact.

## Dimensions

### Effective workload

Effective workload is based on additions and deletions from non-merge commits.
Files excluded by ignore, generated, vendored, binary, or size rules contribute
no workload. Source, test, documentation, configuration, and asset changes are
reported separately. The default score uses additions plus half of deletions;
both raw values remain visible.

### Current ownership

Current ownership counts lines attributed by `git blame --line-porcelain` at the
selected revision. It is a maintenance-oriented view of the current tree, not a
claim of legal ownership. Blank and comment-only lines are not distinguished in
the initial implementation.

### Retained contribution

Retained lines are current owned lines divided by historical additions for the
same contributor. This is an approximation. Refactors, squashes, copied code,
and rewritten history can change attribution without reflecting the underlying
human effort.

### Collaboration

Collaboration uses optional forge metadata such as reviews and issue activity.
Pull-request code is not counted again because its diff already contributes to
effective workload. When forge data is unavailable, RepoLens omits this
dimension and renormalizes the available weights.

## Identity

The author identity is preferred over the committer identity. `.mailmap`, explicit
aliases, and `Co-authored-by` trailers may merge or split credit. Co-authored
workload is divided equally among recognized authors for that commit.

## Merge commits

Merge commits are excluded from workload by default to avoid counting changes
already attributed to their original commits. Only commits reachable from the
selected revision and matching the time filters are included.

## Reproducibility

Each report records the selected revision, resolved commit, RepoLens version,
configuration fingerprint, and generation time. A recipient without access to a
private repository cannot independently prove that the supplied Git history was
authentic. Reports are reproducible evidence, not third-party attestations.

