package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
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

	// Request line handling
	line, err := r.ReadString('\n')
	if err != nil {
		log.Println("read request line:", err)
		return
	}
	line = strings.TrimSuffix(line, "\r\n")
	req, err := parseRequestLine(line)
	if err != nil {
		log.Println(err)
		writeResponse(conn, 400, "text/plain", "Bad Request")
		return
	}
	log.Printf("\n---REQUEST---\n%v\n---REQUEST---\n", req)

	// Headers handling
	headers, err := parseHeaders(r)
	if err != nil {
		log.Println(err)
		writeResponse(conn, 400, "text/plain", "Bad Request")
		return
	}
	log.Printf("\n---HEADERS---\n%v\n---HEADERS---\n", headers)

	// Body handling
	body, err := parseBody(r, headers)
	if err != nil {
		log.Println(err)
		writeResponse(conn, 400, "text/plain", "Bad Request")
		return
	}

	// Determine request type & write response
	switch req.Target {
	case "/":
		err = writeResponse(conn, 200, "text/plain", "Hello, world!")
	case "/about":
		err = writeResponse(conn, 200, "text/plain", "About page")
	case "/echo":
		switch req.Method {
		case "POST":
			err = writeResponse(conn, 200, "text/plain", string(body))
		default:
			err = writeResponse(conn, 405, "text/plain", "Method Not Allowed")
		}
	default:
		err = writeResponse(conn, 404, "text/plain", "Not Found")
	}
	if err != nil {
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

	return nil, fmt.Errorf("too many headers (max 100)")
}

var reasonPhrases = map[int]string{
	200: "OK",
	400: "Bad Request",
	404: "Not Found",
	405: "Method Not Allowed",
	413: "Payload Too Large",
	500: "Internal Server Error",
}

func parseBody(r *bufio.Reader, headers map[string]string) ([]byte, error) {
	lengthStr, ok := headers["content-length"]
	if !ok { // No body?
		return nil, nil
	}
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid content length")
	}
	if length < 0 {
		return nil, fmt.Errorf("client sent a negative Content-Length")
	}
	if length > 1<<20 { // 1 MB
		return nil, fmt.Errorf("body too large: %d", length)
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("body shorter than promised: %w", err)
	}
	return body, nil
}

func writeResponse(w io.Writer, status int, contentType string, body string) error {
	phrase, ok := reasonPhrases[status]
	if !ok {
		phrase = "Unknown"
	}

	lines := []string{
		fmt.Sprintf("HTTP/1.1 %d %s", status, phrase),
		fmt.Sprintf("Content-Type: %s", contentType),
		fmt.Sprintf("Content-Length: %d", len(body)),
		"Connection: close",
	}
	head := strings.Join(lines, "\r\n")
	resp := head + "\r\n\r\n" + body

	_, err := w.Write([]byte(resp))
	return err
}
