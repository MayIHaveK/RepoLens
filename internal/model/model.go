package model

import "time"

const SchemaVersion = 1

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
	SchemaVersion int               `json:"schemaVersion"`
	ID            string            `json:"id"`
	Repository    Repository        `json:"repository"`
	GeneratedAt   time.Time         `json:"generatedAt"`
	DurationMS    int64             `json:"durationMs"`
	Mode          string            `json:"mode"`
	Config        ConfigSnapshot    `json:"config"`
	Summary       Summary           `json:"summary"`
	Contributors  []Contributor     `json:"contributors"`
	Timeline      []TimelineBucket  `json:"timeline"`
	Categories    []CategorySummary `json:"categories"`
	Warnings      []string          `json:"warnings,omitempty"`
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
