package router

import (
	"fmt"
)

// 示例使用函数
func ExampleUsage() {
	router := NewRouter()

	// 添加路由
	router.Get("/", func(ctx *Context) (*Response, error) {
		fmt.Println("访问根路径")
		return NewResponse("欢迎访问"), nil
	})

	router.Get("/users/{id}", func(ctx *Context) (*Response, error) {
		userID := ctx.PathParams["id"]
		fmt.Printf("获取用户信息，用户ID: %s\n", userID)
		return NewResponse(map[string]string{
			"id":   userID,
			"name": "用户" + userID,
		}), nil
	})

	router.Get("/users/{id}/posts/{postId}", func(ctx *Context) (*Response, error) {
		userID := ctx.PathParams["id"]
		postID := ctx.PathParams["postId"]
		fmt.Printf("获取用户 %s 的文章 %s\n", userID, postID)
		return NewResponse(map[string]string{
			"userId": userID,
			"postId": postID,
			"title":  "文章标题",
		}), nil
	})

	router.Post("/users", func(ctx *Context) (*Response, error) {
		fmt.Println("创建新用户")
		resp := NewResponse(map[string]string{"id": "1001", "status": "created"})
		resp.StatusCode = 201
		return resp, nil
	})

	// 测试路径解析
	testPath := "/users/123/posts/456"
	params, handler, err := router.PathParams("GET", testPath)
	if err != nil {
		fmt.Printf("错误: %s\n", err)
	} else {
		fmt.Printf("匹配路径: %s\n", testPath)
		fmt.Printf("提取的参数: %+v\n", params)
		// 调用处理器
		ctx := &Context{
			PathParams:  params,
			QueryParams: make(map[string]string),
		}
		resp, err := handler(ctx)
		if err != nil {
			fmt.Printf("处理器错误: %v\n", err)
		} else {
			fmt.Printf("响应状态码: %d\n", resp.StatusCode)
			fmt.Printf("响应数据: %+v\n", resp.Body)
		}
	}

	// 测试路径参数提取
	routeTemplate := "/users/{id}/posts/{postId}"
	requestPath := "/users/123/posts/456"
	extractedParams := ExtractPathParams(routeTemplate, requestPath)
	fmt.Printf("从路径 %s 中提取参数: %+v\n", requestPath, extractedParams)

	// 测试路径匹配
	isMatch := MatchRoute(routeTemplate, requestPath)
	fmt.Printf("路径 %s 是否匹配模板 %s: %t\n", requestPath, routeTemplate, isMatch)

	// 测试完整的请求匹配（包含查询参数）
	fmt.Println("\n--- 测试查询参数解析 ---")
	fullURL := "/users/789?page=2&size=20&sort=name"
	ctx, handler, err := router.Match("GET", fullURL)
	if err != nil {
		fmt.Printf("错误: %s\n", err)
	} else {
		fmt.Printf("匹配URL: %s\n", fullURL)
		fmt.Printf("路径参数: %+v\n", ctx.PathParams)
		fmt.Printf("查询参数: %+v\n", ctx.QueryParams)
		resp, err := handler(ctx)
		if err != nil {
			fmt.Printf("处理器错误: %v\n", err)
		} else {
			fmt.Printf("响应: %+v\n", resp.Body)
		}
	}
}
