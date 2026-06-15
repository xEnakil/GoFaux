package projectscan

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	maxScanFiles     = 1500
	maxScanFileBytes = 2 * 1024 * 1024
)

type Preview struct {
	Root         string        `json:"root"`
	ScannedFiles int           `json:"scanned_files"`
	Integrations []Integration `json:"integrations"`
	Messages     []string      `json:"messages,omitempty"`
}

type Integration struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Method      string   `json:"method"`
	Endpoint    string   `json:"endpoint"`
	BaseURL     string   `json:"base_url,omitempty"`
	SourceFile  string   `json:"source_file"`
	Line        int      `json:"line"`
	Kind        string   `json:"kind"`
	Direction   string   `json:"direction"`
	External    bool     `json:"external"`
	Confidence  float64  `json:"confidence"`
	RequestDTO  string   `json:"request_dto,omitempty"`
	ResponseDTO string   `json:"response_dto,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

var (
	axiosPattern       = regexp.MustCompile(`(?i)\baxios\.(get|post|put|patch|delete)\s*\(\s*["']([^"']+)["']`)
	fetchPattern       = regexp.MustCompile(`\bfetch\s*\(\s*["']([^"']+)["']`)
	httpClientPattern  = regexp.MustCompile(`(?i)\b(?:this\.)?(?:http|httpClient)\.(get|post|put|patch|delete)\s*(?:<[^>]+>)?\s*\(\s*["']([^"']+)["']`)
	expressRoute       = regexp.MustCompile(`(?i)\b(?:app|router)\.(get|post|put|patch|delete)\s*\(\s*["']([^"']+)["']`)
	goHTTPPattern      = regexp.MustCompile(`\bhttp\.(Get|Post)\s*\(\s*["']([^"']+)["']`)
	goNewRequest       = regexp.MustCompile(`\bhttp\.NewRequest\s*\(\s*["']([A-Z]+)["']\s*,\s*["']([^"']+)["']`)
	goNewRequestCtx    = regexp.MustCompile(`\bhttp\.NewRequestWithContext\s*\([^,]+,\s*["']([A-Z]+)["']\s*,\s*["']([^"']+)["']`)
	goRoutePattern     = regexp.MustCompile(`\b(?:HandleFunc|Handle)\s*\(\s*["']([^"']+)["']`)
	pythonHTTPPattern  = regexp.MustCompile(`(?i)\b(?:requests|httpx)\.(get|post|put|patch|delete)\s*\(\s*["']([^"']+)["']`)
	configURLPattern   = regexp.MustCompile(`(?i)([a-z0-9_.-]*(?:url|uri|endpoint|base-url|base_url)[a-z0-9_.-]*)\s*[:=]\s*["']?([^"'\s#]+)`)
	restTemplate       = regexp.MustCompile(`(?i)\.(getForObject|getForEntity|postForObject|postForEntity|exchange|put|delete)\s*\(\s*["']([^"']+)["']`)
	webClientURI       = regexp.MustCompile(`(?i)\.uri\s*\(\s*["']([^"']+)["']`)
	javaFeignClient    = regexp.MustCompile(`@FeignClient\s*\(([^)]*)\)`)
	javaMapping        = regexp.MustCompile(`@(Get|Post|Put|Patch|Delete)Mapping\s*(?:\(([^)]*)\))?`)
	javaRequestMapping = regexp.MustCompile(`@RequestMapping\s*(?:\(([^)]*)\))?`)
	javaAttr           = regexp.MustCompile(`(?i)(name|value|url|path|method)\s*=\s*(?:RequestMethod\.)?["']?([^"',) ]+)`)
	javaStringLiteral  = regexp.MustCompile(`["']([^"']+)["']`)
)

func Scan(root string) (Preview, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Preview{}, errors.New("project folder is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Preview{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Preview{}, err
	}
	if !info.IsDir() {
		return Preview{}, fmt.Errorf("%s is not a directory", abs)
	}

	scanner := scanner{root: abs, seen: map[string]bool{}}
	err = filepath.WalkDir(abs, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			scanner.messages = append(scanner.messages, walkErr.Error())
			return nil
		}
		if path == abs {
			return nil
		}
		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if scanner.scannedFiles >= maxScanFiles {
			return filepath.SkipAll
		}
		if !isScannableFile(path) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			scanner.messages = append(scanner.messages, err.Error())
			return nil
		}
		if info.Size() > maxScanFileBytes {
			scanner.messages = append(scanner.messages, "skipped large file: "+scanner.rel(path))
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			scanner.messages = append(scanner.messages, err.Error())
			return nil
		}
		scanner.scannedFiles++
		scanner.scanFile(path, string(data))
		return nil
	})
	if err != nil {
		return Preview{}, err
	}
	if scanner.scannedFiles >= maxScanFiles {
		scanner.messages = append(scanner.messages, fmt.Sprintf("scan stopped after %d files to keep analysis local and fast", maxScanFiles))
	}
	sort.SliceStable(scanner.integrations, func(i, j int) bool {
		left, right := scanner.integrations[i], scanner.integrations[j]
		if left.External != right.External {
			return left.External
		}
		if left.SourceFile != right.SourceFile {
			return left.SourceFile < right.SourceFile
		}
		return left.Line < right.Line
	})
	return Preview{
		Root:         abs,
		ScannedFiles: scanner.scannedFiles,
		Integrations: scanner.integrations,
		Messages:     compact(scanner.messages),
	}, nil
}

type scanner struct {
	root         string
	scannedFiles int
	integrations []Integration
	messages     []string
	seen         map[string]bool
}

func (s *scanner) scanFile(path, content string) {
	rel := s.rel(path)
	lines := strings.Split(content, "\n")
	ext := strings.ToLower(filepath.Ext(path))
	lowerContent := strings.ToLower(content)

	if ext == ".java" {
		s.scanJava(rel, lines, lowerContent)
	}
	if ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" || ext == ".mjs" || ext == ".cjs" {
		s.scanJavaScript(rel, lines)
	}
	if ext == ".go" {
		s.scanGo(rel, lines)
	}
	if ext == ".py" {
		s.scanPython(rel, lines)
	}
	if isConfigFile(path) {
		s.scanConfigURLs(rel, lines)
	}
}

func (s *scanner) scanJava(rel string, lines []string, lowerContent string) {
	hasFeign := strings.Contains(lowerContent, "@feignclient")
	hasController := strings.Contains(lowerContent, "@restcontroller") || strings.Contains(lowerContent, "@controller")

	if hasFeign {
		clientName, baseURL := "", ""
		for i, line := range lines {
			if !strings.Contains(line, "@FeignClient") {
				continue
			}
			annotation := collectAnnotation(lines, i)
			clientName = firstNonEmpty(attrValue(annotation, "name"), attrValue(annotation, "value"), clientName)
			baseURL = firstNonEmpty(attrValue(annotation, "url"), baseURL)
		}
		for i, line := range lines {
			method, endpoint, ok := javaMappingFromLine(line)
			if !ok {
				continue
			}
			responseDTO, requestDTO, methodName := javaMethodSignature(lines, i+1)
			name := firstNonEmpty(clientName, methodName, "Spring Feign integration")
			if methodName != "" && clientName != "" {
				name = clientName + "." + methodName
			}
			external := true
			confidence := 0.88
			if baseURL != "" && isAbsoluteURL(baseURL) {
				external = isExternalHost(baseURL)
				confidence = 0.95
			}
			s.addIntegration(Integration{
				Name:        name,
				Method:      method,
				Endpoint:    normalizeEndpoint(endpoint),
				BaseURL:     baseURLFromValue(baseURL),
				SourceFile:  rel,
				Line:        i + 1,
				Kind:        "spring-feign",
				Direction:   "client",
				External:    external,
				Confidence:  confidence,
				RequestDTO:  requestDTO,
				ResponseDTO: responseDTO,
				Evidence:    []string{strings.TrimSpace(line)},
				Tags:        []string{"project-scan", "spring", "feign"},
			})
		}
	}

	if hasController {
		prefix := ""
		for i, line := range lines {
			if strings.Contains(line, "@RequestMapping") && !strings.Contains(line, "method") {
				_, endpoint, ok := javaRequestMappingFromLine(line)
				if ok && endpoint != "" {
					prefix = endpoint
				}
				continue
			}
			method, endpoint, ok := javaMappingFromLine(line)
			if !ok {
				method, endpoint, ok = javaRequestMappingFromLine(line)
			}
			if !ok {
				continue
			}
			responseDTO, requestDTO, methodName := javaMethodSignature(lines, i+1)
			s.addIntegration(Integration{
				Name:        firstNonEmpty(methodName, "Spring controller route"),
				Method:      method,
				Endpoint:    joinEndpoint(prefix, endpoint),
				SourceFile:  rel,
				Line:        i + 1,
				Kind:        "spring-controller",
				Direction:   "server",
				External:    false,
				Confidence:  0.82,
				RequestDTO:  requestDTO,
				ResponseDTO: responseDTO,
				Evidence:    []string{strings.TrimSpace(line)},
				Tags:        []string{"project-scan", "spring", "internal-route"},
			})
		}
	}

	for i, line := range lines {
		if match := restTemplate.FindStringSubmatch(line); len(match) == 3 {
			method := methodFromRestTemplate(match[1])
			value := match[2]
			s.addClientURL(rel, i+1, "spring-http-client", method, value, line, 0.85)
		}
		if match := webClientURI.FindStringSubmatch(line); len(match) == 2 {
			method := nearbyWebClientMethod(lines, i)
			s.addClientURL(rel, i+1, "spring-webclient", method, match[1], line, 0.8)
		}
	}
}

func (s *scanner) scanJavaScript(rel string, lines []string) {
	for i, line := range lines {
		if match := axiosPattern.FindStringSubmatch(line); len(match) == 3 {
			s.addClientURL(rel, i+1, "axios", strings.ToUpper(match[1]), match[2], line, 0.9)
		}
		if match := httpClientPattern.FindStringSubmatch(line); len(match) == 3 {
			s.addClientURL(rel, i+1, "angular-http-client", strings.ToUpper(match[1]), match[2], line, 0.82)
		}
		if match := fetchPattern.FindStringSubmatch(line); len(match) == 2 {
			s.addClientURL(rel, i+1, "fetch", methodFromFetchLine(line), match[1], line, 0.78)
		}
		if match := expressRoute.FindStringSubmatch(line); len(match) == 3 {
			s.addIntegration(Integration{
				Name:       "Express route " + strings.ToUpper(match[1]) + " " + normalizeEndpoint(match[2]),
				Method:     strings.ToUpper(match[1]),
				Endpoint:   normalizeEndpoint(match[2]),
				SourceFile: rel,
				Line:       i + 1,
				Kind:       "express-route",
				Direction:  "server",
				External:   false,
				Confidence: 0.78,
				Evidence:   []string{strings.TrimSpace(line)},
				Tags:       []string{"project-scan", "javascript", "internal-route"},
			})
		}
	}
}

func (s *scanner) scanGo(rel string, lines []string) {
	for i, line := range lines {
		if match := goHTTPPattern.FindStringSubmatch(line); len(match) == 3 {
			method := strings.ToUpper(match[1])
			if method == "GET" || method == "POST" {
				s.addClientURL(rel, i+1, "go-http-client", method, match[2], line, 0.9)
			}
		}
		if match := goNewRequest.FindStringSubmatch(line); len(match) == 3 {
			s.addClientURL(rel, i+1, "go-http-client", strings.ToUpper(match[1]), match[2], line, 0.9)
		}
		if match := goNewRequestCtx.FindStringSubmatch(line); len(match) == 3 {
			s.addClientURL(rel, i+1, "go-http-client", strings.ToUpper(match[1]), match[2], line, 0.9)
		}
		if match := goRoutePattern.FindStringSubmatch(line); len(match) == 2 {
			s.addIntegration(Integration{
				Name:       "Go route " + normalizeEndpoint(match[1]),
				Method:     "GET",
				Endpoint:   normalizeEndpoint(match[1]),
				SourceFile: rel,
				Line:       i + 1,
				Kind:       "go-route",
				Direction:  "server",
				External:   false,
				Confidence: 0.65,
				Evidence:   []string{strings.TrimSpace(line)},
				Tags:       []string{"project-scan", "go", "internal-route"},
			})
		}
	}
}

func (s *scanner) scanPython(rel string, lines []string) {
	for i, line := range lines {
		if match := pythonHTTPPattern.FindStringSubmatch(line); len(match) == 3 {
			s.addClientURL(rel, i+1, "python-http-client", strings.ToUpper(match[1]), match[2], line, 0.9)
		}
	}
}

func (s *scanner) scanConfigURLs(rel string, lines []string) {
	for i, line := range lines {
		match := configURLPattern.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		key := strings.TrimSpace(match[1])
		value := strings.Trim(strings.TrimSpace(match[2]), `"'`)
		if value == "" || strings.HasPrefix(value, "${") {
			continue
		}
		if !strings.Contains(value, "://") && !strings.HasPrefix(value, "/") {
			continue
		}
		endpoint := normalizeEndpoint(value)
		if endpoint == "/" && isAbsoluteURL(value) {
			endpoint = "/" + pathSegmentFromHost(value)
		}
		s.addIntegration(Integration{
			Name:       configName(key, value),
			Method:     "GET",
			Endpoint:   endpoint,
			BaseURL:    baseURLFromValue(value),
			SourceFile: rel,
			Line:       i + 1,
			Kind:       "config-url",
			Direction:  "config",
			External:   isLikelyExternalConfig(value),
			Confidence: confidenceForURL(value, 0.6),
			Evidence:   []string{strings.TrimSpace(line)},
			Tags:       []string{"project-scan", "config-url"},
		})
	}
}

func (s *scanner) addClientURL(rel string, lineNo int, kind, method, value, evidence string, baseConfidence float64) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	endpoint := normalizeEndpoint(value)
	baseURL := baseURLFromValue(value)
	external := true
	confidence := confidenceForURL(value, baseConfidence)
	if isAbsoluteURL(value) {
		external = isExternalHost(value)
	} else if strings.HasPrefix(endpoint, "/") {
		external = true
		confidence = min(confidence, 0.68)
	}
	s.addIntegration(Integration{
		Name:       integrationName(kind, method, endpoint, value),
		Method:     method,
		Endpoint:   endpoint,
		BaseURL:    baseURL,
		SourceFile: rel,
		Line:       lineNo,
		Kind:       kind,
		Direction:  "client",
		External:   external,
		Confidence: confidence,
		Evidence:   []string{strings.TrimSpace(evidence)},
		Tags:       []string{"project-scan", kind},
	})
}

func (s *scanner) addIntegration(in Integration) {
	in.Method = strings.ToUpper(strings.TrimSpace(in.Method))
	if in.Method == "" {
		in.Method = "GET"
	}
	in.Endpoint = normalizeEndpoint(in.Endpoint)
	if in.Endpoint == "/favicon.ico" || strings.HasPrefix(in.Endpoint, "/_gofaux") {
		return
	}
	key := strings.Join([]string{in.Kind, in.Direction, in.Method, in.Endpoint, in.BaseURL, in.SourceFile, fmt.Sprint(in.Line)}, "|")
	if s.seen[key] {
		return
	}
	s.seen[key] = true
	if in.ID == "" {
		in.ID = stableID(key)
	}
	if in.Name == "" {
		in.Name = integrationName(in.Kind, in.Method, in.Endpoint, in.BaseURL)
	}
	if in.Confidence <= 0 {
		in.Confidence = 0.5
	}
	in.Tags = compact(in.Tags)
	in.Evidence = compact(in.Evidence)
	s.integrations = append(s.integrations, in)
}

func (s *scanner) rel(path string) string {
	rel, err := filepath.Rel(s.root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func shouldSkipDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".gofaux", ".tools", ".idea", ".vscode", ".gradle", ".next", ".nuxt", "node_modules", "vendor", "target", "build", "dist", "bin", "obj", "coverage", ".terraform":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func isScannableFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".spec.js") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".java", ".kt", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".go", ".py", ".json", ".yaml", ".yml", ".properties", ".toml", ".env", ".xml":
		return true
	default:
		return base == "dockerfile" || base == "makefile"
	}
}

func isConfigFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".properties" || ext == ".toml" || ext == ".env" || ext == ".xml" {
		return true
	}
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "config") || strings.Contains(base, "application")
}

