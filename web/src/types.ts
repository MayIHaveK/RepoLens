export type Category = 'source' | 'test' | 'docs' | 'config' | 'asset' | 'other'

export interface Weights {
  workload: number
  ownership: number
  retention: number
  collaboration: number
}

export interface Privacy {
  showRepositoryName: boolean
  showCommitMessages: boolean
  showDirectories: boolean
  showContributors: boolean
  showAvatars: boolean
  showCommitSha: boolean
}

export interface Config {
  ref: string
  since?: string
  until?: string
  profile: 'fast' | 'balanced' | 'thorough'
  parallelism: number
  maxFileSizeBytes: number
  maxOwnershipFiles: number
  maxInventoryFiles: number
  enableOwnership: boolean
  enableRetention: boolean
  enableCollaboration: boolean
  detectRenames: boolean
  includeBots: boolean
  includeMerges: boolean
  weights: Weights
  categoryWeights: Record<Category, number>
  privacy: Privacy
  ignoredPatterns: string[]
  generatedPatterns: string[]
  vendoredPatterns: string[]
  botPatterns: string[]
  aliases: Record<string, string>
}

export interface Change {
  additions: number
  deletions: number
}

export interface Contributor {
  id: string
  name: string
  avatarUrl?: string
  commits: number
  additions: number
  deletions: number
  ownedLines: number
  historicalAdded: number
  retentionRate: number
  workloadShare: number
  ownershipShare: number
  retentionShare: number
  collaborationShare: number
  compositeShare: number
  reviews: number
  issues: number
  pullRequests: number
  firstCommitAt?: string
  lastCommitAt?: string
  categories: Record<Category, Change>
  topDirectories?: { name: string; additions: number; deletions: number }[]
  recentCommits?: { sha: string; message: string; timestamp: string; additions: number; deletions: number }[]
}

export interface Analysis {
  programVersion?: string
  algorithmVersion?: string
  project?: Project
  schemaVersion: number
  id: string
  repository: { name: string; ref: string; commitSha: string; headDate?: string; commitCount: number; fileCount: number }
  generatedAt: string
  durationMs: number
  mode: string
  config: { profile: string; weights: Weights; effectiveWeights?: Weights; effective?: Config; ownershipEnabled: boolean; githubEnabled: boolean; privacy: Privacy; fingerprint: string; categoryWeights: Record<Category, number> }
  summary: { contributors: number; commits: number; files: number; additions: number; deletions: number; ownedLines: number; analyzedFiles: number; skippedFiles: number; coveragePercent: number }
  contributors: Contributor[]
  timeline: { period: string; additions: number; deletions: number; commits: number; contributors?: Record<string, Change> }[]
  categories: { category: Category; additions: number; deletions: number; files: number }[]
  warnings?: string[]
}

export type RecentAnalysis = Pick<Analysis, 'id' | 'repository' | 'generatedAt'>
export interface LineCounts { total: number; code: number; comment: number; blank: number }
export interface FileMetric {
  path: string; category: Category; language: string; bytes: number; lines: LineCounts
  status: string; reason?: string; commits: number; additions: number; deletions: number; lastChangedAt?: string
}
export interface Dependency { from: string; to: string; kind: string; resolved: boolean }
export interface ConfigurationInfo {
  path: string; format: string; keys?: string[]; parseStatus: string
  dependencies: number; scripts: number; issues?: string[]
}
export interface Project {
  scope: string; totals: LineCounts; sourceCodeLines: number; textFiles: number; totalFiles: number
  skipped: Record<string, number>; historySkipped: Record<string, number>
  files: FileMetric[]; languages: { language: string; files: number; lines: LineCounts }[]
  directories: { path: string; files: number; lines: LineCounts }[]
  dependencies: Dependency[]; configurations: ConfigurationInfo[]; cycles: string[][]; notes: string[]
}

export interface Job {
  id: string
  status: 'queued' | 'running' | 'complete' | 'failed' | 'cancelled'
  progress: { phase: string; message: string; current: number; total: number; percent: number }
  analysisId?: string
  error?: string
  cached: boolean
}
