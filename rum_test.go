package rum

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func testHTTP(method, url string, status int, result string, t *testing.T) {
	var req *http.Request
	req, _ = http.NewRequest(method, url, nil)
	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:   1,
			DisableKeepAlives: true,
		},
	}
	if resp, err := client.Do(req); err != nil {
		t.Error(err)
	} else if resp.StatusCode != status {
		t.Error(resp.StatusCode)
	} else if body, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Error(err)
	} else if string(body) != result {
		t.Error(string(body))
	}
}

func TestRum(t *testing.T) {
	addr := ":8080"
	m := New()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	done := make(chan struct{})
	go func() {
		m.Run(addr)
		close(done)
	}()
	time.Sleep(time.Millisecond * 10)
	testHTTP("GET", "http://"+addr+"/", http.StatusOK, "Hello World", t)
	m.Close()
	<-done
}

func TestFastRum(t *testing.T) {
	addr := ":8080"
	m := New()
	m.SetFast(true)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	done := make(chan struct{})
	go func() {
		m.Run(addr)
		close(done)
	}()
	time.Sleep(time.Millisecond * 10)
	testHTTP("GET", "http://"+addr+"/", http.StatusOK, "Hello World", t)
	m.Close()
	<-done
}

func TestRumPoll(t *testing.T) {
	addr := ":8080"
	m := New()
	m.SetPoll(true)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	done := make(chan struct{})
	go func() {
		m.Run(addr)
		close(done)
	}()
	time.Sleep(time.Millisecond * 10)
	testHTTP("GET", "http://"+addr+"/", http.StatusOK, "Hello World", t)
	m.Close()
	<-done
}

func TestFastRumPoll(t *testing.T) {
	addr := ":8080"
	m := New()
	m.SetFast(true)
	m.SetPoll(true)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	done := make(chan struct{})
	go func() {
		m.Run(addr)
		close(done)
	}()
	time.Sleep(time.Millisecond * 10)
	testHTTP("GET", "http://"+addr+"/", http.StatusOK, "Hello World", t)
	m.Close()
	<-done
}
