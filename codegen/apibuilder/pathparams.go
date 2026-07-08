package apibuilder

import (
	"regexp"
	"strings"
)

var pathParamPattern = regexp.MustCompile(`\{([^}]+)\}`)

// PathParam describes a path template placeholder such as {tenant_id}.
type PathParam struct {
	Name   string // OpenAPI path parameter name, e.g. tenant_id
	GoName string // Go identifier, e.g. tenantId
}

// PathBuildInfo describes how generated code should construct a resource path.
type PathBuildInfo struct {
	UseBuildResourcePathWithID bool
	ResourcePath               string
	SubPathSegments            []string
	PathParams                 []PathParam
}

// ExtractPathParams returns all {param} placeholders in path order.
func ExtractPathParams(path string) []PathParam {
	matches := pathParamPattern.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}

	params := make([]PathParam, 0, len(matches))
	for _, match := range matches {
		name := match[1]
		params = append(params, PathParam{
			Name:   name,
			GoName: pathParamToGoName(name),
		})
	}
	return params
}

// AnalyzePathBuild chooses BuildResourcePathWithID for a single path parameter,
// or positional InterpolatePathTemplate when multiple placeholders are present.
func AnalyzePathBuild(path string) PathBuildInfo {
	params := ExtractPathParams(path)
	info := PathBuildInfo{PathParams: params}
	if len(params) != 1 {
		return info
	}

	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	placeholder := "{" + params[0].Name + "}"
	paramIndex := -1
	for i, part := range pathParts {
		if part == placeholder {
			paramIndex = i
			break
		}
	}
	if paramIndex <= 0 {
		return info
	}

	info.UseBuildResourcePathWithID = true
	info.ResourcePath = strings.Join(pathParts[:paramIndex], "/")
	if paramIndex < len(pathParts)-1 {
		info.SubPathSegments = pathParts[paramIndex+1:]
	}
	return info
}

// PathHasIDParam reports whether the path has a /{id}/ path parameter segment.
func PathHasIDParam(path string) bool {
	for _, param := range ExtractPathParams(path) {
		if param.Name == "id" {
			return true
		}
	}
	return false
}

func pathParamToGoName(name string) string {
	if name == "id" {
		return "id"
	}

	parts := strings.Split(name, "_")
	var b strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			b.WriteString(strings.ToLower(part[:1]))
			if len(part) > 1 {
				b.WriteString(part[1:])
			}
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}