func collectAnnotation(lines []string, start int) string {
	var b strings.Builder
	for i := start; i < len(lines) && i < start+5; i++ {
		b.WriteString(strings.TrimSpace(lines[i]))
		if strings.Contains(lines[i], ")") {
			break
		}
	}
	return b.String()
}

func javaMappingFromLine(line string) (string, string, bool) {
	if match := javaMapping.FindStringSubmatch(line); len(match) == 3 {
		method := strings.ToUpper(match[1])
		endpoint := firstStringLiteral(match[2])
		return method, endpoint, true
	}
	return "", "", false
}

func javaRequestMappingFromLine(line string) (string, string, bool) {
	match := javaRequestMapping.FindStringSubmatch(line)
	if len(match) != 2 {
		return "", "", false
	}
	annotation := match[1]
	method := attrValue(annotation, "method")
	if method == "" {
		method = "GET"
	}
	method = strings.TrimPrefix(strings.ToUpper(method), "REQUESTMETHOD.")
	endpoint := firstNonEmpty(attrValue(annotation, "value"), attrValue(annotation, "path"), firstStringLiteral(annotation))
	return method, endpoint, true
}

func attrValue(annotation, name string) string {
	for _, match := range javaAttr.FindAllStringSubmatch(annotation, -1) {
		if len(match) == 3 && strings.EqualFold(match[1], name) {
			return strings.Trim(match[2], ` "'`)
		}
	}
	if strings.EqualFold(name, "value") {
		return firstStringLiteral(annotation)
	}
	return ""
}

