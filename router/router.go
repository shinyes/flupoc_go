package router

import (
	"fmt"
	"strings"
)

// Context 请求上下文，包含路由处理所需的所有信息
type Context struct {
	PathParams  map[string]string // 路径参数，如 /users/{id} 中的 id
	QueryParams map[string]string // 查询参数，如 ?page=1&size=10
	// 未来可扩展：
	// RequestBody  []byte
	// Method string
	// Path string
}

// Response 响应数据结构
type Response struct {
	StatusCode int               // 状态码
	Headers    map[string]string // 响应头
	Body       interface{}       // 响应体，可以是字符串、[]byte、结构体等
}

// TODO NewResponse 创建一个默认的成功响应
func NewResponse(body interface{}) *Response {
	return &Response{
		StatusCode: 200,
		Headers:    make(map[string]string),
		Body:       body,
	}
}

// HandlerFunc 定义路由处理函数类型
// 返回响应数据和可能的错误
type HandlerFunc func(*Context) (*Response, error)

// Route 定义路由结构
type Route struct {
	Path    string
	Method  string
	Handler HandlerFunc
}

// Router 路由管理器
type Router struct {
	routes []*Route
}

// NewRouter 创建一个新的路由器
func NewRouter() *Router {
	return &Router{
		routes: make([]*Route, 0),
	}
}

// AddRoute 添加路由
func (r *Router) AddRoute(method, path string, handler HandlerFunc) {
	route := &Route{
		Path:    path,
		Method:  method,
		Handler: handler,
	}
	r.routes = append(r.routes, route)
}

// Get 添加GET路由
func (r *Router) Get(path string, handler HandlerFunc) {
	r.AddRoute("GET", path, handler)
}

// Post 添加POST路由
func (r *Router) Post(path string, handler HandlerFunc) {
	r.AddRoute("POST", path, handler)
}

// Put 添加PUT路由
func (r *Router) Put(path string, handler HandlerFunc) {
	r.AddRoute("PUT", path, handler)
}

// Delete 添加DELETE路由
func (r *Router) Delete(path string, handler HandlerFunc) {
	r.AddRoute("DELETE", path, handler)
}

// PathParams 解析路径参数
func (r *Router) PathParams(method, path string) (map[string]string, HandlerFunc, error) {
	for _, route := range r.routes {
		if strings.EqualFold(route.Method, method) {
			params := ExtractPathParams(route.Path, path)
			if params != nil {
				return params, route.Handler, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("no route found for %s %s", method, path)
}

// Match 匹配路由并返回完整的Context和Handler
// requestURL 可以包含查询参数，如 "/users/123?page=1&size=10"
func (r *Router) Match(method, requestURL string) (*Context, HandlerFunc, error) {
	// 分离路径和查询参数
	path, queryString := SplitPathAndQuery(requestURL)

	// 匹配路由
	for _, route := range r.routes {
		if strings.EqualFold(route.Method, method) {
			pathParams := ExtractPathParams(route.Path, path)
			if pathParams != nil {
				// 解析查询参数
				queryParams := ParseQueryParams(queryString)

				ctx := &Context{
					PathParams:  pathParams,
					QueryParams: queryParams,
				}
				return ctx, route.Handler, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("no route found for %s %s", method, requestURL)
}

// SplitPathAndQuery 分离路径和查询字符串
func SplitPathAndQuery(requestURL string) (path, query string) {
	if idx := strings.Index(requestURL, "?"); idx != -1 {
		return requestURL[:idx], requestURL[idx+1:]
	}
	return requestURL, ""
}

// ParseQueryParams 解析查询参数字符串
// 例如: "page=1&size=10&name=test" -> map["page":"1", "size":"10", "name":"test"]
func ParseQueryParams(queryString string) map[string]string {
	params := make(map[string]string)
	if queryString == "" {
		return params
	}

	// 按 & 分割参数
	pairs := strings.Split(queryString, "&")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}

		// 按 = 分割键值
		if idx := strings.Index(pair, "="); idx != -1 {
			key := pair[:idx]
			value := pair[idx+1:]
			params[key] = value
		} else {
			// 如果没有 =，则将值设为空字符串
			params[pair] = ""
		}
	}

	return params
}

// ParsePath 解析路径,返回路径段和参数名称映射
func ParsePath(path string) ([]string, map[string]string) {
	trimmed := strings.Trim(path, "/")
	// 处理空路径或根路径
	if trimmed == "" {
		return []string{}, make(map[string]string)
	}

	parts := strings.Split(trimmed, "/")
	params := make(map[string]string)
	staticParts := make([]string, len(parts))

	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := strings.Trim(part, "{}")
			// 记录参数名称及其在路径中的位置信息
			params[paramName] = fmt.Sprintf("position:%d", i)
			staticParts[i] = ":" + paramName // 使用:表示参数位置
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

	// 处理根路径
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
			// 静态路径匹配
			continue
		} else {
			// 不匹配
			return nil
		}
	}

	return params
}

// MatchRoute 简单判断路径是否匹配路由
func MatchRoute(routeTemplate, requestPath string) bool {
	routeTrimmed := strings.Trim(routeTemplate, "/")
	requestTrimmed := strings.Trim(requestPath, "/")

	// 处理根路径
	if routeTrimmed == "" && requestTrimmed == "" {
		return true
	}

	routeParts := strings.Split(routeTrimmed, "/")
	requestParts := strings.Split(requestTrimmed, "/")

	if len(routeParts) != len(requestParts) {
		return false
	}

	for i, routePart := range routeParts {
		requestPart := requestParts[i]

		if strings.HasPrefix(routePart, "{") && strings.HasSuffix(routePart, "}") {
			// 参数路径，总是匹配
			continue
		} else if routePart == requestPart {
			// 静态路径匹配
			continue
		} else {
			// 不匹配
			return false
		}
	}

	return true
}
