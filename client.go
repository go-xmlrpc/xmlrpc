package xmlrpc

import (
	"bytes"
	"net"
	"net/http"
	"time"

	"gopkg.in/scgi.v0"
)

func init() {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	transport.RegisterProtocol("scgi", &scgi.Client{})

	DefaultTransport = transport
}

// DefaultTransport is copied from the go1.11.4 net/http documentation so we can
// create a transport which also supports scgi.
var DefaultTransport http.RoundTripper

func NewClient(target string) *Client {
	httpClient := &http.Client{
		Transport: DefaultTransport,
	}

	return &Client{
		target: target,
		client: httpClient,
	}
}

type Client struct {
	target string
	client *http.Client
}

func (c *Client) Call(name string, args ...interface{}) ([]interface{}, error) {
	data, err := Marshal(name, args...)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Post(c.target, "text/xml", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out, err := Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	return out, nil
}
