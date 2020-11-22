// Copyright (c) 2019 Meng Huang (mhboy@outlook.com)
// This package is licensed under a MIT license that can be found in the LICENSE file.

package rum

import (
	"errors"
	"net/http"
	"strings"
	"sync"
)

const (
	options = 1 << iota
	get
	head
	post
	put
	delete
	trace
	connect
	patch
)

// ErrGroupExisted is the error returned by Group when registers a existed group.
var ErrGroupExisted = errors.New("Group Existed")

// ErrParamsKeyEmpty is the error returned by HandleFunc when the params key is empty.
var ErrParamsKeyEmpty = errors.New("Params key must be not empty")

// Mux is an HTTP request multiplexer.
type Mux struct {
	mut         sync.RWMutex
	prefixes    map[string]*prefix
	middlewares []http.Handler
	notFound    http.Handler
	group       string
	groups      map[string]*Mux
}

type prefix struct {
	prefix string
	m      map[string]*Entry
}

// Entry represents an HTTP HandlerFunc entry.
type Entry struct {
	handler http.Handler
	key     string
	match   []string
	params  map[string]string
	method  int
	get     http.Handler
	post    http.Handler
	put     http.Handler
	delete  http.Handler
	patch   http.Handler
	head    http.Handler
	options http.Handler
	trace   http.Handler
	connect http.Handler
}

// NewMux returns a new Mux.
func NewMux() *Mux {
	m := &Mux{
		prefixes: make(map[string]*prefix),
		groups:   make(map[string]*Mux),
	}
	return m
}

func newGroup(group string) *Mux {
	m := &Mux{
		prefixes: make(map[string]*prefix),
		groups:   make(map[string]*Mux),
		group:    group,
	}
	return m
}

// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := m.replace(r.URL.Path)
	m.mut.RLock()
	entry := m.searchEntry(path, w, r)
	m.mut.RUnlock()
	if entry != nil {
		m.serveEntry(entry, w, r)
		return
	}
	if m.notFound != nil {
		m.notFound.ServeHTTP(w, r)
		return
	}
	http.Error(w, "404 Not Found : "+r.URL.String(), http.StatusNotFound)
}

func (m *Mux) searchEntry(path string, w http.ResponseWriter, r *http.Request) *Entry {
	if entry := m.getHandlerFunc(path); entry != nil {
		return entry
	}
	for _, groupMux := range m.groups {
		if entry := groupMux.searchEntry(path, w, r); entry != nil {
			return entry
		}
	}
	return nil
}

func (m *Mux) serveEntry(entry *Entry, w http.ResponseWriter, r *http.Request) {
	if entry.method == 0 {
		m.serveHandler(entry.handler, w, r)
	} else if r.Method == "GET" && entry.method&get > 0 {
		m.serveHandler(entry.get, w, r)
	} else if r.Method == "POST" && entry.method&post > 0 {
		m.serveHandler(entry.post, w, r)
	} else if r.Method == "PUT" && entry.method&put > 0 {
		m.serveHandler(entry.put, w, r)
	} else if r.Method == "DELETE" && entry.method&delete > 0 {
		m.serveHandler(entry.delete, w, r)
	} else if r.Method == "PATCH" && entry.method&patch > 0 {
		m.serveHandler(entry.patch, w, r)
	} else if r.Method == "HEAD" && entry.method&head > 0 {
		m.serveHandler(entry.head, w, r)
	} else if r.Method == "OPTIONS" && entry.method&options > 0 {
		m.serveHandler(entry.options, w, r)
	} else if r.Method == "TRACE" && entry.method&trace > 0 {
		m.serveHandler(entry.trace, w, r)
	} else if r.Method == "CONNECT" && entry.method&connect > 0 {
		m.serveHandler(entry.connect, w, r)
	}
}

func (m *Mux) serveHandler(handler http.Handler, w http.ResponseWriter, r *http.Request) {
	m.middleware(w, r)
	if handler != nil {
		handler.ServeHTTP(w, r)
	}
}

func (m *Mux) getHandlerFunc(path string) *Entry {
	if prefix, key, ok := m.matchParams(path); ok {
		if entry, ok := m.prefixes[prefix].m[key]; ok {
			return entry
		}
	}
	return nil
}

// HandleFunc registers the handler function for the given pattern
// in the Mux.
func (m *Mux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) *Entry {
	return m.Handle(pattern, http.HandlerFunc(handler))
}

// Handle registers the handler for the given pattern
// in the Mux.
func (m *Mux) Handle(pattern string, handler http.Handler) *Entry {
	m.mut.Lock()
	defer m.mut.Unlock()
	pattern = m.replace(pattern)
	pre, key, match, params := m.parseParams(m.group + pattern)
	if v, ok := m.prefixes[pre]; ok {
		if entry, ok := v.m[key]; ok {
			entry.handler = handler
			entry.key = key
			entry.match = match
			entry.params = params
			m.prefixes[pre].m[key] = entry
			return entry
		}
		entry := &Entry{}
		entry.handler = handler
		entry.key = key
		entry.match = match
		entry.params = params
		m.prefixes[pre].m[key] = entry
		return entry
	}
	m.prefixes[pre] = &prefix{m: make(map[string]*Entry), prefix: pre}
	entry := &Entry{}
	entry.handler = handler
	entry.key = key
	entry.match = match
	entry.params = params
	m.prefixes[pre].m[key] = entry
	return entry
}

