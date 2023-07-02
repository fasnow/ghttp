package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// 超时和代理的优先级
// Do() > 全局 > 初始化 > 默认

var (
	UserAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.37",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/113.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36 OPR/98.0.0.0 (Edition beta)",
	}
)

var (
	globalProxyEnabled   bool
	globalTimeoutEnabled bool
	globalProxy          *url.URL
	globalTimeout        time.Duration
	defaultHttpTimeout   = 20 * time.Second
)

type Client struct {
	client        *http.Client
	mutex         sync.Mutex
	globalProxy   *url.URL
	globalTimeout time.Duration
	Proxy         *url.URL
	Timeout       time.Duration
	Context       *context.Context
}

type Options struct {
	Proxy   *url.URL
	Timeout time.Duration
	Context *context.Context
}

func (g *Client) new(timeout ...time.Duration) *http.Client {
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if len(timeout) > 0 && timeout[0] > 0 {
		return &http.Client{
			Transport: transCfg,   // disable tls verify
			Timeout:   timeout[0], //必须设置一个超时，不然程序会抛出非自定义错误
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}, //不跳转302
		}
	}
	return &http.Client{
		Transport: transCfg,           // disable tls verify
		Timeout:   defaultHttpTimeout, //必须设置一个超时，不然程序会抛出非自定义错误
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (g *Client) Do(request *http.Request, option ...Options) (*http.Response, error) {
	//只有第一次调用时会执行
	if g.client == nil {
		g.client = g.new()
	}

	g.mutex.Lock()
	if globalTimeoutEnabled {
		g.client.Timeout = globalTimeout
	} else {
		if g.Timeout > 0 {
			g.client.Timeout = g.Timeout
		} else {
			g.client.Timeout = defaultHttpTimeout
		}
	}
	if globalProxyEnabled {
		g.client.Transport.(*http.Transport).Proxy = http.ProxyURL(globalProxy)
	} else {
		g.client.Transport.(*http.Transport).Proxy = http.ProxyURL(g.Proxy)
	}
	if _, ok := request.Header["User-Agent"]; !ok {
		request.Header.Set("User-Agent", UserAgents[rand.Intn(len(UserAgents))])
	}
	if g.Context != nil {
		request.WithContext(*g.Context)
	}
	//设置了参数则只对本次请求生成
	if len(option) > 0 {
		var client = g.new()
		currentProxy := g.client.Transport.(*http.Transport).Proxy
		op := option[0]
		newTimeout := op.Timeout
		newProxy := op.Proxy
		if newTimeout > 0 {
			client.Timeout = newTimeout
		} else {
			if globalTimeoutEnabled {
				client.Timeout = globalTimeout
			} else {
				if g.Timeout > 0 {
					client.Timeout = g.Timeout
				} else {
					client.Timeout = defaultHttpTimeout
				}
			}
		}
		if newProxy != nil {
			client.Transport.(*http.Transport).Proxy = http.ProxyURL(newProxy)
		} else {
			if globalProxyEnabled {
				client.Transport.(*http.Transport).Proxy = http.ProxyURL(globalProxy)
			} else {
				client.Transport.(*http.Transport).Proxy = currentProxy
			}
		}
		if _, ok := request.Header["User-Agent"]; !ok {
			request.Header.Set("User-Agent", UserAgents[rand.Intn(len(UserAgents))])
		}
		if op.Context != nil {
			request.WithContext(*op.Context)
		}
		g.mutex.Unlock()
		return client.Do(request)
	}
	g.mutex.Unlock()
	return g.client.Do(request)
}

func GetResponseBody(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	reader := bufio.NewReader(body)
	var result []byte
	for {
		buffer := make([]byte, 1024)
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		result = append(result, buffer[:n]...)
	}
	return result, nil
}

func SetGlobalProxy(proxy string) error {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		globalProxy = nil
		globalProxyEnabled = false
	} else {
		newProxyURL, err := url.Parse(proxy)
		if err != nil {
			return err
		}
		globalProxyEnabled = true
		globalProxy = newProxyURL
	}
	return nil
}

func SetGlobalTimeout(timeout time.Duration) {
	if timeout > 0 {
		globalTimeout = defaultHttpTimeout
		globalTimeoutEnabled = false
		return
	} else {
		globalTimeout = timeout
		globalTimeoutEnabled = true
	}
}
