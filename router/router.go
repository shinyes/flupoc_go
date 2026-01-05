package router

import (
	"fmt"
	"strings"
)

// Route 定义路由结构
type Route struct {
	Path    string
	Method  string
	Handler HandlerFunc
	// Middlewares 组合了路由组中间件和路由级中间件，按声明顺序执行
	Middlewares []Middleware
	wrapped     HandlerFunc
}

// Router 路由管理器
type Router struct {
	routes      []*Route
	middlewares []Middleware
}

// RouteGroup 路由组，负责前缀拼接与组级中间件
type RouteGroup struct {
	prefix      string
	middlewares []Middleware
	router      *Router
}

// NewRouter 创建一个新的路由器
func NewRouter() *Router {
	return &Router{
		routes:      make([]*Route, 0),
		middlewares: make([]Middleware, 0),
	}
}

// AddRoute 添加路由并附带可选中间件
func (r *Router) AddRoute(method, path string, handler HandlerFunc, middlewares ...Middleware) {
	combined := append([]Middleware{}, r.middlewares...)
	combined = append(combined, middlewares...)

	route := &Route{
		Path:        normalizePath(path),
		Method:      method,
		Handler:     handler,
		Middlewares: combined,
		wrapped:     wrapWithMiddlewares(handler, combined),
	}
	r.routes = append(r.routes, route)
}

// Get 添加GET路由
func (r *Router) Get(path string, handler HandlerFunc, middlewares ...Middleware) {
	r.AddRoute("GET", path, handler, middlewares...)
}

// Post 添加POST路由
func (r *Router) Post(path string, handler HandlerFunc, middlewares ...Middleware) {
	r.AddRoute("POST", path, handler, middlewares...)
}

// Put 添加PUT路由
func (r *Router) Put(path string, handler HandlerFunc, middlewares ...Middleware) {
	r.AddRoute("PUT", path, handler, middlewares...)
}

// Delete 添加DELETE路由
func (r *Router) Delete(path string, handler HandlerFunc, middlewares ...Middleware) {
	r.AddRoute("DELETE", path, handler, middlewares...)
}

// Group 创建路由组，可附带组级中间件
func (r *Router) Group(prefix string, middlewares ...Middleware) *RouteGroup {
	return &RouteGroup{
		prefix:      normalizePath(prefix),
		middlewares: append([]Middleware{}, middlewares...),
		router:      r,
	}
}

// Use 为根路由注册中间件，等价于根组
func (r *Router) Use(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// PathParams 解析路径参数
func (r *Router) PathParams(method, path string) (map[string]string, HandlerFunc, error) {
	ctx, handler, err := r.Match(method, path)
	if err != nil {
		return nil, nil, err
	}
	return ctx.PathParams, handler, nil
}

// Match 查找匹配的路由并返回包含路径参数的 Context。
// requestURL 可以包含查询参数，例如 "/users/123?page=1&size=10"。
func (r *Router) Match(method, requestURL string) (*Context, HandlerFunc, error) {
	path, queryString := SplitPathAndQuery(requestURL)
	normalized := normalizePath(path)

	for _, route := range r.routes {
		if strings.EqualFold(route.Method, method) {
			if pathParams := ExtractPathParams(route.Path, normalized); pathParams != nil {
				queryParams := ParseQueryParams(queryString)
				ctx := NewContext(nil)
				ctx.PathParams = pathParams
				ctx.QueryParams = queryParams
				return ctx, route.wrapped, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no route found for %s %s", method, requestURL)
}

// ServeRequest 将请求匹配路由并执行中间件链。
func (r *Router) ServeRequest(req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	ctx, handler, err := r.Match(req.Method, req.Path)
	if err != nil {
		return nil, err
	}

	ctx.RequestBody = req.Body
	ctx.Method = req.Method
	ctx.Path = req.Path

	return handler(ctx)
}

// wrapWithMiddlewares 按注册顺序将中间件包裹在 handler 外层
func wrapWithMiddlewares(handler HandlerFunc, middlewares []Middleware) HandlerFunc {
	finalHandler := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}
	return finalHandler
}

// joinPaths 拼接组前缀和子路径
func joinPaths(prefix, path string) string {
	if prefix == "" || prefix == "/" {
		return normalizePath(path)
	}
	return normalizePath(prefix + "/" + strings.Trim(path, "/"))
}

// Use 追加组级中间件
func (g *RouteGroup) Use(middlewares ...Middleware) {
	g.middlewares = append(g.middlewares, middlewares...)
}

// AddRoute 路由组添加路由
func (g *RouteGroup) AddRoute(method, path string, handler HandlerFunc, middlewares ...Middleware) {
	fullPath := joinPaths(g.prefix, path)
	combined := append([]Middleware{}, g.middlewares...)
	combined = append(combined, middlewares...)
	g.router.AddRoute(method, fullPath, handler, combined...)
}

// Get 添加GET路由
func (g *RouteGroup) Get(path string, handler HandlerFunc, middlewares ...Middleware) {
	g.AddRoute("GET", path, handler, middlewares...)
}

// Post 添加POST路由
func (g *RouteGroup) Post(path string, handler HandlerFunc, middlewares ...Middleware) {
	g.AddRoute("POST", path, handler, middlewares...)
}

// Put 添加PUT路由
func (g *RouteGroup) Put(path string, handler HandlerFunc, middlewares ...Middleware) {
	g.AddRoute("PUT", path, handler, middlewares...)
}

// Delete 添加DELETE路由
func (g *RouteGroup) Delete(path string, handler HandlerFunc, middlewares ...Middleware) {
	g.AddRoute("DELETE", path, handler, middlewares...)
}
