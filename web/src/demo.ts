import type { Analysis, Category, Change, Contributor } from './types'

const change = (additions: number, deletions: number): Change => ({ additions, deletions })
const categories = (source: number, test: number, docs: number): Record<Category, Change> => ({
  source: change(source, Math.round(source * 0.31)),
  test: change(test, Math.round(test * 0.18)),
  docs: change(docs, Math.round(docs * 0.08)),
  config: change(Math.round(source * 0.05), Math.round(source * 0.02)),
  asset: change(0, 0),
  other: change(Math.round(source * 0.03), Math.round(source * 0.01)),
})

const people: Contributor[] = [
  { id: '01', name: 'MayIHaveK', commits: 184, additions: 28640, deletions: 9310, ownedLines: 18324, historicalAdded: 28640, retentionRate: 64, workloadShare: 54.2, ownershipShare: 59.8, retentionShare: 63.1, collaborationShare: 47.2, compositeShare: 57.6, reviews: 18, issues: 23, pullRequests: 42, categories: categories(24400, 2810, 920), topDirectories: [{ name: 'server', additions: 10620, deletions: 3280 }, { name: 'client', additions: 8340, deletions: 2900 }] },
  { id: '02', name: 'kaixiten', commits: 96, additions: 17120, deletions: 6140, ownedLines: 9470, historicalAdded: 17120, retentionRate: 55.3, workloadShare: 33.1, ownershipShare: 30.9, retentionShare: 27.2, collaborationShare: 36.1, compositeShare: 31.9, reviews: 14, issues: 11, pullRequests: 21, categories: categories(13920, 1820, 680), topDirectories: [{ name: 'gameplay', additions: 7920, deletions: 2040 }] },
  { id: '03', name: 'Lin Chen', commits: 34, additions: 5680, deletions: 1910, ownedLines: 2860, historicalAdded: 5680, retentionRate: 50.4, workloadShare: 10.5, ownershipShare: 9.3, retentionShare: 8.1, collaborationShare: 13.9, compositeShare: 9.9, reviews: 7, issues: 4, pullRequests: 8, categories: categories(4020, 1030, 410) },
  { id: '04', name: 'Avery', commits: 11, additions: 1140, deletions: 320, ownedLines: 0, historicalAdded: 1140, retentionRate: 0, workloadShare: 2.2, ownershipShare: 0, retentionShare: 1.6, collaborationShare: 2.8, compositeShare: 0.6, reviews: 2, issues: 1, pullRequests: 2, categories: categories(780, 140, 180) },
]

export const demoAnalysis: Analysis = {
  schemaVersion: 1,
  id: 'demo-analysis',
  repository: { name: 'dragonminez', ref: 'master', commitSha: '99e1dd31b49622a0ec1b91ac92a2cc296f59bc28', commitCount: 325, fileCount: 1842 },
  generatedAt: new Date().toISOString(),
  durationMs: 8431,
  mode: 'git-only',
  config: { profile: 'balanced', weights: { workload: 0.35, ownership: 0.35, retention: 0.2, collaboration: 0.1 }, ownershipEnabled: true, githubEnabled: false, privacy: { showRepositoryName: true, showCommitMessages: true, showDirectories: true, showContributors: true, showAvatars: true, showCommitSha: true }, fingerprint: 'c91d0f82a137bc42', categoryWeights: { source: 1, test: 0.9, docs: 0.45, config: 0.65, asset: 0.25, other: 0.4 } },
  summary: { contributors: 4, commits: 325, files: 1842, additions: 52580, deletions: 17680, ownedLines: 30654, analyzedFiles: 1627, skippedFiles: 215, coveragePercent: 88.3 },
  contributors: people,
  timeline: [
    ['2026-02', 2100, 480, 14], ['2026-03', 3900, 1040, 26], ['2026-04', 6200, 1860, 41],
    ['2026-05', 8400, 2430, 55], ['2026-06', 11300, 3810, 68], ['2026-07', 14980, 5140, 83], ['2026-08', 5700, 2920, 38],
  ].map(([period, additions, deletions, commits]) => ({ period: String(period), additions: Number(additions), deletions: Number(deletions), commits: Number(commits) })),
  categories: [
    { category: 'source', additions: 43120, deletions: 13740, files: 918 },
    { category: 'test', additions: 5800, deletions: 1320, files: 264 },
    { category: 'docs', additions: 2190, deletions: 430, files: 112 },
    { category: 'config', additions: 1470, deletions: 710, files: 333 },
  ],
  warnings: [],
}

