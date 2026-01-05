package router

import (
	"reflect"
	"testing"
)

func TestRouter_AddRouteAndPathParams(t *testing.T) {
	router := NewRouter()

	// 添加测试路由
	router.AddRoute("GET", "/users/{id}", func(ctx *Context) (*Response, error) {
		return NewResponse("user data"), nil
	})

	router.AddRoute("GET", "/users/{id}/posts/{postId}", func(ctx *Context) (*Response, error) {
		return NewResponse("post data"), nil
	})

	router.AddRoute("POST", "/users", func(ctx *Context) (*Response, error) {
		return NewResponse("created"), nil
	})

	// 测试路径参数提取
	t.Run("提取路径参数", func(t *testing.T) {
		params, _, err := router.PathParams("GET", "/users/123")
		if err != nil {
			t.Errorf("期望成功提取参数，但出现错误: %v", err)
		}

		if params["id"] != "123" {
			t.Errorf("期望参数 id 为 '123'，实际得到 '%s'", params["id"])
		}
	})

	t.Run("提取多个路径参数", func(t *testing.T) {
		params, _, err := router.PathParams("GET", "/users/123/posts/456")
		if err != nil {
			t.Errorf("期望成功提取参数，但出现错误: %v", err)
		}

		if params["id"] != "123" {
			t.Errorf("期望参数 id 为 '123'，实际得到 '%s'", params["id"])
		}

		if params["postId"] != "456" {
			t.Errorf("期望参数 postId 为 '456'，实际得到 '%s'", params["postId"])
		}
	})

	t.Run("方法未找到", func(t *testing.T) {
		_, _, err := router.PathParams("PUT", "/users/123")
		if err == nil {
			t.Error("期望返回错误，但实际成功")
		}
	})

	t.Run("路径未找到", func(t *testing.T) {
		_, _, err := router.PathParams("GET", "/products/123")
		if err == nil {
			t.Error("期望返回错误，但实际成功")
		}
	})
}

func TestParsePath(t *testing.T) {
	t.Run("解析静态路径", func(t *testing.T) {
		parts, params := ParsePath("/users/profile")
		expectedParts := []string{"users", "profile"}

		if len(parts) != len(expectedParts) {
			t.Errorf("期望 %d 个路径段，实际得到 %d 个", len(expectedParts), len(parts))
		} else {
			for i, part := range expectedParts {
				if parts[i] != part {
					t.Errorf("期望路径段为 '%s'，实际得到 '%s'", part, parts[i])
				}
			}
		}

		if len(params) != 0 {
			t.Errorf("期望没有参数，实际得到 %d 个参数", len(params))
		}
	})

	t.Run("解析带参数的路径", func(t *testing.T) {
		parts, params := ParsePath("/users/{id}/posts/{postId}")
		expectedParts := []string{"users", ":id", "posts", ":postId"}

		if len(parts) != len(expectedParts) {
			t.Errorf("期望 %d 个路径段，实际得到 %d 个", len(expectedParts), len(parts))
		} else {
			for i, part := range expectedParts {
				if parts[i] != part {
					t.Errorf("期望路径段为 '%s'，实际得到 '%s'", part, parts[i])
				}
			}
		}

		if len(params) != 2 {
			t.Errorf("期望有2个参数，实际得到 %d 个参数", len(params))
		} else {
			if _, exists := params["id"]; !exists {
				t.Error("期望存在参数 'id'")
			}
			if _, exists := params["postId"]; !exists {
				t.Error("期望存在参数 'postId'")
			}
		}
	})
}

func TestExtractPathParams(t *testing.T) {
	t.Run("提取路径参数", func(t *testing.T) {
		routeTemplate := "/users/{id}/posts/{postId}"
		requestPath := "/users/123/posts/456"
		params := ExtractPathParams(routeTemplate, requestPath)

		if params["id"] != "123" {
			t.Errorf("期望参数 id 为 '123'，实际得到 '%s'", params["id"])
		}

		if params["postId"] != "456" {
			t.Errorf("期望参数 postId 为 '456'，实际得到 '%s'", params["postId"])
		}
	})

	t.Run("非匹配路径", func(t *testing.T) {
		routeTemplate := "/users/{id}/posts/{postId}"
		requestPath := "/users/123/comments/456"
		params := ExtractPathParams(routeTemplate, requestPath)

		if params != nil {
			t.Error("期望返回 nil，因为路径不匹配")
		}
	})
}

