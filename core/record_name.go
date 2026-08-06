package core

import (
	"net/url"
	"regexp"
	"strings"
)

var apiVersionPattern = regexp.MustCompile(`^(?i)(v\d+|latest)$`)

// knownResourceDisplayNames maps conventional API resource path segments to display names.
var knownResourceDisplayNames = map[string]string{
	"vtasks": "VTask",
}

// recordDisplayName returns a singular capitalized resource name derived from the
// record's conventional self URL (e.g. .../api/v5/views/6/ -> "View").
// Returns an empty string when the record has no url or the url is not conventional.
func recordDisplayName(r Record) string {
	resourceSegment, ok := conventionalResourceSegmentFromRecord(r)
	if !ok {
		return ""
	}
	return resourceSegmentToDisplayName(resourceSegment)
}

func conventionalResourceSegmentFromRecord(r Record) (string, bool) {
	urlVal, ok := r["url"]
	if !ok {
		return "", false
	}
	urlStr, ok := urlVal.(string)
	if !ok || urlStr == "" {
		return "", false
	}
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", false
	}
	return conventionalResourceSegment(parsed)
}

// conventionalResourceSegment checks whether parsed is a conventional VAST resource URL:
// /api/{version}/{resource}/{id}/ with version like v5 or latest.
func conventionalResourceSegment(parsed *url.URL) (string, bool) {
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	apiIdx := -1
	for i, part := range parts {
		if strings.EqualFold(part, "api") {
			apiIdx = i
			break
		}
	}
	if apiIdx < 0 {
		return "", false
	}

	rest := parts[apiIdx+1:]
	if len(rest) != 3 {
		return "", false
	}

	version, resource, id := rest[0], rest[1], rest[2]
	if !apiVersionPattern.MatchString(version) {
		return "", false
	}
	if resource == "" || id == "" {
		return "", false
	}
	return resource, true
}

func resourceSegmentToDisplayName(segment string) string {
	if display, ok := knownResourceDisplayNames[strings.ToLower(segment)]; ok {
		return display
	}
	return capitalizeIdentifier(singularizeResourceSegment(segment))
}

func singularizeResourceSegment(segment string) string {
	lower := strings.ToLower(segment)
	switch lower {
	case "policies":
		return "policy"
	case "queries":
		return "query"
	case "entries":
		return "entry"
	case "indices":
		return "index"
	}
	if strings.HasSuffix(lower, "ies") && len(lower) > 3 {
		return lower[:len(lower)-3] + "y"
	}
	if strings.HasSuffix(lower, "s") && len(lower) > 1 {
		return lower[:len(lower)-1]
	}
	return lower
}

func capitalizeIdentifier(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func isVTaskRecord(record Record) bool {
	return recordDisplayName(record) == VTaskKey
}
