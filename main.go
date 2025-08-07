package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

func main() {
	// 定义命令行参数
	port := flag.String("port", "8080", "监听端口")
	target := flag.String("target", "", "目标地址 (例如: https://example.com 或 tcp://192.168.1.100:3306)")

	flag.Parse()

	// 检查是否提供了目标地址
	if *target == "" {
		log.Fatal("请提供目标地址，例如: -target=https://example.com 或 -target=tcp://192.168.1.100:3306")
	}

	// 解析目标地址协议
	if strings.HasPrefix(*target, "http") {
		// HTTP/HTTPS代理
		targetURL, err := url.Parse(*target)
		if err != nil {
			log.Fatalf("无效的目标地址: %v", err)
		}

		// 创建优化的反向代理
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		if strings.HasPrefix(*target, "https://") {
			proxy.Director = func(req *http.Request) {
				req.URL.Scheme = "https"
				req.URL.Host = targetURL.Host
				req.Host = targetURL.Host
			}
		}

		log.Printf("HTTP转发服务已启动，监听端口: %s\n", *port)
		log.Printf("所有请求将转发到: %s\n", *targetURL)
		log.Printf("访问 http://localhost:%s 进行使用\n", *port)

		log.Fatal(http.ListenAndServe(":"+*port, proxy))
	} else if strings.HasPrefix(*target, "tcp://") {
		// TCP代理
		targetAddr := strings.TrimPrefix(*target, "tcp://")

		log.Printf("TCP转发服务已启动，监听端口: %s\n", *port)
		log.Printf("所有TCP流量将转发到: %s\n", targetAddr)

		// 启动TCP代理服务
		log.Fatal(startTCPProxy(*port, targetAddr))
	} else if strings.HasPrefix(*target, "udp://") {
		// UDP代理
		targetAddr := strings.TrimPrefix(*target, "udp://")

		log.Printf("UDP转发服务已启动，监听端口: %s\n", *port)
		log.Printf("所有UDP流量将转发到: %s\n", targetAddr)

		// 启动UDP代理服务
		log.Fatal(startUDPProxy(*port, targetAddr))
	} else {
		log.Fatal("不支持的目标地址协议，请使用 http://, https://, tcp:// 或 udp:// 前缀")
	}
}

// startTCPProxy 启动TCP代理服务
func startTCPProxy(listenPort, targetAddr string) error {
	listener, err := net.Listen("tcp", ":"+listenPort)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("TCP代理监听在端口 %s\n", listenPort)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}

		go handleTCPConnection(clientConn, targetAddr)
	}
}

// handleTCPConnection 处理单个TCP连接
func handleTCPConnection(clientConn net.Conn, targetAddr string) {
	defer clientConn.Close()

	// 连接到目标服务器
	serverConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		log.Printf("连接到目标服务器失败 %s: %v", targetAddr, err)
		return
	}
	defer serverConn.Close()

	// 双向复制数据
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		transferData(serverConn, clientConn)
	}()

	go func() {
		defer wg.Done()
		transferData(clientConn, serverConn)
	}()

	// 等待两个方向的数据传输都完成后才关闭连接
	wg.Wait()
}

// transferData 在两个连接之间复制数据
func transferData(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("数据传输错误: %v", err)
	}
}

// startUDPProxy 启动UDP代理服务
func startUDPProxy(listenPort, targetAddr string) error {
	// 创建UDP监听
	listenAddr, err := net.ResolveUDPAddr("udp", ":"+listenPort)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("UDP代理监听在端口 %s\n", listenPort)

	// 用于存储客户端地址和对应的目标连接映射
	connections := make(map[string]*net.UDPConn)
	mutex := sync.RWMutex{}

	buf := make([]byte, 65535) // UDP最大包大小

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("读取UDP数据失败: %v", err)
			continue
		}

		// 查找或创建到目标服务器的连接
		mutex.RLock()
		serverConn, exists := connections[clientAddr.String()]
		mutex.RUnlock()

		if !exists {
			// 创建到目标服务器的新连接
			serverAddr, err := net.ResolveUDPAddr("udp", targetAddr)
			if err != nil {
				log.Printf("解析目标地址失败 %s: %v", targetAddr, err)
				continue
			}

			serverConn, err = net.DialUDP("udp", nil, serverAddr)
			if err != nil {
				log.Printf("连接到目标服务器失败 %s: %v", targetAddr, err)
				continue
			}

			mutex.Lock()
			connections[clientAddr.String()] = serverConn
			mutex.Unlock()

			// 启动从服务器到客户端的数据传输
			go func(clientAddr *net.UDPAddr, serverConn *net.UDPConn) {
				serverBuf := make([]byte, 65535)
				for {
					n, err := serverConn.Read(serverBuf)
					if err != nil {
						break
					}

					_, err = conn.WriteToUDP(serverBuf[:n], clientAddr)
					if err != nil {
						log.Printf("向客户端写入数据失败: %v", err)
						break
					}
				}

				// 清理连接
				mutex.Lock()
				delete(connections, clientAddr.String())
				mutex.Unlock()
				serverConn.Close()
			}(clientAddr, serverConn)
		}

		// 将数据转发到目标服务器
		_, err = serverConn.Write(buf[:n])
		if err != nil {
			log.Printf("向目标服务器写入数据失败: %v", err)
			mutex.Lock()
			delete(connections, clientAddr.String())
			mutex.Unlock()
			serverConn.Close()
		}
	}
}
