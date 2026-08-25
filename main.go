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
	ln, err := net.Listen("tcp", ":6767")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Println("listening on :6767")

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

func route(conn net.Conn, req Request) {
	var err error

	switch req.Target {
	case "/":
		err = writeResponse(conn, newResponse(200, "Hello, world!"))
	case "/about":
		err = writeResponse(conn, newResponse(200, "About Page"))
	case "/echo":
		switch req.Method {
		case "POST":
			err = writeResponse(conn, newResponse(200, string(req.Body)))
		default:
			err = writeResponse(conn, newResponse(405, "Method Now Allowed"))
		}
	default:
		err = writeResponse(conn, newResponse(404, "Not Found"))
	}

	if err != nil {
		log.Println("write:", err)
	}
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