func firstStringLiteral(value string) string {
	match := javaStringLiteral.FindStringSubmatch(value)
	if len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func javaMethodSignature(lines []string, start int) (responseDTO, requestDTO, methodName string) {
	var joined string
	for i := start; i < len(lines) && i < start+7; i++ {
		text := strings.TrimSpace(lines[i])
		if text == "" || strings.HasPrefix(text, "@") {
			continue
		}
		joined += " " + text
		if strings.Contains(text, ";") || strings.Contains(text, "{") {
			break
		}
	}
	joined = strings.Join(strings.Fields(joined), " ")
	if joined == "" {
		return "", "", ""
	}
	if match := regexp.MustCompile(`(?:public\s+)?([A-Za-z0-9_.$<>?,]+)\s+([A-Za-z0-9_]+)\s*\((.*)\)`).FindStringSubmatch(joined); len(match) == 4 {
		responseDTO = cleanupDTO(match[1])
		methodName = match[2]
		if bodyMatch := regexp.MustCompile(`@RequestBody\s+([A-Za-z0-9_.$<>?,]+)`).FindStringSubmatch(match[3]); len(bodyMatch) == 2 {
			requestDTO = cleanupDTO(bodyMatch[1])
		}
	}
	return responseDTO, requestDTO, methodName
}

func cleanupDTO(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "ResponseEntity<")
	value = strings.TrimSuffix(value, ">")
	value = strings.TrimPrefix(value, "Mono<")
	value = strings.TrimSuffix(value, ">")
	if value == "void" || value == "Void" {
		return ""
	}
	return value
}

