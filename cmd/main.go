package main

import (
	"flag"
	"log"
	"strings"

	"github.com/cykyes/flupoc-go/router"
	tcplayer "github.com/cykyes/flupoc-go/tcp_layer"
)

func main() {
	addrs := flag.String("addrs", "127.0.0.1:5128", "listen addresses (comma separated)")
	certFile := flag.String("cert", "", "TLS certificate file")
	keyFile := flag.String("key", "", "TLS private key file")
	flag.Parse()

	if *certFile == "" || *keyFile == "" {
		log.Fatal("必须提供 --cert 和 --key 参数")
	}

	parsedAddrs := parseAddrs(*addrs)
	if len(parsedAddrs) == 0 {
		log.Fatal("未提供有效地址")
	}

	r := router.NewRouter()
	r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
		return router.Bytes(ctx.RequestBody), nil
	})

	if err := tcplayer.ListenAndServeTLS(parsedAddrs, *certFile, *keyFile, r, nil); err != nil {
		log.Fatalf("服务器退出: %v", err)
	}
}

func parseAddrs(s string) []string {
	var result []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
