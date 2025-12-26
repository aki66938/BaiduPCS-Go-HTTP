package panhome

import (
	"net/url"

	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs/expires"
	"github.com/qjfoidnh/BaiduPCS-Go/requester"
)

const (
	// OperationSignature signature
	OperationSignature = "signature"
)

var (
	panBaiduComURL = &url.URL{
		Scheme: "https",
		Host:   "pan.baidu.com",
	}
	// PanHomeUserAgent PanHome User-Agent
	PanHomeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

type (
	PanHome struct {
		client *requester.HTTPClient
		ua     string
		bduss  string

		sign1, sign3 []rune
		timestamp    string

		signRes     SignRes
		signExpires expires.Expires
	}
)

func NewPanHome(client *requester.HTTPClient) *PanHome {
	ph := PanHome{}
	if client != nil {
		newC := *client
		ph.client = &newC
	}
	return &ph
}

func (ph *PanHome) lazyInit() {
	if ph.client == nil {
		ph.client = requester.NewHTTPClient()
	}
}
