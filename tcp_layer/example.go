package tcplayer

import (
	"log"

	"github.com/cykyes/flupoc-go/router"
)

// ExampleServer demonstrates starting a multi-address TLS server.
func ExampleServer() {
	r := router.NewRouter()
	r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
		return router.Bytes(ctx.RequestBody), nil
	})

	addrs := []string{"127.0.0.1:5128", "[::]:8443"}
	cert := "server.crt"
	key := "server.key"

	if err := ListenAndServeTLS(addrs, cert, key, r, nil); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}
