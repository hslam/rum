// Copyright (c) 2020 Meng Huang (mhboy@outlook.com)
// This package is licensed under a MIT license that can be found in the LICENSE file.

// Package rum implements an HTTP server.
package rum

import (
	"bufio"
	"github.com/hslam/netpoll"
	"github.com/hslam/request"
	"github.com/hslam/response"
	"net"
	"net/http"
	"sync"
)

// DefaultServer is the default HTTP server.
var DefaultServer = New()

// Rum is an HTTP server.
type Rum struct {
	*Mux
	Handler   http.Handler
	fast      bool
	poll      bool
	mut       sync.Mutex
	listeners []net.Listener
	pollers   []*netpoll.Server
}

// New returns a new Rum instance.
func New() *Rum {
	return &Rum{Mux: NewMux()}
}

// SetFast enables the Server to use simple request parser.
func (m *Rum) SetFast(fast bool) {
	m.fast = fast
}

// SetPoll enables the Server to use netpoll based on epoll/kqueue.
func (m *Rum) SetPoll(poll bool) {
	m.poll = poll
}

// Run listens on the TCP network address addr and then calls
// Serve with m to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
//
// Run always returns a non-nil error.
func (m *Rum) Run(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return m.Serve(ln)
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each, or registering the conn fd to poll
// that will trigger the fd to read requests and then call handler
// to reply to them.
func (m *Rum) Serve(l net.Listener) error {
	if m.poll {
		var handler = m.Handler
		if handler == nil {
			handler = m
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
		if m.fast {
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
		poller := &netpoll.Server{
			Handler: h,
		}
		m.mut.Lock()
		m.pollers = append(m.pollers, poller)
		m.mut.Unlock()
		return poller.Serve(l)
	}
	m.mut.Lock()
	m.listeners = append(m.listeners, l)
	m.mut.Unlock()
	if m.fast {
		for {
			conn, err := l.Accept()
			if err != nil {
				return err
			}
			go m.serveFastConn(conn)
		}
	} else {
		for {
			conn, err := l.Accept()
			if err != nil {
				return err
			}
			go m.serveConn(conn)
		}
	}
}

// Close closes the HTTP server.
func (m *Rum) Close() error {
	m.mut.Lock()
	defer m.mut.Unlock()
	for _, lis := range m.listeners {
		lis.Close()
	}
	m.listeners = []net.Listener{}
	for _, poller := range m.pollers {
		poller.Close()
	}
	m.pollers = []*netpoll.Server{}
	m.Handler = nil
	return nil
}

func (m *Rum) serveConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	rw := bufio.NewReadWriter(reader, bufio.NewWriter(conn))
	var err error
	var req *http.Request
	var handler = m.Handler
	if handler == nil {
		handler = m
	}
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
}

func (m *Rum) serveFastConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	rw := bufio.NewReadWriter(reader, bufio.NewWriter(conn))
	var err error
	var req *http.Request
	var handler = m.Handler
	if handler == nil {
		handler = m
	}
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
	rum := DefaultServer
	rum.Handler = handler
	rum.SetFast(fast)
	return rum.Run(addr)
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
	rum := DefaultServer
	rum.Handler = handler
	rum.SetFast(fast)
	rum.SetPoll(true)
	return rum.Run(addr)
}