func TestMatchRoute(t *testing.T) {
	t.Run("匹配路径", func(t *testing.T) {
		routeTemplate := "/users/{id}/posts/{postId}"
		requestPath := "/users/123/posts/456"
		isMatch := MatchRoute(routeTemplate, requestPath)

		if !isMatch {
			t.Error("期望路径匹配，但实际不匹配")
		}
	})

	t.Run("非匹配路径", func(t *testing.T) {
		routeTemplate := "/users/{id}/posts/{postId}"
		requestPath := "/users/123/comments/456"
		isMatch := MatchRoute(routeTemplate, requestPath)

		if isMatch {
			t.Error("期望路径不匹配，但实际匹配")
		}
	})

	t.Run("不同路径长度", func(t *testing.T) {
		routeTemplate := "/users/{id}/posts/{postId}"
		requestPath := "/users/123"
		isMatch := MatchRoute(routeTemplate, requestPath)

		if isMatch {
			t.Error("期望路径不匹配，但实际匹配")
		}
	})
}

func TestHTTPMethods(t *testing.T) {
	router := NewRouter()

	// 测试不同的HTTP方法
	router.Get("/test", func(ctx *Context) (*Response, error) {
		return NewResponse("get"), nil
	})

	router.Post("/test", func(ctx *Context) (*Response, error) {
		return NewResponse("post"), nil
	})

	router.Put("/test", func(ctx *Context) (*Response, error) {
		return NewResponse("put"), nil
	})

	router.Delete("/test", func(ctx *Context) (*Response, error) {
		return NewResponse("delete"), nil
	})

	// 测试各种方法是否都能正确匹配
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		_, _, err := router.PathParams(method, "/test")
		if err != nil {
			t.Errorf("%s 方法匹配失败: %v", method, err)
		}
	}
}

func TestParseQueryParams(t *testing.T) {
	t.Run("解析标准查询参数", func(t *testing.T) {
		query := "page=1&size=10&name=test"
		params := ParseQueryParams(query)

		if params["page"] != "1" {
			t.Errorf("期望 page=1，实际得到 %s", params["page"])
		}
		if params["size"] != "10" {
			t.Errorf("期望 size=10，实际得到 %s", params["size"])
		}
		if params["name"] != "test" {
			t.Errorf("期望 name=test，实际得到 %s", params["name"])
		}
	})

	t.Run("解析空查询字符串", func(t *testing.T) {
		params := ParseQueryParams("")
		if len(params) != 0 {
			t.Errorf("期望空map，实际得到 %d 个参数", len(params))
		}
	})

	t.Run("解析没有值的参数", func(t *testing.T) {
		query := "flag&another"
		params := ParseQueryParams(query)

		if params["flag"] != "" {
			t.Errorf("期望 flag 为空字符串，实际得到 %s", params["flag"])
		}
		if params["another"] != "" {
			t.Errorf("期望 another 为空字符串，实际得到 %s", params["another"])
		}
	})
}

func TestSplitPathAndQuery(t *testing.T) {
	t.Run("分离路径和查询参数", func(t *testing.T) {
		path, query := SplitPathAndQuery("/users/123?page=1&size=10")

		if path != "/users/123" {
			t.Errorf("期望路径为 /users/123，实际得到 %s", path)
		}
		if query != "page=1&size=10" {
			t.Errorf("期望查询参数为 page=1&size=10，实际得到 %s", query)
		}
	})

	t.Run("没有查询参数", func(t *testing.T) {
		path, query := SplitPathAndQuery("/users/123")

		if path != "/users/123" {
			t.Errorf("期望路径为 /users/123，实际得到 %s", path)
		}
		if query != "" {
			t.Errorf("期望查询参数为空，实际得到 %s", query)
		}
	})
}

func TestRouter_Match(t *testing.T) {
	router := NewRouter()

	router.Get("/users/{id}", func(ctx *Context) (*Response, error) {
		return NewResponse(map[string]string{"id": ctx.PathParams["id"]}), nil
	})

	t.Run("匹配带查询参数的请求", func(t *testing.T) {
		ctx, handler, err := router.Match("GET", "/users/123?page=1&size=10")

		if err != nil {
			t.Errorf("期望匹配成功，但出现错误: %v", err)
		}

		if handler == nil {
			t.Error("期望返回handler，但为nil")
		}

		if ctx.PathParams["id"] != "123" {
			t.Errorf("期望路径参数 id=123，实际得到 %s", ctx.PathParams["id"])
		}

		if ctx.QueryParams["page"] != "1" {
			t.Errorf("期望查询参数 page=1，实际得到 %s", ctx.QueryParams["page"])
		}

		if ctx.QueryParams["size"] != "10" {
			t.Errorf("期望查询参数 size=10，实际得到 %s", ctx.QueryParams["size"])
		}
	})

	t.Run("匹配不带查询参数的请求", func(t *testing.T) {
		ctx, handler, err := router.Match("GET", "/users/456")

		if err != nil {
			t.Errorf("期望匹配成功，但出现错误: %v", err)
		}

		if handler == nil {
			t.Error("期望返回handler，但为nil")
		}

		if ctx.PathParams["id"] != "456" {
			t.Errorf("期望路径参数 id=456，实际得到 %s", ctx.PathParams["id"])
		}

		if len(ctx.QueryParams) != 0 {
			t.Errorf("期望没有查询参数，实际得到 %d 个", len(ctx.QueryParams))
		}
	})
}