// Group registers a group for the given pattern in the Mux.
func (m *Mux) Group(group string, f func(m *Mux)) {
	m.mut.Lock()
	defer m.mut.Unlock()
	group = m.replace(group)
	groupMux := newGroup(group)
	f(groupMux)
	if _, ok := m.groups[group]; ok {
		panic(ErrGroupExisted)
	}
	groupMux.middlewares = m.middlewares
	m.groups[group] = groupMux
}

// NotFound registers the not found handler function in the Mux.
func (m *Mux) NotFound(handler http.HandlerFunc) {
	m.mut.Lock()
	defer m.mut.Unlock()
	m.notFound = handler
}

// Use uses middleware.
func (m *Mux) Use(handler http.HandlerFunc) {
	m.mut.Lock()
	defer m.mut.Unlock()
	m.middlewares = append(m.middlewares, handler)
}

func (m *Mux) middleware(w http.ResponseWriter, r *http.Request) {
	for _, handler := range m.middlewares {
		handler.ServeHTTP(w, r)
	}
}

// Params returns http request params.
func (m *Mux) Params(r *http.Request) map[string]string {
	params := make(map[string]string)
	path := m.replace(r.URL.Path)
	m.mut.RLock()
	defer m.mut.RUnlock()
	if prefix, key, ok := m.matchParams(path); ok {
		if entry, ok := m.prefixes[prefix].m[key]; ok &&
			len(entry.match) > 0 && len(path) > len(prefix) {
			strs := strings.Split(path[len(prefix):], "/")
			if len(strs) == len(entry.match) {
				for i := 0; i < len(strs); i++ {
					if entry.match[i] != "" {
						params[entry.match[i]] = strs[i]
					}
				}
			}
		}
	}
	return params
}

func (m *Mux) matchParams(path string) (string, string, bool) {
	for _, p := range m.prefixes {
		if strings.HasPrefix(path, p.prefix) {
			r := path[len(p.prefix):]
			if r == "" {
				return p.prefix, "", true
			}
			for _, v := range p.m {
				count := strings.Count(r, "/")
				if count+1 == len(v.match) {
					form := strings.Split(r, "/")
					key := ""
					for i := 0; i < len(form); i++ {
						if v.match[i] != "" {
							if i > 0 {
								key += "/:"
							} else {
								key += ":"
							}
						} else {
							key += "/" + form[i]
						}
					}
					if key == v.key {
						return p.prefix, v.key, true
					}
				}
			}
		}
	}
	return "", "", false
}

func (m *Mux) parseParams(pattern string) (string, string, []string, map[string]string) {
	prefix := ""
	var match []string
	key := ""
	params := make(map[string]string)
	if strings.Contains(pattern, ":") {
		idx := strings.Index(pattern, ":")
		prefix = pattern[:idx]
		if idx+1 == len(pattern) || strings.Contains(pattern, ":/") {
			panic(ErrParamsKeyEmpty)
		}
		match = strings.Split(pattern[idx:], "/")
		for i := 0; i < len(match); i++ {
			if strings.Contains(match[i], ":") {
				match[i] = strings.Trim(match[i], ":")
				params[match[i]] = ""
				if i > 0 {
					key += "/:"
				} else {
					key += ":"
				}
			} else {
				key += "/" + match[i]
				match[i] = ""
			}
		}
	} else {
		prefix = pattern
	}
	return prefix, key, match, params
}

func (m *Mux) replace(s string) string {
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	return s
}

// GET adds a GET HTTP method for the entry.
func (entry *Entry) GET() *Entry {
	entry.method |= get
	entry.get = entry.handler
	return entry
}

// POST adds a POST HTTP method for the entry.
func (entry *Entry) POST() *Entry {
	entry.method |= post
	entry.post = entry.handler
	return entry
}

// PUT adds a PUT HTTP method for the entry.
func (entry *Entry) PUT() *Entry {
	entry.method |= put
	entry.put = entry.handler
	return entry
}

// DELETE adds a DELETE HTTP method for the entry.
func (entry *Entry) DELETE() *Entry {
	entry.method |= delete
	entry.delete = entry.handler
	return entry
}

// PATCH adds a PATCH HTTP method for the entry.
func (entry *Entry) PATCH() *Entry {
	entry.method |= patch
	entry.patch = entry.handler
	return entry
}

// HEAD adds a HEAD HTTP method for the entry.
func (entry *Entry) HEAD() *Entry {
	entry.method |= head
	entry.head = entry.handler
	return entry
}

// OPTIONS adds a OPTIONS HTTP method for the entry.
func (entry *Entry) OPTIONS() *Entry {
	entry.method |= options
	entry.options = entry.handler
	return entry
}

// TRACE adds a TRACE HTTP method for the entry.
func (entry *Entry) TRACE() *Entry {
	entry.method |= trace
	entry.trace = entry.handler
	return entry
}

// CONNECT adds a CONNECT HTTP method for the entry.
func (entry *Entry) CONNECT() *Entry {
	entry.method |= connect
	entry.connect = entry.handler
	return entry
}

// All adds all HTTP method for the entry.
func (entry *Entry) All() {
	entry.GET()
	entry.POST()
	entry.HEAD()
	entry.OPTIONS()
	entry.PUT()
	entry.PATCH()
	entry.DELETE()
	entry.TRACE()
	entry.CONNECT()
}
