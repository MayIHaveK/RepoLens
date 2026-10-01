package analysis

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MayIHaveK/RepoLens/internal/model"
)

var importPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)(?:\bimport|\bexport)\s+(?:[^;\n]*?\s+from\s*)?["']([^"'\r\n]+)["']`),
	regexp.MustCompile(`\b(?:require|import)\s*\(\s*["']([^"'\r\n]+)["']\s*\)`),
}
var cssImport = regexp.MustCompile(`@import\s+(?:url\(\s*)?["']([^"'\r\n]+)["']`)
var htmlReference = regexp.MustCompile(`\b(?:src|href)\s*=\s*["']([^"'\r\n]+)["']`)
var topLevelKey = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_.-]*)\s*[:=]`)

func inspectReferences(filePath string, data []byte, category model.Category) ([]model.Dependency, *model.ConfigurationInfo, string) {
	edges := []model.Dependency{}
	ext, base := strings.ToLower(path.Ext(filePath)), strings.ToLower(path.Base(filePath))
	add := func(value, kind string) {
		edges = append(edges, model.Dependency{From: filePath, To: value, Kind: kind})
	}
	if ext == ".go" {
		file, _ := parser.ParseFile(token.NewFileSet(), filePath, data, parser.ImportsOnly)
		if file != nil {
			for _, item := range file.Imports {
				if value, err := strconv.Unquote(item.Path.Value); err == nil {
					add(value, "go-import")
				}
			}
		}
	}
	if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".vue" || ext == ".svelte" {
		text := stripJSONComments(data) // Only remove comments; keep string references intact.
		for _, pattern := range importPatterns {
			for _, match := range pattern.FindAllSubmatch(text, -1) {
				add(string(match[1]), "import")
			}
		}
	}
	if ext == ".css" || ext == ".scss" {
		for _, match := range cssImport.FindAllSubmatch(data, -1) {
			add(string(match[1]), "style-import")
		}
	}
	if ext == ".html" {
		for _, match := range htmlReference.FindAllSubmatch(data, -1) {
			value := string(match[1])
			if !strings.Contains(value, ":") && !strings.HasPrefix(value, "#") && !strings.HasPrefix(value, "//") {
				add(value, "asset-reference")
			}
		}
	}
	if category != model.CategoryConfig && base != "go.mod" {
		return edges, nil, ""
	}
	info := &model.ConfigurationInfo{Path: filePath, Format: strings.TrimPrefix(ext, "."), ParseStatus: "structure-only", Keys: []string{}, Issues: []string{}}
	if base == "go.mod" {
		info.Format = "go.mod"
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "module" {
				return edges, info, strings.Trim(fields[1], `"`)
			}
		}
		info.Issues = append(info.Issues, "缺少 module 声明")
		return edges, info, ""
	}
	if ext == ".json" {
		payload := data
		if strings.HasPrefix(base, "tsconfig") {
			payload = stripTrailingCommas(stripJSONComments(data))
		}
		var value map[string]json.RawMessage
		if json.Unmarshal(payload, &value) != nil || value == nil {
			info.ParseStatus = "invalid"
			info.Issues = append(info.Issues, "JSON 必须是有效的对象")
			return edges, info, ""
		}
		info.ParseStatus = "valid"
		for key := range value {
			info.Keys = append(info.Keys, key)
		}
		sort.Strings(info.Keys)
		if hasDuplicateJSONKeys(payload) {
			info.Issues = append(info.Issues, "存在重复键，后面的值会覆盖前面的值")
		}
		if base == "package.json" {
			for _, key := range []string{"dependencies", "devDependencies", "peerDependencies", "optionalDependencies"} {
				var items map[string]json.RawMessage
				_ = json.Unmarshal(value[key], &items)
				info.Dependencies += len(items)
			}
			var scripts map[string]string
			_ = json.Unmarshal(value["scripts"], &scripts)
			info.Scripts = len(scripts)
		}
		if strings.HasPrefix(base, "tsconfig") {
			var extends string
			if json.Unmarshal(value["extends"], &extends) == nil && extends != "" {
				add(extends, "config-extends")
			}
			var parents []string
			if json.Unmarshal(value["extends"], &parents) == nil {
				for _, parent := range parents {
					add(parent, "config-extends")
				}
			}
			var references []struct {
				Path string `json:"path"`
			}
			_ = json.Unmarshal(value["references"], &references)
			for _, reference := range references {
				if reference.Path != "" {
					add(reference.Path, "config-reference")
				}
			}
		}
	} else if ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".properties" || ext == ".conf" {
		seen := map[string]bool{}
		section := ""
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if (ext == ".toml" || ext == ".ini") && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				section = strings.Trim(trimmed, "[] ")
				continue
			}
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			match := topLevelKey.FindStringSubmatch(line)
			if len(match) == 2 {
				key := match[1]
				if section != "" {
					key = section + "." + key
				}
				if seen[key] {
					info.Issues = append(info.Issues, "存在重复顶层键")
				}
				seen[key] = true
			}
		}
		for key := range seen {
			info.Keys = append(info.Keys, key)
		}
		sort.Strings(info.Keys)
	}
	return edges, info, ""
}

