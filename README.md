# rum
Package rum implements an HTTP server.

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
#### Example
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

#### Fast Example
```go
package main

import (
	"github.com/hslam/rum"
	"net/http"
)

func main() {
	m := rum.New()
	m.SetFast(true)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	m.Run(":8080")
}
```

#### Netpoll Example
```go
package main

import (
	"github.com/hslam/rum"
	"net/http"
)

func main() {
	m := rum.New()
	m.SetPoll(true)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	m.Run(":8080")
}
```

#### Netpoll Fast Example
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
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	m.Run(":8080")
}
```

curl -XGET http://localhost:8080
```
Hello World
```

### License
This package is licensed under a MIT license (Copyright (c) 2020 Meng Huang)


### Author
rum was written by Meng Huang.


