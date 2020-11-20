// Copyright (c) 2020 Meng Huang (mhboy@outlook.com)
// This package is licensed under a MIT license that can be found in the LICENSE file.

// Package rum implements an HTTP server.
package rum

import (
	"bufio"
	"github.com/hslam/mux"
	"github.com/hslam/netpoll"
	"github.com/hslam/request"
	"github.com/hslam/response"
	"net"
	"net/http"
	"sync"
)

// NewServeMux allocates and returns a new Mux.
func NewServeMux() *Rum { return New() }

// DefaultServeMux is the default Mux used by ListenAndServe.
var DefaultServeMux = New()

// Rum is an HTTP server.
type Rum struct {
	*mux.Mux
	fast bool
}

// New returns a new Rum instance.
func New() *Rum {
	return &Rum{Mux: mux.New()}
}

// NewFast returns a new Rum instance but with the simple request parser.
func NewFast() *Rum {
	return &Rum{Mux: mux.New(), fast: true}
}

// Run listens on the TCP network address addr and then calls
// Serve with m to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
//
// Run always returns a non-nil error.
func (m *Rum) Run(addr string) error {
	return listenAndServe(addr, m, m.fast)
}

// RunPoll is like Run but based on epoll/kqueue.
func (m *Rum) RunPoll(addr string) error {
	return listenAndServePoll(addr, m, m.fast)
}

// ListenAndServe listens on the TCP network address addr and then calls
// Serve with handler to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
//
// The handler is typically nil, in which case the DefaultServeMux is used.
//
// ListenAndServe always returns a non-nil error.
func ListenAndServe(addr string, handler http.Handler) error {
	return listenAndServe(addr, handler, false)
}

// ListenAndServeFast is like ListenAndServe but with the simple request parser.
func ListenAndServeFast(addr string, handler http.Handler) error {
	return listenAndServe(addr, handler, true)
}

func listenAndServe(addr string, handler http.Handler, fast bool) error {
	if handler == nil {
		handler = DefaultServeMux
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if fast {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return err
			}
			go func(conn net.Conn) {
				reader := bufio.NewReader(conn)
				rw := bufio.NewReadWriter(reader, bufio.NewWriter(conn))
				var err error
				var req *http.Request
				for {
					req, err = request.ReadFastRequest(reader)
					if err != nil {
						break
					}
					res := response.NewResponse(req, conn, rw)
					handler.ServeHTTP(res, req)
					res.FinishRequest()
					request.FreeRequest(req)
					response.FreeResponse(res)
				}
			}(conn)
		}
	} else {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return err
			}
			go func(conn net.Conn) {
				reader := bufio.NewReader(conn)
				rw := bufio.NewReadWriter(reader, bufio.NewWriter(conn))
				var err error
				var req *http.Request
				for {
					req, err = http.ReadRequest(reader)
					if err != nil {
						break
					}
					res := response.NewResponse(req, conn, rw)
					handler.ServeHTTP(res, req)
					res.FinishRequest()
					response.FreeResponse(res)
				}
			}(conn)
		}
	}
}

// ListenAndServePoll is like ListenAndServe but based on epoll/kqueue.
func ListenAndServePoll(addr string, handler http.Handler) error {
	return listenAndServePoll(addr, handler, false)
}

// ListenAndServePollFast is like ListenAndServePoll but with the simple request parser.
func ListenAndServePollFast(addr string, handler http.Handler) error {
	return listenAndServePoll(addr, handler, true)
}

func listenAndServePoll(addr string, handler http.Handler, fast bool) error {
	if handler == nil {
		handler = DefaultServeMux
	}
	var h = &netpoll.ConnHandler{}
	type Context struct {
		reader  *bufio.Reader
		rw      *bufio.ReadWriter
		conn    net.Conn
		serving sync.Mutex
	}
	h.SetUpgrade(func(conn net.Conn) (netpoll.Context, error) {
		reader := bufio.NewReader(conn)
		rw := bufio.NewReadWriter(reader, bufio.NewWriter(conn))
		return &Context{reader: reader, conn: conn, rw: rw}, nil
	})
	if fast {
		h.SetServe(func(context netpoll.Context) error {
			ctx := context.(*Context)
			var err error
			var req *http.Request
			ctx.serving.Lock()
			req, err = request.ReadFastRequest(ctx.reader)
			if err != nil {
				ctx.serving.Unlock()
				return err
			}
			res := response.NewResponse(req, ctx.conn, ctx.rw)
			handler.ServeHTTP(res, req)
			res.FinishRequest()
			ctx.serving.Unlock()
			request.FreeRequest(req)
			response.FreeResponse(res)
			return nil
		})
	} else {
		h.SetServe(func(context netpoll.Context) error {
			ctx := context.(*Context)
			var err error
			var req *http.Request
			ctx.serving.Lock()
			req, err = http.ReadRequest(ctx.reader)
			if err != nil {
				ctx.serving.Unlock()
				return err
			}
			res := response.NewResponse(req, ctx.conn, ctx.rw)
			handler.ServeHTTP(res, req)
			res.FinishRequest()
			ctx.serving.Unlock()
			response.FreeResponse(res)
			return nil
		})
	}
	return netpoll.ListenAndServe("tcp", addr, h)
}
