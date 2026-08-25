package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", ":6767")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.SetFlags(0)
	printBanner(":6767")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	req, err := parseRequest(bufio.NewReader(conn))
	if err != nil {
		log.Println(err)
		writeResponse(conn, newResponse(400, "Bad Request"))
		return
	}

	route(conn, req)
}

// --- Routing ---

func route(conn net.Conn, req Request) {
	start := time.Now()
	handler, ok := routes[req.Target]

	var res Response
	if ok {
		res = handler(req)
	} else {
		res = newResponse(404, "Not Found")
	}

	logRequest(req, res.Status, time.Since(start))

	if err := writeResponse(conn, res); err != nil {
		log.Println("write:", err)
	}
}

type Handler func(req Request) Response

var routes = map[string]Handler{
	"/":      homeHandler,
	"/about": aboutHandler,
	"/echo":  echoHandler,
}

func homeHandler(req Request) Response {
	return newResponse(200, "Hello, world!")
}

func aboutHandler(req Request) Response {
	return newResponse(200, "About page")
}

func echoHandler(req Request) Response {
	if req.Method != "POST" {
		return newResponse(405, "Method Not Allowed")
	}
	return newResponse(200, string(req.Body))
}

// --- Request Parsing ---

type Request struct {
	RequestLine
	Headers map[string]string
	Body    []byte
}

type RequestLine struct {
	Method  string
	Target  string
	Version string
}

func parseRequest(r *bufio.Reader) (Request, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return Request{}, fmt.Errorf("read request line: %w", err)
	}
	line = strings.TrimSuffix(line, "\r\n")

	reqLine, err := parseRequestLine(line)
	if err != nil {
		return Request{}, err
	}

	headers, err := parseHeaders(r)
	if err != nil {
		return Request{}, err
	}

	body, err := parseBody(r, headers)
	if err != nil {
		return Request{}, err
	}

	return Request{
		RequestLine: reqLine,
		Headers:     headers,
		Body:        body,
	}, nil
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
		if line == "" { // End of headers
			return headers, nil
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 1 {
			return nil, fmt.Errorf("malformed line in headers: %q", line)
		}

		key := strings.ToLower(parts[0])
		value := strings.TrimSpace(parts[1])
		headers[key] = value
	}

	return nil, fmt.Errorf("too many headers (max 100)")
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

// --- Response Writing ---

var reasonPhrases = map[int]string{
	200: "OK",
	400: "Bad Request",
	404: "Not Found",
	405: "Method Not Allowed",
	413: "Payload Too Large",
	500: "Internal Server Error",
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

func (res Response) Bytes() []byte {
	phrase, ok := reasonPhrases[res.Status]
	if !ok {
		phrase = "Unknown"
	}

	head := fmt.Sprintf("HTTP/1.1 %d %s\r\n", res.Status, phrase)
	head += fmt.Sprintf("Content-Length: %d\r\n", len(res.Body))

	for key, value := range res.Headers {
		head += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	head += "\r\n"
	return append([]byte(head), res.Body...)
}

func newResponse(status int, body string) Response {
	return Response{
		Status: status,
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"Connection":   "close",
		},
		Body: []byte(body),
	}
}

func writeResponse(w io.Writer, res Response) error {
	_, err := w.Write(res.Bytes())
	return err
}

// --- CLI ---

const banner = `
 ▄· ▄▌      ▄▄▄  ▄• ▄▌
▐█▪██▌▪     ▀▄ █·█▪██▌
▐█▌▐█▪ ▄█▀▄ ▐▀▀▄ █▌▐█▌
 ▐█▀·.▐█▌.▐▌▐█•█▌▐█▄█▌
  ▀ •  ▀█▄▀▪.▀  ▀ ▀▀▀ 
`

const (
	reset  = "\033[0m"
	dim    = "\033[90m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	cyan   = "\033[36m"
)

func printBanner(addr string) {
	fmt.Print(cyan + banner + reset)
	fmt.Printf("\n%sserving on%s http://localhost%s\n\n", dim, reset, addr)
}

func statusColor(status int) string {
	switch {
	case status >= 500:
		return red
	case status >= 400:
		return yellow
	default:
		return green
	}
}

func logRequest(req Request, status int, took time.Duration) {
	log.Printf("%s%s%s  %s%-6s%s %-24s %s%d%s  %s%s%s",
		dim, time.Now().Format("15:04:05"), reset,
		cyan, req.Method, reset,
		req.Target,
		statusColor(status), status, reset,
		dim, took.Round(time.Microsecond), reset,
	)
}
