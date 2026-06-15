package mock

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
)

func (d Definition) Matches(r *http.Request) (Match, bool) {
	if NormalizeMethod(r.Method) != NormalizeMethod(d.Method) {
		return Match{}, false
	}

	params, ok := MatchPath(d.Endpoint, r.URL.Path)
	if !ok {
		return Match{}, false
	}
	if !MatchesQuery(d.Query, r.URL.Query()) {
		return Match{}, false
	}
	if !MatchesHeaders(d.RequestHeaders, r.Header) {
		return Match{}, false
	}

	score := len(splitPath(d.Endpoint))*10 + len(d.Query)*5 + len(d.RequestHeaders)*3
	return Match{PathParams: params, Score: score}, true
}

func MatchPath(pattern, actual string) (map[string]string, bool) {
	pattern = NormalizeEndpoint(pattern)
	actual = NormalizeEndpoint(actual)
	if pattern == actual {
		return map[string]string{}, true
	}

	patternParts := splitPath(pattern)
	actualParts := splitPath(actual)
	if len(patternParts) != len(actualParts) {
		return nil, false
	}

	params := make(map[string]string)
	for i, patternPart := range patternParts {
		actualPart, err := url.PathUnescape(actualParts[i])
		if err != nil {
			actualPart = actualParts[i]
		}

		if isPathParam(patternPart) {
			name := strings.TrimSuffix(strings.TrimPrefix(patternPart, "{"), "}")
			name = strings.TrimPrefix(name, ":")
			if name == "" {
				return nil, false
			}
			params[name] = actualPart
			continue
		}

		if patternPart != actualParts[i] {
			return nil, false
		}
	}
	return params, true
}

func MatchesQuery(expected map[string]string, actual url.Values) bool {
	for key, want := range expected {
		if actual.Get(key) != want {
			return false
		}
	}
	return true
}

func MatchesHeaders(expected map[string]string, actual http.Header) bool {
	for key, want := range expected {
		if actual.Get(key) != want {
			return false
		}
	}
	return true
}

func SortByMatchPriority(defs []Definition) {
	sort.SliceStable(defs, func(i, j int) bool {
		if defs[i].Priority != defs[j].Priority {
			return defs[i].Priority > defs[j].Priority
		}
		if len(splitPath(defs[i].Endpoint)) != len(splitPath(defs[j].Endpoint)) {
			return len(splitPath(defs[i].Endpoint)) > len(splitPath(defs[j].Endpoint))
		}
		if len(defs[i].Query) != len(defs[j].Query) {
			return len(defs[i].Query) > len(defs[j].Query)
		}
		return defs[i].CreatedAt.Before(defs[j].CreatedAt)
	})
}

func splitPath(value string) []string {
	value = strings.Trim(NormalizeEndpoint(value), "/")
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

func isPathParam(segment string) bool {
	return (strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")) ||
		strings.HasPrefix(segment, ":")
}
