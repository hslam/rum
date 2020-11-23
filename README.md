# rum
Package rum implements an HTTP server. The rum server is compatible with net/http and faster than net/http.

## [Benchmark](http://github.com/hslam/http-benchmark "http-benchmark")
<img src="https://raw.githubusercontent.com/hslam/http-benchmark/master/http-qps.png" width = "400" height = "300" alt="qps" align=center><img src="https://raw.githubusercontent.com/hslam/http-benchmark/master/http-p99.png" width = "400" height = "300" alt="p99" align=center>


## Features
* Fully compatible with the http.HandlerFunc interface.
* Support other router that implements the http.Handler interface.
* [Epoll / Kqueue](https://github.com/hslam/netpoll "netpoll")
* [HTTP request](https://github.com/hslam/request "request")
* [HTTP response](https://github.com/hslam/request "response")
* [HTTP request multiplexer](https://github.com/hslam/mux "mux")
* [HTTP handler](https://github.com/hslam/handler "handler")

## Get started

### Install
```
go get github.com/hslam/rum
```
### Import
```
import "github.com/hslam/rum"
```

### Usage
#### Simple Example
```go
package main

import (
	"github.com/hslam/rum"
	"net/http"
)

func main() {
	m := rum.New()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	m.Run(":8080")
}
```

curl http://localhost:8080
```
Hello World
```

#### Example
```go
package main

import (
	"github.com/hslam/rum"
	"net/http"
)

func main() {
	m := rum.New()
	m.SetFast(true)
	m.SetPoll(true)
	m.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found : "+r.URL.String(), http.StatusNotFound)
	})
	m.Use(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	})
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	m.HandleFunc("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
		params := m.Params(r)
		w.Write([]byte("Hello " + params["name"]))
	}).GET()
	m.Group("/group", func(m *rum.Mux) {
		m.HandleFunc("/foo/:id", func(w http.ResponseWriter, r *http.Request) {
			params := m.Params(r)
			w.Write([]byte("group/foo id:" + params["id"]))
		}).GET()
		m.HandleFunc("/bar/:id", func(w http.ResponseWriter, r *http.Request) {
			params := m.Params(r)
			w.Write([]byte("group/bar id:" + params["id"]))
		}).GET()
	})
	m.Run(":8080")
}
```

curl http://localhost:8080/hello/rum
```
Hello rum
```

curl http://localhost:8080/group/foo/1
```
group/foo id:1
```

curl http://localhost:8080/group/bar/2
```
group/bar id:2
```

#### Use Other Router Example
The router must implement the http.Handler interface, for example using the http.ServeMux.
```go
package main

import (
	"github.com/hslam/rum"
	"net/http"
)

func main() {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	server := rum.New()
	server.Handler = router
	server.Run(":8080")
}
```

### License
This package is licensed under a MIT license (Copyright (c) 2020 Meng Huang)

### Author
rum was written by Meng Huang.
