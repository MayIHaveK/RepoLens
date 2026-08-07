package filter

import (
	"path"
	"strings"

	"github.com/repolens/repolens/internal/config"
	"github.com/repolens/repolens/internal/model"
)

type Classification struct {
	Category model.Category
	Ignored  bool
	Reason   string
}

type Classifier struct {
	ignored   Matcher
	generated Matcher
	vendored  Matcher
}

func NewClassifier(cfg config.Config) Classifier {
	return Classifier{
		ignored:   NewMatcher(cfg.IgnoredPatterns),
		generated: NewMatcher(cfg.GeneratedPatterns),
		vendored:  NewMatcher(cfg.VendoredPatterns),
	}
}

func (c Classifier) Classify(filePath string) Classification {
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	if c.ignored.Match(filePath) {
		return Classification{Ignored: true, Reason: "ignored"}
	}
	if c.generated.Match(filePath) {
		return Classification{Ignored: true, Reason: "generated"}
	}
	if c.vendored.Match(filePath) {
		return Classification{Ignored: true, Reason: "vendored"}
	}

	base := strings.ToLower(path.Base(filePath))
	ext := strings.ToLower(path.Ext(base))
	if isBinaryExtension(ext) {
		return Classification{Category: model.CategoryAsset, Ignored: true, Reason: "binary"}
	}
	if isTestPath(filePath, base) {
		return Classification{Category: model.CategoryTest}
	}
	if isDocumentation(base, ext) {
		return Classification{Category: model.CategoryDocs}
	}
	if isConfiguration(base, ext) {
		return Classification{Category: model.CategoryConfig}
	}
	if isSource(ext) {
		return Classification{Category: model.CategorySource}
	}
	if isAsset(ext) {
		return Classification{Category: model.CategoryAsset}
	}
	return Classification{Category: model.CategoryOther}
}

func isTestPath(filePath, base string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, "/test/") || strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/__tests__/") || strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, ".test.") || strings.Contains(base, ".spec.")
}

func isDocumentation(base, ext string) bool {
	if strings.HasPrefix(base, "readme") || strings.HasPrefix(base, "license") ||
		strings.HasPrefix(base, "changelog") || strings.HasPrefix(base, "contributing") {
		return true
	}
	return contains([]string{".md", ".mdx", ".rst", ".adoc", ".txt"}, ext)
}

func isConfiguration(base, ext string) bool {
	if contains([]string{"dockerfile", "makefile", "justfile", ".gitignore", ".gitattributes", ".editorconfig"}, base) {
		return true
	}
	return contains([]string{".json", ".yaml", ".yml", ".toml", ".ini", ".properties", ".xml", ".gradle", ".conf"}, ext)
}

func isSource(ext string) bool {
	return contains([]string{
		".go", ".rs", ".java", ".kt", ".kts", ".c", ".h", ".cc", ".cpp", ".hpp",
		".cs", ".fs", ".vb", ".py", ".rb", ".php", ".swift", ".m", ".mm",
		".js", ".jsx", ".ts", ".tsx", ".vue", ".svelte", ".html", ".css", ".scss",
		".sql", ".sh", ".bash", ".ps1", ".lua", ".dart", ".scala", ".clj", ".ex", ".exs",
	}, ext)
}

func isAsset(ext string) bool {
	return contains([]string{".svg", ".glsl", ".vert", ".frag", ".obj", ".mtl"}, ext)
}

func isBinaryExtension(ext string) bool {
	return contains([]string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".bmp", ".tiff",
		".mp3", ".wav", ".ogg", ".flac", ".mp4", ".mov", ".avi", ".webm",
		".zip", ".7z", ".rar", ".gz", ".jar", ".war", ".dll", ".exe", ".so", ".dylib",
		".pdf", ".woff", ".woff2", ".ttf", ".otf", ".class", ".pyc",
	}, ext)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
