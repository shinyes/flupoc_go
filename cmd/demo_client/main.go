package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cykyes/flupoc-go/client"
)

// go run ./cmd/demo_client --addr=192.168.110.115:5128 --path=/echo --body="hello demo" --ca="d:\\gencert-data\\certs\\192.168.110.115-chain.pem"

func main() {
	addr := flag.String("addr", "127.0.0.1:5128", "server address")
	method := flag.String("method", "POST", "request method")
	path := flag.String("path", "/echo", "request path")
	body := flag.String("body", "hello", "request body")
	ca := flag.String("ca", "", "CA certificate for server verification")
	cert := flag.String("cert", "", "client certificate (optional)")
	key := flag.String("key", "", "client private key (optional)")
	insecure := flag.Bool("insecure", false, "skip server certificate verification")
	flag.Parse()

	cli, err := client.New(client.Options{
		CertFile: *cert,
		KeyFile:  *key,
		CAFile:   *ca,
		Insecure: *insecure,
	})
	if err != nil {
		log.Fatalf("创建客户端: %v", err)
	}

	resp, err := cli.Do(*addr, *method, *path, []byte(*body))
	if err != nil {
		log.Fatalf("请求失败: %v", err)
	}

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应体: %s\n", resp.Body)
}
