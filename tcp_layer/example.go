package tcplayer

import (
	"log"

	"github.com/cykyes/flupoc-go/protocol/service"
	"github.com/cykyes/flupoc-go/router"
)

// ExampleServer demonstrates starting a multi-address TLS server.
func ExampleServer() {
	r := router.NewRouter()
	r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
		return router.Bytes(ctx.RequestBody), nil
	})

	svc := service.New(r, service.Options{})

	if err := ListenAndServeTLS([]string{"127.0.0.1:5128"}, "server.crt", "server.key", svc.Handle); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}
