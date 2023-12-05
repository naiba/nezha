package utils

import (
	"crypto/tls"
	"net/http"
	"time"
)

var (
	HttpClientSkipTlsVerify *http.Client
	HttpClient              *http.Client
)

func init() {
	HttpClientSkipTlsVerify = httpClient(_httpClient{
		Transport: httpTransport(_httpTransport{
			VerifySSL: true,
		}),
	})
	HttpClient = httpClient(_httpClient{
		Transport: httpTransport(_httpTransport{
			VerifySSL: false,
		}),
	})
}

type _httpTransport struct {
	VerifySSL bool
}

func httpTransport(conf _httpTransport) *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: conf.VerifySSL},
		Proxy:           http.ProxyFromEnvironment,
	}
}

type _httpClient struct {
	Transport *http.Transport
}

func httpClient(conf _httpClient) *http.Client {
	return &http.Client{
		Transport: conf.Transport,
		Timeout:   time.Minute * 10,
	}
}
