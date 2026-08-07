package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"runtime"
	"strings"

	"github.com/repolens/repolens/internal/model"
)

type Config struct {
	Ref                 string                     `json:"ref"`
	Since               string                     `json:"since,omitempty"`
	Until               string                     `json:"until,omitempty"`
	Profile             string                     `json:"profile"`
	Parallelism         int                        `json:"parallelism"`
	MaxFileSizeBytes    int64                      `json:"maxFileSizeBytes"`
	MaxOwnershipFiles   int                        `json:"maxOwnershipFiles"`
	EnableOwnership     bool                       `json:"enableOwnership"`
	EnableRetention     bool                       `json:"enableRetention"`
	EnableCollaboration bool                       `json:"enableCollaboration"`
	DetectRenames       bool                       `json:"detectRenames"`
	IncludeBots         bool                       `json:"includeBots"`
	IncludeMerges       bool                       `json:"includeMerges"`
	Weights             model.Weights              `json:"weights"`
	CategoryWeights     map[model.Category]float64 `json:"categoryWeights"`
	Privacy             model.Privacy              `json:"privacy"`
	IgnoredPatterns     []string                   `json:"ignoredPatterns"`
	GeneratedPatterns   []string                   `json:"generatedPatterns"`
	VendoredPatterns    []string                   `json:"vendoredPatterns"`
	BotPatterns         []string                   `json:"botPatterns"`
	Aliases             map[string]string          `json:"aliases"`
}

func Default() Config {
	parallelism := runtime.NumCPU() / 2
	if parallelism < 2 {
		parallelism = 2
	}
	if parallelism > 8 {
		parallelism = 8
	}

	return Config{
		Ref:                 "HEAD",
		Profile:             "balanced",
		Parallelism:         parallelism,
		MaxFileSizeBytes:    2 * 1024 * 1024,
		MaxOwnershipFiles:   12_000,
		EnableOwnership:     true,
		EnableRetention:     true,
		EnableCollaboration: true,
		DetectRenames:       true,
		Weights: model.Weights{
			Workload:      0.35,
			Ownership:     0.35,
			Retention:     0.20,
			Collaboration: 0.10,
		},
		CategoryWeights: map[model.Category]float64{
			model.CategorySource: 1.00,
			model.CategoryTest:   0.90,
			model.CategoryDocs:   0.45,
			model.CategoryConfig: 0.65,
			model.CategoryAsset:  0.25,
			model.CategoryOther:  0.40,
		},
		Privacy: model.Privacy{
			ShowRepositoryName: true,
			ShowCommitMessages: true,
			ShowDirectories:    true,
			ShowContributors:   true,
			ShowAvatars:        true,
			ShowCommitSHA:      true,
		},
		IgnoredPatterns: []string{
			".git/**", "**/node_modules/**", "**/vendor/**", "**/dist/**", "**/build/**",
			"**/target/**", "**/.idea/**", "**/.vscode/**", "**/coverage/**", "*.min.js",
			"*.min.css", "*.map", "*.lock", "package-lock.json", "pnpm-lock.yaml",
		},
		GeneratedPatterns: []string{
			"**/*.generated.*", "**/*_generated.*", "**/*.g.cs", "**/*.designer.cs",
			"**/generated/**", "**/gen/**",
		},
		VendoredPatterns: []string{
			"**/vendor/**", "**/third_party/**", "**/third-party/**", "**/external/**", "**/deps/**",
		},
		BotPatterns: []string{"[bot]", "dependabot", "renovate", "github-actions"},
		Aliases:     map[string]string{},
	}
}

func (c *Config) Normalize() {
	defaults := Default()
	if strings.TrimSpace(c.Ref) == "" {
		c.Ref = defaults.Ref
	}
	if c.Profile == "" {
		c.Profile = defaults.Profile
	}
	if c.Parallelism <= 0 {
		c.Parallelism = defaults.Parallelism
	}
	if c.Parallelism > 32 {
		c.Parallelism = 32
	}
	if c.MaxFileSizeBytes <= 0 {
		c.MaxFileSizeBytes = defaults.MaxFileSizeBytes
	}
	if c.MaxOwnershipFiles <= 0 {
		c.MaxOwnershipFiles = defaults.MaxOwnershipFiles
	}
	if c.Weights.Workload+c.Weights.Ownership+c.Weights.Retention+c.Weights.Collaboration <= 0 {
		c.Weights = defaults.Weights
	}
	if len(c.CategoryWeights) == 0 {
		c.CategoryWeights = defaults.CategoryWeights
	}
	if c.IgnoredPatterns == nil {
		c.IgnoredPatterns = defaults.IgnoredPatterns
	}
	if c.GeneratedPatterns == nil {
		c.GeneratedPatterns = defaults.GeneratedPatterns
	}
	if c.VendoredPatterns == nil {
		c.VendoredPatterns = defaults.VendoredPatterns
	}
	if c.BotPatterns == nil {
		c.BotPatterns = defaults.BotPatterns
	}
	if c.Aliases == nil {
		c.Aliases = map[string]string{}
	}
}

func (c Config) Fingerprint() string {
	data, _ := json.Marshal(c)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func (c Config) Snapshot() model.ConfigSnapshot {
	return model.ConfigSnapshot{
		Profile:          c.Profile,
		Weights:          c.Weights,
		OwnershipEnabled: c.EnableOwnership,
		GitHubEnabled:    false,
		Privacy:          c.Privacy,
		Fingerprint:      c.Fingerprint(),
		CategoryWeights:  c.CategoryWeights,
	}
}
