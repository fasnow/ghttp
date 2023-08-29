package ghttp

import (
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestGlobalTimeout(t *testing.T) {
	client1 := Client{Timeout: 1 * time.Microsecond}
	SetGlobalTimeout(1 * time.Microsecond)
	req1, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	resp1, err := client1.Do(req1)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp1.Status)
	}

	SetGlobalTimeout(1 * time.Second)
	client2 := Client{Timeout: 1 * time.Second}
	req2, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	rsp2, err := client2.Do(req2)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rsp2.Status)
}

func TestOptionalTimeout1(t *testing.T) {
	client1 := Client{Timeout: 1 * time.Microsecond}
	req1, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	resp1, err := client1.Do(req1)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp1.Status)
	}

	client2 := Client{Timeout: 1 * time.Second}
	req2, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	rsp2, err := client2.Do(req2)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rsp2.Status)
}

func TestOptionalTimeout2(t *testing.T) {
	client1 := Client{}
	req1, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	resp1, err := client1.Do(req1, SetTimeout(1*time.Microsecond))
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp1.Status)
	}

	client2 := Client{}
	req2, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	rsp2, err := client2.Do(req2, SetTimeout(1*time.Microsecond))
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rsp2.Status)
}

func TestOptionalProxy1(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:8081")
	client1 := Client{Proxy: u}
	req1, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	resp1, err := client1.Do(req1)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp1.Status)
	}

	client2 := Client{Proxy: nil}
	req2, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	rsp2, err := client2.Do(req2)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rsp2.Status)
}

func TestOptionalProxy2(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:8081")
	client1 := Client{}
	req1, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	resp1, err := client1.Do(req1, SetProxy(u))
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp1.Status)
	}

	client2 := Client{}
	req2, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
	rsp2, err := client2.Do(req2, SetProxy(nil))
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(rsp2.Status)
}

func TestRedirect(t *testing.T) {
	client1 := &Client{Redirect: false}
	req1, _ := http.NewRequest("GET", "http://www.baidu.com", nil)
	resp1, err := client1.Do(req1)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(resp1.StatusCode)
	}
}

func Test(t *testing.T) {
	u1, _ := url.Parse("http://127.0.0.1:8081")
	u2, _ := url.Parse("http://127.0.0.1:8080")
	_ = SetGlobalProxy("http://127.0.0.1:8080")
	SetGlobalTimeout(1 * time.Second)
	client1 := &Client{
		Proxy:               u1,
		Timeout:             1 * time.Microsecond,
		Context:             nil,
		StopWhenContextDone: false,
	}
	var wg = &sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			req1, _ := http.NewRequest("GET", "https://www.github.com", nil)
			resp1, err := client1.Do(req1, SetProxy(u2))
			if err != nil {
				t.Log(err)
			} else {
				t.Log(resp1.Status)
			}
		}()
	}
	wg.Wait()
	//resp1, err = client1.Do(req1, Options{Proxy: nil})
	//if err != nil {
	//	t.Log(err)
	//} else {
	//	t.Log(resp1.Status)
	//}
	//
	//resp1, err = client1.Do(req1, Options{Proxy: nil})
	//if err != nil {
	//	t.Log(err)
	//} else {
	//	t.Log(resp1.Status)
	//}
}
