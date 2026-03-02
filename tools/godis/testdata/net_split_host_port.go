package main

import "net"

func main() {
	// Simple host:port
	host, port, err := net.SplitHostPort("localhost:8080")
	if err != nil {
		println("error")
	}
	println(host) // localhost
	println(port) // 8080

	// IPv6 with brackets
	host2, port2, _ := net.SplitHostPort("[::1]:443")
	println(host2) // ::1
	println(port2) // 443

	// IP:port
	host3, port3, _ := net.SplitHostPort("127.0.0.1:80")
	println(host3) // 127.0.0.1
	println(port3) // 80

	// JoinHostPort
	joined := net.JoinHostPort("example.com", "443")
	println(joined) // example.com:443

	println("ok")
}
