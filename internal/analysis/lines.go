package analysis

import (
	"bytes"
	"path"
	"strings"

	"github.com/MayIHaveK/RepoLens/internal/model"
)

func languageFor(filePath string) string {
	base := strings.ToLower(path.Base(filePath))
	if base == "dockerfile" {
		return "Dockerfile"
	}
	if base == "makefile" {
		return "Makefile"
	}
	if base == "go.mod" || base == "go.sum" {
		return "Go module"
	}
	languages := map[string]string{
		".go": "Go", ".ts": "TypeScript", ".tsx": "TSX", ".js": "JavaScript", ".jsx": "JSX",
		".java": "Java", ".kt": "Kotlin", ".kts": "Kotlin", ".rs": "Rust", ".py": "Python",
		".c": "C", ".h": "C", ".cpp": "C++", ".cc": "C++", ".hpp": "C++", ".cs": "C#",
		".css": "CSS", ".scss": "SCSS", ".html": "HTML", ".xml": "XML", ".svg": "SVG",
		".json": "JSON", ".yaml": "YAML", ".yml": "YAML", ".toml": "TOML", ".ini": "INI",
		".properties": "Properties", ".md": "Markdown", ".mdx": "MDX", ".rst": "Text",
		".ps1": "PowerShell", ".sh": "Shell", ".bash": "Shell", ".cmd": "Batch", ".bat": "Batch",
		".sql": "SQL", ".lua": "Lua", ".rb": "Ruby", ".php": "PHP", ".swift": "Swift",
		".vue": "Vue", ".svelte": "Svelte", ".gradle": "Groovy", ".conf": "Configuration",
	}
	if language, ok := languages[strings.ToLower(path.Ext(filePath))]; ok {
		return language
	}
	return "Text"
}

// countLines uses lexical comment/string states. Mixed comment/code lines count
// as code. Unsupported formats count nonblank content; this is not an AST metric.
func countLines(data []byte, language string) model.LineCounts {
	var result model.LineCounts
	if len(data) == 0 {
		return result
	}
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	hash := strings.Contains("|Python|Shell|PowerShell|Ruby|YAML|TOML|INI|Properties|Makefile|Dockerfile|Configuration|", "|"+language+"|")
	slash := strings.Contains("|Go|TypeScript|TSX|JavaScript|JSX|Java|Kotlin|Rust|C|C++|C#|SCSS|PHP|Swift|Vue|Svelte|Groovy|", "|"+language+"|")
	block := slash || language == "CSS" || language == "SQL"
	markup := language == "HTML" || language == "XML" || language == "SVG" || language == "Vue" || language == "Svelte" || language == "Markdown"
	var blockEnd, quote string
	for _, raw := range lines {
		line := strings.TrimSuffix(string(raw), "\r")
		result.Total++
		if strings.TrimSpace(line) == "" {
			result.Blank++
			continue
		}
		code, comment := false, false
		if language == "Batch" {
			lower := strings.ToLower(strings.TrimSpace(line))
			if strings.HasPrefix(lower, "::") || lower == "rem" || strings.HasPrefix(lower, "rem ") || strings.HasPrefix(lower, "@rem ") {
				result.Comment++
				continue
			}
		}
		for i := 0; i < len(line); {
			rest := line[i:]
			if blockEnd != "" {
				comment = true
				if end := strings.Index(rest, blockEnd); end >= 0 {
					i += end + len(blockEnd)
					blockEnd = ""
					continue
				}
				break
			}
			if quote != "" {
				code = true
				if strings.HasPrefix(rest, quote) {
					i += len(quote)
					quote = ""
					continue
				}
				if line[i] == '\\' && quote != "`" && i+1 < len(line) {
					i += 2
				} else {
					i++
				}
				continue
			}
			if line[i] == ' ' || line[i] == '\t' {
				i++
				continue
			}
			if markup && strings.HasPrefix(rest, "<!--") {
				blockEnd = "-->"
				comment = true
				i += 4
				continue
			}
			if block && strings.HasPrefix(rest, "/*") {
				blockEnd = "*/"
				comment = true
				i += 2
				continue
			}
			if (slash && strings.HasPrefix(rest, "//")) || (hash && line[i] == '#') || ((language == "SQL" || language == "Lua") && strings.HasPrefix(rest, "--")) || (language == "INI" && line[i] == ';') {
				comment = true
				break
			}
			code = true
			if language == "Python" && (strings.HasPrefix(rest, `"""`) || strings.HasPrefix(rest, "'''")) {
				quote = rest[:3]
				i += 3
				continue
			}
			if line[i] == '"' || line[i] == '\'' || (line[i] == '`' && language != "PowerShell") {
				quote = rest[:1]
			}
			i++
		}
		if code {
			result.Code++
		} else if comment {
			result.Comment++
		} else {
			result.Code++
		}
	}
	return result
}

func addLines(to *model.LineCounts, from model.LineCounts) {
	to.Total += from.Total
	to.Code += from.Code
	to.Comment += from.Comment
	to.Blank += from.Blank
}
