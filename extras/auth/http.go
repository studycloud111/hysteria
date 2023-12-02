package auth

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/apernet/hysteria/core/server"
)

const (
	httpAuthTimeout = 10 * time.Second
)

var _ server.Authenticator = &HTTPAuthenticator{}

var errInvalidStatusCode = errors.New("invalid status code")

type HTTPAuthenticator struct {
	Client *http.Client
	URL    string
}

// NewHTTPAuthenticator 创建一个新的HTTPAuthenticator
func NewHTTPAuthenticator(url string, insecure bool) *HTTPAuthenticator {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: insecure,
	}
	return &HTTPAuthenticator{
		Client: &http.Client{
			Transport: tr,
			Timeout:   httpAuthTimeout,
		},
		URL: url,
	}
}

type httpAuthRequest struct {
	Addr string `json:"addr"`
	Auth string `json:"auth"`
	Tx   uint64 `json:"tx"`
}

type httpAuthResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id"`
}

// post 向配置的URL发送认证请求
func (a *HTTPAuthenticator) post(req *httpAuthRequest) (*httpAuthResponse, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(req)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Post(a.URL, "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: received status code %d", errInvalidStatusCode, resp.StatusCode)
	}

	var authResp httpAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}
	return &authResp, nil
}

// Authenticate 实现Authenticator接口的方法
func (a *HTTPAuthenticator) Authenticate(addr net.Addr, auth string, tx uint64) (ok bool, id string) {
	req := &httpAuthRequest{
		Addr: addr.String(),
		Auth: auth,
		Tx:   tx,
	}
	resp, err := a.post(req)
	if err != nil {
		// 可以在这里添加日志记录
		return false, ""
	}
	return resp.OK, resp.ID
}