func stripJSONComments(data []byte) []byte {
	result := append([]byte(nil), data...)
	var quote byte
	for i := 0; i < len(result); i++ {
		if quote != 0 {
			if result[i] == '\\' {
				i++
				continue
			}
			if result[i] == quote {
				quote = 0
			}
			continue
		}
		if result[i] == '"' || result[i] == '\'' || result[i] == '`' {
			quote = result[i]
			continue
		}
		if i+1 < len(result) && result[i] == '/' && result[i+1] == '/' {
			for i < len(result) && result[i] != '\n' {
				result[i] = ' '
				i++
			}
			continue
		}
		if i+1 < len(result) && result[i] == '/' && result[i+1] == '*' {
			result[i] = ' '
			i++
			result[i] = ' '
			i++
			for i < len(result) {
				if i+1 < len(result) && result[i] == '*' && result[i+1] == '/' {
					result[i] = ' '
					i++
					result[i] = ' '
					break
				}
				if result[i] != '\n' && result[i] != '\r' {
					result[i] = ' '
				}
				i++
			}
		}
	}
	return result
}

func stripTrailingCommas(data []byte) []byte {
	result := append([]byte(nil), data...)
	quoted := false
	for i := 0; i < len(result); i++ {
		if quoted && result[i] == '\\' {
			i++
			continue
		}
		if result[i] == '"' {
			quoted = !quoted
			continue
		}
		if !quoted && result[i] == ',' {
			j := i + 1
			for j < len(result) && bytes.ContainsRune([]byte(" \r\n\t"), rune(result[j])) {
				j++
			}
			if j < len(result) && (result[j] == '}' || result[j] == ']') {
				result[i] = ' '
			}
		}
	}
	return result
}

func hasDuplicateJSONKeys(data []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() (bool, error)
	walk = func() (bool, error) {
		value, err := decoder.Token()
		if err != nil {
			return false, err
		}
		delimiter, ok := value.(json.Delim)
		if !ok {
			return false, nil
		}
		duplicate := false
		if delimiter == '{' {
			keys := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return false, err
				}
				text, _ := key.(string)
				if keys[text] {
					duplicate = true
				}
				keys[text] = true
				nested, err := walk()
				if err != nil {
					return false, err
				}
				duplicate = duplicate || nested
			}
		} else if delimiter == '[' {
			for decoder.More() {
				nested, err := walk()
				if err != nil {
					return false, err
				}
				duplicate = duplicate || nested
			}
		}
		_, err = decoder.Token()
		return duplicate, err
	}
	duplicate, err := walk()
	return err == nil && duplicate
}

