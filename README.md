<div align="center">

# Yoru 🐈‍⬛

<i>A basic HTTP server built from scratch in Go with raw TCP sockets</i>

</div>

---

## Usage

```
go run main.go
```

The server starts on port 6767.

## Test

```
curl localhost:6767/
curl localhost:6767/about
curl -X POST -d "hello" localhost:6767/echo
```

---

## Routes

| Method | Path | |
| --- | --- | --- |
| GET | `/` | Hello, world! |
| GET | `/about` | About page |
| POST | `/echo` | Returns the request body |
