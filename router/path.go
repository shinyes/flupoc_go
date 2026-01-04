package router

import (
	"fmt"
	"strings"
)

// ParsePath 解析路径,返回路径段和参数名称映射
func ParsePath(path string) ([]string, map[string]string) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}, make(map[string]string)
	}

	parts := strings.Split(trimmed, "/")
	params := make(map[string]string)
	staticParts := make([]string, len(parts))

	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := strings.Trim(part, "{}")
			params[paramName] = fmt.Sprintf("position:%d", i)
			staticParts[i] = ":" + paramName
		} else {
			staticParts[i] = part
		}
	}

	return staticParts, params
}

// ExtractPathParams 从请求路径和路由模板中提取参数
func ExtractPathParams(routeTemplate, requestPath string) map[string]string {
	routeTrimmed := strings.Trim(routeTemplate, "/")
	requestTrimmed := strings.Trim(requestPath, "/")

	if routeTrimmed == "" && requestTrimmed == "" {
		return make(map[string]string)
	}

	routeParts := strings.Split(routeTrimmed, "/")
	requestParts := strings.Split(requestTrimmed, "/")

	if len(routeParts) != len(requestParts) {
		return nil
	}

	params := make(map[string]string)

	for i, routePart := range routeParts {
		requestPart := requestParts[i]

		if strings.HasPrefix(routePart, "{") && strings.HasSuffix(routePart, "}") {
			paramName := strings.Trim(routePart, "{}")
			params[paramName] = requestPart
		} else if routePart == requestPart {
			continue
		} else {
			return nil
		}
	}

	return params
}

// MatchRoute 简单判断路径是否匹配路由
func MatchRoute(routeTemplate, requestPath string) bool {
	return ExtractPathParams(routeTemplate, requestPath) != nil
}

// SplitPathAndQuery 分离路径和查询字符串
func SplitPathAndQuery(requestURL string) (path, query string) {
	path, query, ok := strings.Cut(requestURL, "?")
	if !ok {
		return requestURL, ""
	}
	return path, query
}

// ParseQueryParams 解析查询参数字符串
// 例如: "page=1&size=10&name=test" -> map["page":"1", "size":"10", "name":"test"]
func ParseQueryParams(queryString string) map[string]string {
	params := make(map[string]string)
	if queryString == "" {
		return params
	}

	for _, pair := range strings.Split(queryString, "&") {
		if pair == "" {
			continue
		}
		key, val, ok := strings.Cut(pair, "=")
		if !ok {
			params[pair] = ""
			continue
		}
		params[key] = val
	}

	return params
}

// normalizePath 确保路径以 / 开头且不含重复斜杠
func normalizePath(path string) string {
	return "/" + strings.Trim(path, "/")
}
