package model

import (
	"encoding/json"
	"time"
)

const SchemaVersion = 2
const AlgorithmVersion = "inventory-2"

type Category string

const (
	CategorySource Category = "source"
	CategoryTest   Category = "test"
	CategoryDocs   Category = "docs"
	CategoryConfig Category = "config"
	CategoryAsset  Category = "asset"
	CategoryOther  Category = "other"
)

type Analysis struct {
	ProgramVersion   string            `json:"programVersion"`
	AlgorithmVersion string            `json:"algorithmVersion"`
	Project          *Project          `json:"project,omitempty"`
	SchemaVersion    int               `json:"schemaVersion"`
	ID               string            `json:"id"`
	Repository       Repository        `json:"repository"`
	GeneratedAt      time.Time         `json:"generatedAt"`
	DurationMS       int64             `json:"durationMs"`
	Mode             string            `json:"mode"`
	Config           ConfigSnapshot    `json:"config"`
	Summary          Summary           `json:"summary"`
	Contributors     []Contributor     `json:"contributors"`
	Timeline         []TimelineBucket  `json:"timeline"`
	Categories       []CategorySummary `json:"categories"`
	Warnings         []string          `json:"warnings,omitempty"`
}

type Repository struct {
	Name        string `json:"name"`
	Ref         string `json:"ref"`
	CommitSHA   string `json:"commitSha"`
	HeadDate    string `json:"headDate,omitempty"`
	CommitCount int64  `json:"commitCount"`
	FileCount   int64  `json:"fileCount"`
}

type ConfigSnapshot struct {
	Effective        json.RawMessage      `json:"effective,omitempty"`
	EffectiveWeights Weights              `json:"effectiveWeights"`
	Profile          string               `json:"profile"`
	Weights          Weights              `json:"weights"`
	OwnershipEnabled bool                 `json:"ownershipEnabled"`
	GitHubEnabled    bool                 `json:"githubEnabled"`
	Privacy          Privacy              `json:"privacy"`
	Fingerprint      string               `json:"fingerprint"`
	CategoryWeights  map[Category]float64 `json:"categoryWeights"`
}

type Weights struct {
	Workload      float64 `json:"workload"`
	Ownership     float64 `json:"ownership"`
	Retention     float64 `json:"retention"`
	Collaboration float64 `json:"collaboration"`
}

type Privacy struct {
	ShowRepositoryName bool `json:"showRepositoryName"`
	ShowCommitMessages bool `json:"showCommitMessages"`
	ShowDirectories    bool `json:"showDirectories"`
	ShowContributors   bool `json:"showContributors"`
	ShowAvatars        bool `json:"showAvatars"`
	ShowCommitSHA      bool `json:"showCommitSha"`
}

type Summary struct {
	Contributors    int     `json:"contributors"`
	Commits         int64   `json:"commits"`
	Files           int64   `json:"files"`
	Additions       int64   `json:"additions"`
	Deletions       int64   `json:"deletions"`
	OwnedLines      int64   `json:"ownedLines"`
	AnalyzedFiles   int64   `json:"analyzedFiles"`
	SkippedFiles    int64   `json:"skippedFiles"`
	CoveragePercent float64 `json:"coveragePercent"`
}

type Contributor struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	AvatarURL          string              `json:"avatarUrl,omitempty"`
	Commits            int64               `json:"commits"`
	Additions          int64               `json:"additions"`
	Deletions          int64               `json:"deletions"`
	OwnedLines         int64               `json:"ownedLines"`
	HistoricalAdded    int64               `json:"historicalAdded"`
	RetentionRate      float64             `json:"retentionRate"`
	WorkloadShare      float64             `json:"workloadShare"`
	OwnershipShare     float64             `json:"ownershipShare"`
	RetentionShare     float64             `json:"retentionShare"`
	CollaborationShare float64             `json:"collaborationShare"`
	CompositeShare     float64             `json:"compositeShare"`
	Reviews            int64               `json:"reviews"`
	Issues             int64               `json:"issues"`
	PullRequests       int64               `json:"pullRequests"`
	FirstCommitAt      string              `json:"firstCommitAt,omitempty"`
	LastCommitAt       string              `json:"lastCommitAt,omitempty"`
	Categories         map[Category]Change `json:"categories"`
	TopDirectories     []DirectoryShare    `json:"topDirectories,omitempty"`
	RecentCommits      []CommitSummary     `json:"recentCommits,omitempty"`
}