func methodFromRestTemplate(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "post"):
		return "POST"
	case strings.HasPrefix(lower, "put"):
		return "PUT"
	case strings.HasPrefix(lower, "delete"):
		return "DELETE"
	case strings.Contains(lower, "exchange"):
		return "GET"
	default:
		return "GET"
	}
}

func nearbyWebClientMethod(lines []string, index int) string {
	start := index - 4
	if start < 0 {
		start = 0
	}
	window := strings.ToLower(strings.Join(lines[start:index+1], " "))
	for _, method := range []string{"delete", "patch", "put", "post", "get"} {
		if strings.Contains(window, "."+method+"(") {
			return strings.ToUpper(method)
		}
	}
	return "GET"
}

func methodFromFetchLine(line string) string {
	match := regexp.MustCompile(`(?i)method\s*:\s*["']([A-Z]+)["']`).FindStringSubmatch(line)
	if len(match) == 2 {
		return strings.ToUpper(match[1])
	}
	return "GET"
}

func normalizeEndpoint(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if value == "" {
		return "/"
	}
	if strings.HasPrefix(value, "${") {
		if idx := strings.Index(value, "}"); idx >= 0 && idx+1 < len(value) {
			value = value[idx+1:]
		}
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		path := parsed.EscapedPath()
		if path == "" {
			path = "/"
		}
		return ensureSlash(path)
	}
	if strings.HasPrefix(value, "http") {
		return "/"
	}
	if strings.HasPrefix(value, "/") {
		return value
	}
	if strings.Contains(value, "/") {
		parts := strings.SplitN(value, "/", 2)
		return ensureSlash(parts[1])
	}
	return ensureSlash(value)
}

