package kcrawl

import "net/url"

type ProxyPool interface {
	GetHttpProxy() *url.URL
	GetSock5Proxy() *url.URL
}
