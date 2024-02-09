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

	once sync.Once
)

// Client 超时和代理的优先级
// Do() > 全局 > 初始化 > 默认
type Client struct {
	http                *http.Client
	mutex               sync.Mutex
	globalProxy         *url.URL
	globalTimeout       time.Duration
	StopWhenContextDone bool
	Proxy               *url.URL
	Timeout             time.Duration
	Context             *context.Context
	Redirect            bool
}

func (g *Client) redirect(req *http.Request, via []*http.Request) error {
	if g.Redirect {
		return nil
	}
	return http.ErrUseLastResponse
}

func (g *Client) new(timeout ...time.Duration) *http.Client {
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if len(timeout) > 0 && timeout[0] > 0 {
		return &http.Client{
			Transport:     transCfg,   // disable tls verify
			Timeout:       timeout[0], //必须设置一个超时，不然程序会抛出非自定义错误
			CheckRedirect: g.redirect,
		}
	}
	return &http.Client{
		Transport:     transCfg,           // disable tls verify
		Timeout:       defaultHttpTimeout, //必须设置一个超时，不然程序会抛出非自定义错误
		CheckRedirect: g.redirect,
	}
}

func (g *Client) Do(request *http.Request, options ...Options) (*http.Response, error) {
	g.mutex.Lock()
	once.Do(func() {
		g.http = g.new()
	})
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
	if len(options) > 0 {
		var httpClient = g.new()
		httpClient.Timeout = g.Timeout
		httpClient.Transport.(*http.Transport).Proxy = g.http.Transport.(*http.Transport).Proxy
		if _, ok := request.Header["User-Agent"]; !ok { //设置默认UA头
			request.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		}
		if g.Context != nil {
			request.WithContext(*g.Context)
		}
		applyOptions(httpClient, request, options...)
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

type Options func(c *http.Client, request *http.Request)

func SetTimeout(timeout time.Duration) Options {
	return func(c *http.Client, r *http.Request) {
		if timeout > 0 {
			c.Timeout = timeout //本次请求
		} else {
			if globalTimeoutEnabled {
				c.Timeout = globalTimeout //全局超时
			} else {
				if c.Timeout <= 0 {
					c.Timeout = defaultHttpTimeout //默认超时
				}
				//否则使用g.timeout
			}
		}
	}
}

func SetProxy(value *url.URL) Options {
	return func(c *http.Client, r *http.Request) {
		if value != nil { //是否新代理
			c.Transport.(*http.Transport).Proxy = http.ProxyURL(value) //本次请求
		} else {
			if globalProxyEnabled {
				c.Transport.(*http.Transport).Proxy = http.ProxyURL(globalProxy) //全局代理
			}
		}
	}
}

func SetContext(value *context.Context) Options {
	return func(c *http.Client, r *http.Request) {
		if value != nil {
			r.WithContext(*value)
		}
	}
}

func SetHeaders(headers []struct{ key, value string }) Options {
	return func(c *http.Client, r *http.Request) {
		for _, header := range headers {
			r.Header.Set(header.key, header.value)
		}
	}
}

func EnableRedirect(value bool) Options {
	return func(c *http.Client, r *http.Request) {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if value {
				return nil
			}
			return http.ErrUseLastResponse
		}
	}
}

func applyOptions(c *http.Client, r *http.Request, options ...Options) {
	for _, option := range options {
		option(c, r)
	}
}