func resolveReferences(references []model.Dependency, files []model.FileMetric, module string) ([]model.Dependency, [][]string) {
	paths := map[string]bool{}
	packages := map[string][]string{}
	for _, file := range files {
		paths[file.Path] = true
		if strings.HasSuffix(file.Path, ".go") && !strings.HasSuffix(file.Path, "_test.go") {
			directory := path.Dir(file.Path)
			packages[directory] = append(packages[directory], file.Path)
		}
	}
	edges := []model.Dependency{}
	seen := map[string]bool{}
	add := func(edge model.Dependency) {
		key := edge.From + "\x00" + edge.To + "\x00" + edge.Kind
		if !seen[key] {
			seen[key] = true
			edges = append(edges, edge)
		}
	}
	for _, reference := range references {
		target := reference.To
		if reference.Kind == "go-import" {
			if module != "" && (target == module || strings.HasPrefix(target, module+"/")) {
				directory := strings.TrimPrefix(strings.TrimPrefix(target, module), "/")
				if directory == "" {
					directory = "."
				}
				// A Go import points to a package. One edge per package avoids expanding
				// an import into every source file, which would create a quadratic graph.
				reference.From = path.Dir(reference.From)
				reference.To = directory
				reference.Kind = "go-package"
				reference.Resolved = len(packages[directory]) > 0
			}
			add(reference)
			continue
		}
		local := strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/") || reference.Kind == "config-reference" || reference.Kind == "asset-reference" || reference.Kind == "style-import"
		if !local {
			add(reference)
			continue
		}
		target = strings.SplitN(strings.SplitN(target, "?", 2)[0], "#", 2)[0]
		if strings.HasPrefix(target, "/") {
			root := "."
			if reference.Kind == "asset-reference" {
				root = path.Dir(reference.From)
				for directory := root; ; directory = path.Dir(directory) {
					if paths[path.Join(directory, "package.json")] {
						root = directory
						break
					}
					if directory == "." {
						break
					}
				}
			}
			target = path.Join(root, strings.TrimPrefix(target, "/"))
		} else {
			target = path.Join(path.Dir(reference.From), target)
		}
		candidates := []string{target}
		if reference.Kind == "config-extends" || reference.Kind == "config-reference" {
			candidates = append(candidates, target+".json", path.Join(target, "tsconfig.json"))
		} else {
			for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".json", ".css", ".scss", ".vue", ".svelte"} {
				candidates = append(candidates, target+ext, path.Join(target, "index"+ext))
			}
			if strings.HasSuffix(target, ".js") {
				candidates = append(candidates, strings.TrimSuffix(target, ".js")+".ts", strings.TrimSuffix(target, ".js")+".tsx")
			}
		}
		reference.To = target
		for _, candidate := range candidates {
			if paths[candidate] {
				reference.To = candidate
				reference.Resolved = true
				break
			}
		}
		add(reference)
	}
	sort.Slice(edges, func(i, j int) bool {
		left, right := edges[i], edges[j]
		if left.From != right.From {
			return left.From < right.From
		}
		if left.To != right.To {
			return left.To < right.To
		}
		return left.Kind < right.Kind
	})
	return edges, dependencyCycles(edges)
}

// Tarjan SCC reports deterministic cycle groups instead of exponentially many paths.
func dependencyCycles(edges []model.Dependency) [][]string {
	graph := map[string][]string{}
	self := map[string]bool{}
	for _, edge := range edges {
		if !edge.Resolved {
			continue
		}
		prefix := "file:"
		if edge.Kind == "go-package" {
			prefix = "package:"
		}
		from, to := prefix+edge.From, prefix+edge.To
		graph[from] = append(graph[from], to)
		if from == to {
			self[from] = true
		}
	}
	indexes, low, active := map[string]int{}, map[string]int{}, map[string]bool{}
	stack := []string{}
	next := 0
	result := [][]string{}
	var visit func(string)
	visit = func(node string) {
		next++
		indexes[node] = next
		low[node] = next
		stack = append(stack, node)
		active[node] = true
		for _, target := range graph[node] {
			if indexes[target] == 0 {
				visit(target)
				low[node] = min(low[node], low[target])
			} else if active[target] {
				low[node] = min(low[node], indexes[target])
			}
		}
		if low[node] != indexes[node] {
			return
		}
		group := []string{}
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			active[last] = false
			group = append(group, strings.SplitN(last, ":", 2)[1])
			if last == node {
				break
			}
		}
		if len(group) > 1 || self[node] {
			sort.Strings(group)
			result = append(result, group)
		}
	}
	nodes := make([]string, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if indexes[node] == 0 {
			visit(node)
		}
	}
	sort.Slice(result, func(i, j int) bool { return strings.Join(result[i], "\x00") < strings.Join(result[j], "\x00") })
	return result
}
