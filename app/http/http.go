package http

import (
	"fmt"
	"net"
	"os"
)

func Serve(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	response := "HTTP/1.1 200 OK\r\n\r\n"
	conn.Write([]byte(response))
}
