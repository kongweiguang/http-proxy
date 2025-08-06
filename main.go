package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	// 定义命令行参数
	port := flag.String("port", "8080", "监听端口")
	target := flag.String("target", "", "目标地址 (例如: https://example.com)")

	flag.Parse()

	// 检查是否提供了目标地址
	if *target == "" {
		log.Fatal("请提供目标地址，例如: -target=https://example.com")
	}

	// 解析目标地址
	targetURL, err := url.Parse(*target)
	if err != nil {
		log.Fatalf("无效的目标地址: %v", err)
	}

	// 创建优化的反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	log.Printf("HTTP转发服务已启动，监听端口: %s\n", *port)
	log.Printf("所有请求将转发到: %s\n", *targetURL)
	log.Printf("访问 http://localhost:%s 进行使用\n", *port)

	log.Fatal(http.ListenAndServe(":"+*port, proxy))
}