func TestMiddlewareOrder(t *testing.T) {
	order := make([]string, 0)

	rootMW := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (*Response, error) {
			order = append(order, "root")
			return next(ctx)
		}
	}

	groupMW := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (*Response, error) {
			order = append(order, "group")
			return next(ctx)
		}
	}

	routeMW := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (*Response, error) {
			order = append(order, "route")
			return next(ctx)
		}
	}

	router := NewRouter()
	router.Use(rootMW)

	api := router.Group("/api", groupMW)
	api.Get("/items/{id}", func(ctx *Context) (*Response, error) {
		order = append(order, "handler")
		return NewResponse(ctx.PathParams["id"]), nil
	}, routeMW)

	ctx, handler, err := router.Match("GET", "/api/items/99")
	if err != nil {
		t.Fatalf("期望匹配成功，但出现错误: %v", err)
	}

	if _, err := handler(ctx); err != nil {
		t.Fatalf("期望处理成功，但出现错误: %v", err)
	}

	expected := []string{"root", "group", "route", "handler"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("中间件执行顺序错误，期望 %v，实际 %v", expected, order)
	}
}

func TestRouteLevelMiddlewareWithoutGroup(t *testing.T) {
	order := make([]string, 0)

	routeMW := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (*Response, error) {
			order = append(order, "route")
			return next(ctx)
		}
	}

	router := NewRouter()
	router.Get("/ping", func(ctx *Context) (*Response, error) {
		order = append(order, "handler")
		return NewResponse("pong"), nil
	}, routeMW)

	ctx, handler, err := router.Match("GET", "/ping")
	if err != nil {
		t.Fatalf("期望匹配成功，但出现错误: %v", err)
	}

	if _, err := handler(ctx); err != nil {
		t.Fatalf("期望处理成功，但出现错误: %v", err)
	}

	expected := []string{"route", "handler"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("路由级中间件执行顺序错误，期望 %v，实际 %v", expected, order)
	}
}

func TestServeRequest(t *testing.T) {
	router := NewRouter()

	router.Post("/echo", func(ctx *Context) (*Response, error) {
		return NewResponse(map[string]string{
			"body": string(ctx.RequestBody),
		}), nil
	})

	resp, err := router.ServeRequest(&Request{Method: "POST", Path: "/echo", Body: []byte("hello")})
	if err != nil {
		t.Fatalf("期望路由成功，实际错误: %v", err)
	}

	if resp.Body == nil {
		t.Fatalf("期望响应体非空")
	}

	bodyMap, ok := resp.Body.(map[string]string)
	if !ok {
		t.Fatalf("期望响应体为 map[string]string，实际类型 %T", resp.Body)
	}

	if bodyMap["body"] != "hello" {
		t.Fatalf("响应内容不匹配，期望 hello，实际 %s", bodyMap["body"])
	}
}

func TestServeRequestWithMiddleware(t *testing.T) {
	order := make([]string, 0)

	mwRoot := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (*Response, error) {
			order = append(order, "mwRoot")
			return next(ctx)
		}
	}

	mwRoute := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (*Response, error) {
			order = append(order, "mwRoute")
			return next(ctx)
		}
	}

	router := NewRouter()
	router.Use(mwRoot)

	router.Post("/demo", func(ctx *Context) (*Response, error) {
		order = append(order, "handler")
		return NewTextResponse("ok"), nil
	}, mwRoute)

	resp, err := router.ServeRequest(&Request{Method: "POST", Path: "/demo", Body: []byte("body")})
	if err != nil {
		t.Fatalf("期望处理成功，实际错误: %v", err)
	}

	if got := resp.Headers["Content-Type"]; got != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain; charset=utf-8, got %s", got)
	}

	bytesBody, err := resp.Bytes()
	if err != nil {
		t.Fatalf("期望序列化成功，实际错误: %v", err)
	}
	if string(bytesBody) != "ok" {
		t.Fatalf("期望响应体 ok，实际 %s", string(bytesBody))
	}

	expectedOrder := []string{"mwRoot", "mwRoute", "handler"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("中间件执行顺序不匹配，期望 %v，实际 %v", expectedOrder, order)
	}
}