func joinEndpoint(prefix, endpoint string) string {
	prefix = normalizeEndpoint(prefix)
	endpoint = normalizeEndpoint(endpoint)
	if prefix == "/" {
		return endpoint
	}
	if endpoint == "/" {
		return prefix
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func ensureSlash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
}

func baseURLFromValue(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return parsed.Scheme + "://" + parsed.Host
	}
	if strings.HasPrefix(value, "${") {
		if idx := strings.Index(value, "}"); idx >= 0 {
			return value[:idx+1]
		}
	}
	return ""
}

func isAbsoluteURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func isLikelyExternalConfig(value string) bool {
	if isAbsoluteURL(value) {
		return isExternalHost(value)
	}
	return false
}

func isExternalHost(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := parsed.Hostname()
	if host == "" {
		host = parsed.Host
	}
	host = strings.ToLower(host)
	if host == "localhost" || host == "0.0.0.0" || host == "::1" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback() && !ip.IsPrivate()
	}
	return !strings.HasSuffix(host, ".local")
}

func confidenceForURL(value string, base float64) float64 {
	if isAbsoluteURL(value) {
		if isExternalHost(value) {
			return min(0.98, base+0.08)
		}
		return min(base, 0.55)
	}
	if strings.HasPrefix(value, "${") {
		return min(base, 0.72)
	}
	return base
}

func pathSegmentFromHost(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "external-service"
	}
	host := parsed.Hostname()
	host = strings.TrimPrefix(host, "api.")
	host = strings.Split(host, ".")[0]
	if host == "" {
		return "external-service"
	}
	return sanitizeSlug(host)
}

func configName(key, value string) string {
	if base := pathSegmentFromHost(value); base != "external-service" {
		return "Config URL " + key + " (" + base + ")"
	}
	return "Config URL " + key
}

func integrationName(kind, method, endpoint, value string) string {
	if host := pathSegmentFromHost(value); isAbsoluteURL(value) && host != "external-service" {
		return strings.ToUpper(method) + " " + host + " " + endpoint
	}
	return strings.ToUpper(method) + " " + endpoint + " (" + kind + ")"
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func compact(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func sanitizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "external-service"
	}
	return b.String()
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