// Project contains local file metadata, never source text or configuration values.
type Project struct {
	HistorySkipped  map[string]int64    `json:"historySkipped"`
	Scope           string              `json:"scope"`
	Totals          LineCounts          `json:"totals"`
	SourceCodeLines int64               `json:"sourceCodeLines"`
	TextFiles       int64               `json:"textFiles"`
	TotalFiles      int64               `json:"totalFiles"`
	Skipped         map[string]int64    `json:"skipped"`
	Files           []FileMetric        `json:"files,omitempty"`
	Languages       []LanguageSummary   `json:"languages"`
	Directories     []DirectorySummary  `json:"directories,omitempty"`
	Dependencies    []Dependency        `json:"dependencies,omitempty"`
	Configurations  []ConfigurationInfo `json:"configurations,omitempty"`
	Cycles          [][]string          `json:"cycles,omitempty"`
	Notes           []string            `json:"notes"`
}

type LineCounts struct {
	Total   int64 `json:"total"`
	Code    int64 `json:"code"`
	Comment int64 `json:"comment"`
	Blank   int64 `json:"blank"`
}
type FileMetric struct {
	Path          string     `json:"path"`
	Category      Category   `json:"category"`
	Language      string     `json:"language"`
	Bytes         int64      `json:"bytes"`
	Lines         LineCounts `json:"lines"`
	Status        string     `json:"status"`
	Reason        string     `json:"reason,omitempty"`
	Commits       int64      `json:"commits"`
	Additions     int64      `json:"additions"`
	Deletions     int64      `json:"deletions"`
	LastChangedAt string     `json:"lastChangedAt,omitempty"`
}
type LanguageSummary struct {
	Language string     `json:"language"`
	Files    int64      `json:"files"`
	Lines    LineCounts `json:"lines"`
}
type DirectorySummary struct {
	Path  string     `json:"path"`
	Files int64      `json:"files"`
	Lines LineCounts `json:"lines"`
}
type Dependency struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Kind     string `json:"kind"`
	Resolved bool   `json:"resolved"`
}
type ConfigurationInfo struct {
	Path         string   `json:"path"`
	Format       string   `json:"format"`
	Keys         []string `json:"keys,omitempty"`
	ParseStatus  string   `json:"parseStatus"`
	Dependencies int      `json:"dependencies"`
	Scripts      int      `json:"scripts"`
	Issues       []string `json:"issues,omitempty"`
}

type Change struct {
	Additions int64 `json:"additions"`
	Deletions int64 `json:"deletions"`
}

type DirectoryShare struct {
	Name      string `json:"name"`
	Additions int64  `json:"additions"`
	Deletions int64  `json:"deletions"`
}

type CommitSummary struct {
	SHA       string `json:"sha"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Additions int64  `json:"additions"`
	Deletions int64  `json:"deletions"`
}

type TimelineBucket struct {
	Period       string            `json:"period"`
	Additions    int64             `json:"additions"`
	Deletions    int64             `json:"deletions"`
	Commits      int64             `json:"commits"`
	Contributors map[string]Change `json:"contributors,omitempty"`
}

type CategorySummary struct {
	Category  Category `json:"category"`
	Additions int64    `json:"additions"`
	Deletions int64    `json:"deletions"`
	Files     int64    `json:"files"`
}

type Progress struct {
	Phase   string  `json:"phase"`
	Message string  `json:"message"`
	Current int64   `json:"current"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
}
