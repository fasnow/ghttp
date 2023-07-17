package ghttp

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
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.37",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/113.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36 OPR/98.0.0.0 (Edition beta)",
	}
)

var (
	globalProxyEnabled   bool
	globalTimeoutEnabled bool

	globalProxyMutex sync.Mutex
	globalProxy      *url.URL

	globalTimeoutMutex sync.Mutex
	globalTimeout      time.Duration

	defaultHttpTimeoutMutex sync.Mutex
	defaultHttpTimeout      = 20 * time.Second
)

// Client 超时和代理的优先级
// Do() > 全局 > 初始化 > 默认
type Client struct {
	http                *http.Client
	mutex               sync.Mutex
	globalProxy         *url.URL
	globalTimeout       time.Duration
	Proxy               *url.URL
	Timeout             time.Duration
	Context             *context.Context
	StopWhenContextDone bool
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
	g.mutex.Lock()
	//只有第一次调用时会执行
	if g.http == nil {
		g.http = g.new()
	}
	if globalTimeoutEnabled {
		g.http.Timeout = globalTimeout
	} else {
		if g.Timeout > 0 {
			g.http.Timeout = g.Timeout
		} else {
			g.http.Timeout = defaultHttpTimeout
		}
	}
	if globalProxyEnabled {
		g.http.Transport.(*http.Transport).Proxy = http.ProxyURL(globalProxy)
	} else {
		g.http.Transport.(*http.Transport).Proxy = http.ProxyURL(g.Proxy)
	}
	if _, ok := request.Header["User-Agent"]; !ok {
		request.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	}
	if g.Context != nil {
		request.WithContext(*g.Context)
	}
	//设置了参数则只对本次请求生成
	if len(option) > 0 {
		var httpClient = g.new()
		currentProxy := g.http.Transport.(*http.Transport).Proxy
		op := option[0]
		newTimeout := op.Timeout
		newProxy := op.Proxy
		if newTimeout > 0 {
			httpClient.Timeout = newTimeout //本次请求
		} else {
			if globalTimeoutEnabled {
				httpClient.Timeout = globalTimeout //全局超时
			} else {
				if g.Timeout > 0 {
					httpClient.Timeout = g.Timeout //初始化超时
				} else {
					httpClient.Timeout = defaultHttpTimeout //默认超时
				}
			}
		}
		if newProxy != nil {
			httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(newProxy) //本次请求
		} else {
			if globalProxyEnabled {
				httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(globalProxy) //全局代理
			} else {
				httpClient.Transport.(*http.Transport).Proxy = currentProxy //当前代理
			}
		}
		if _, ok := request.Header["User-Agent"]; !ok {
			request.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		}
		if op.Context != nil {
			request.WithContext(*op.Context)
		}
		g.mutex.Unlock()
		return httpClient.Do(request)
	}
	g.mutex.Unlock()
	if g.StopWhenContextDone {
		select {
		case <-(*g.Context).Done():
			return nil, (*g.Context).Err()
		default:
			return g.http.Do(request)
		}
	}
	return g.http.Do(request)
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

// SetGlobalProxy 全局代理,优先级最高
func SetGlobalProxy(proxy string) error {
	globalProxyMutex.Lock()
	defer globalProxyMutex.Unlock()
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

// SetGlobalTimeout 全局默认超时时间,优先级最高
func SetGlobalTimeout(timeout time.Duration) {
	globalTimeoutMutex.Lock()
	defer globalTimeoutMutex.Unlock()
	if timeout > 0 {
		globalTimeout = timeout
		globalTimeoutEnabled = true
		return
	} else {
		globalTimeout = defaultHttpTimeout
		globalTimeoutEnabled = false
	}
}

// SetDefaultTimeout 全局默认超时时间,优先级最低
func SetDefaultTimeout(timeout time.Duration) {
	defaultHttpTimeoutMutex.Lock()
	defer defaultHttpTimeoutMutex.Unlock()
	if timeout > 0 {
		defaultHttpTimeout = timeout
	}
}

func GetOptionalUserAgents() []string {
	return userAgents
}
