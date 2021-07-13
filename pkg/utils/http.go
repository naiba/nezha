package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

func NewSingleStackHTTPClient(httpTimeout, dialTimeout, keepAliveTimeout time.Duration, ipv6 bool) *http.Client {
	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: keepAliveTimeout,
	}

	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			ip, err := resolveIP(addr, ipv6)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, ip)
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   httpTimeout,
	}
}

func resolveIP(addr string, ipv6 bool) (string, error) {
	url := strings.Split(addr, ":")

	dnsServers := []string{"[2606:4700:4700::1001]", "[2001:4860:4860::8844]", "[2400:3200::1]", "[2400:3200:baba::1]"}
	if !ipv6 {
		dnsServers = []string{"1.0.0.1", "8.8.4.4", "223.5.5.5", "223.6.6.6"}
	}

	res, err := net.LookupIP(url[0])
	if err != nil {
		for i := 0; i < len(dnsServers); i++ {
			r := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: time.Second * 10,
					}
					return d.DialContext(ctx, "udp", dnsServers[i]+":53")
				},
			}
			res, err = r.LookupIP(context.Background(), "ip", url[0])
			if err == nil {
				break
			}
		}
	}

	if err != nil {
		return "", err
	}

	var ipv4Resolved, ipv6Resolved bool

	for i := 0; i < len(res); i++ {
		ip := res[i].String()
		if strings.Contains(ip, ".") && !ipv6 {
			ipv4Resolved = true
			url[0] = ip
			break
		}
		if strings.Contains(ip, ":") && ipv6 {
			ipv6Resolved = true
			url[0] = "[" + ip + "]"
			break
		}
	}

	if ipv6 && !ipv6Resolved {
		return "", errors.New("the AAAA record not resolved")
	}

	if !ipv6 && !ipv4Resolved {
		return "", errors.New("the A record not resolved")
	}

	return strings.Join(url, ":"), nil
}
