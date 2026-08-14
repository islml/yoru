package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Println("listening on :8080")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)

	line, err := r.ReadString('\n')
	if err != nil {
		log.Println("read request line:", err)
		return
	}
	line = strings.TrimSuffix(line, "\r\n")
	req, err := parseRequestLine(line)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("---REQUEST---\n%v\n---REQUEST---\n", req)

	headers, err := parseHeaders(r)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("---HEADERS---\n%v\n---HEADERS---\n", headers)

	body := "Hello, world!"
	resp := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)

	if _, err := conn.Write([]byte(resp)); err != nil {
		log.Println("write:", err)
	}
}

type RequestLine struct {
	Method  string
	Target  string
	Version string
}

func parseRequestLine(line string) (RequestLine, error) {
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return RequestLine{}, fmt.Errorf("malformed request line: %q", line)
	}

	return RequestLine{
		Method:  parts[0],
		Target:  parts[1],
		Version: parts[2],
	}, nil
}

func parseHeaders(r *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)
	limit := 100

	for limit > 0 {
		limit--

		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSuffix(line, "\r\n")
		if line == "" {
			return headers, nil
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 1 {
			return nil, fmt.Errorf("malformed line: %q", line)
		}

		key := strings.ToLower(parts[0])
		value := strings.TrimSpace(parts[1])
		headers[key] = value
	}

	return nil, fmt.Errorf("too many headers (max %d)", limit)
}