package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/MayIHaveK/RepoLens/internal/model"
)

type Config struct {
	Ref                 string                     `json:"ref"`
	Since               string                     `json:"since,omitempty"`
	Until               string                     `json:"until,omitempty"`
	Profile             string                     `json:"profile"`
	Parallelism         int                        `json:"parallelism"`
	MaxFileSizeBytes    int64                      `json:"maxFileSizeBytes"`
	MaxOwnershipFiles   int                        `json:"maxOwnershipFiles"`
	MaxInventoryFiles   int                        `json:"maxInventoryFiles"`
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
		MaxInventoryFiles:   50_000,
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
			"**/target/**", "**/.idea/**", "**/.vscode/**", "**/coverage/**", "**/*.min.js",
			"**/*.min.css", "**/*.map", "**/*.lock", "**/package-lock.json", "**/pnpm-lock.yaml",
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
	if c.MaxInventoryFiles <= 0 {
		c.MaxInventoryFiles = defaults.MaxInventoryFiles
	}
	if c.Weights.Workload+c.Weights.Ownership+c.Weights.Retention+c.Weights.Collaboration <= 0 {
		c.Weights = defaults.Weights
	}
	weights := make(map[model.Category]float64, len(defaults.CategoryWeights))
	for category, value := range defaults.CategoryWeights {
		weights[category] = value
	}
	for category, value := range c.CategoryWeights {
		weights[category] = value
	}
	c.CategoryWeights = weights
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
	aliases := make(map[string]string, len(c.Aliases))
	keys := make([]string, 0, len(c.Aliases))
	for key := range c.Aliases {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		aliases[strings.ToLower(strings.TrimSpace(key))] = strings.ToLower(strings.TrimSpace(c.Aliases[key]))
	}
	c.Aliases = aliases
}

// UnmarshalJSON overlays supplied fields on defaults, preserving explicit false/zero values.
func (c *Config) UnmarshalJSON(data []byte) error {
	type plain Config
	value := plain(Default())
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*c = Config(value)
	return nil
}

func (c Config) Validate() error {
	if c.Weights.Workload+c.Weights.Ownership+c.Weights.Retention+c.Weights.Collaboration <= 0 {
		return fmt.Errorf("至少设置一个大于零的贡献权重")
	}
	if c.Profile != "" && c.Profile != "fast" && c.Profile != "balanced" && c.Profile != "thorough" {
		return fmt.Errorf("未知性能模式")
	}
	for _, weight := range []float64{c.Weights.Workload, c.Weights.Ownership, c.Weights.Retention, c.Weights.Collaboration} {
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 || weight > 1 {
			return fmt.Errorf("贡献权重必须在 0 到 1 之间")
		}
	}
	for category, weight := range c.CategoryWeights {
		if _, ok := Default().CategoryWeights[category]; !ok {
			return fmt.Errorf("未知代码分类 %q", category)
		}
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 || weight > 1 {
			return fmt.Errorf("代码类型权重必须在 0 到 1 之间")
		}
	}
	for source, target := range c.Aliases {
		if strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" {
			return fmt.Errorf("身份别名不能包含空身份")
		}
	}
	if c.Parallelism < 0 || c.Parallelism > 32 || c.MaxFileSizeBytes < 0 || c.MaxFileSizeBytes > 100*1024*1024 || c.MaxOwnershipFiles < 0 || c.MaxOwnershipFiles > 1_000_000 || c.MaxInventoryFiles < 0 || c.MaxInventoryFiles > 1_000_000 {
		return fmt.Errorf("性能限制超出范围：并行 1–32，文件至多 100 MB，文件数至多 100 万")
	}
	var since, until time.Time
	var err error
	if c.Since != "" {
		since, err = ParseDate(c.Since)
		if err != nil {
			return fmt.Errorf("开始日期必须是 YYYY-MM-DD 或 RFC3339")
		}
	}
	if c.Until != "" {
		until, err = ParseDate(c.Until)
		if err != nil {
			return fmt.Errorf("结束日期必须是 YYYY-MM-DD 或 RFC3339")
		}
	}
	if !since.IsZero() && !until.IsZero() && since.After(until) {
		return fmt.Errorf("开始日期不能晚于结束日期")
	}
	return nil
}

func ParseDate(value string) (time.Time, error) {
	if len(value) == 10 {
		return time.Parse("2006-01-02", value)
	}
	return time.Parse(time.RFC3339, value)
}

// Prepared is the single canonical configuration used by caching and analysis.
func (c Config) Prepared() (Config, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	c.Normalize()
	if c.Profile == "fast" {
		c.EnableOwnership = false
		c.EnableRetention = false
		c.DetectRenames = false
	}
	if c.Profile == "thorough" {
		c.MaxOwnershipFiles = 1_000_000
	}
	if !c.EnableOwnership || c.Since != "" || c.Until != "" {
		c.EnableRetention = false
	}
	return c, nil
}

func (c Config) Fingerprint() string {
	data, _ := json.Marshal(c)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func (c Config) Snapshot() model.ConfigSnapshot {
	effective, _ := json.Marshal(c)
	return model.ConfigSnapshot{
		Effective:        effective,
		Profile:          c.Profile,
		Weights:          c.Weights,
		OwnershipEnabled: c.EnableOwnership,
		GitHubEnabled:    false,
		Privacy:          c.Privacy,
		Fingerprint:      c.Fingerprint(),
		CategoryWeights:  c.CategoryWeights,
	}
}
