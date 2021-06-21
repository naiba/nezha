package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
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

	m := new(dns.Msg)
	if ipv6 {
		m.SetQuestion(dns.Fqdn(url[0]), dns.TypeAAAA)
	} else {
		m.SetQuestion(dns.Fqdn(url[0]), dns.TypeA)
	}
	m.RecursionDesired = true

	dnsServers := []string{"2606:4700:4700::1001", "2001:4860:4860::8844", "2400:3200::1", "2400:3200:baba::1"}
	if !ipv6 {
		dnsServers = []string{"1.0.0.1", "8.8.4.4", "223.5.5.5", "223.6.6.6"}
	}

	var wg sync.WaitGroup
	var resolveLock sync.RWMutex
	var ipv4Resolved, ipv6Resolved bool

	wg.Add(len(dnsServers))
	go func() {

	}()
	for i := 0; i < len(dnsServers); i++ {
		go func(i int) {
			defer wg.Done()
			c := new(dns.Client)
			c.Timeout = time.Second * 3
			r, _, err := c.Exchange(m, net.JoinHostPort(dnsServers[i], "53"))
			if err != nil {
				return
			}
			resolveLock.Lock()
			defer resolveLock.Unlock()
			if ipv6 && ipv6Resolved {
				return
			}
			if !ipv6 && ipv4Resolved {
				return
			}
			for _, ans := range r.Answer {
				if ipv6 {
					if aaaa, ok := ans.(*dns.AAAA); ok {
						url[0] = "[" + aaaa.AAAA.String() + "]"
						ipv6Resolved = true
					}
				} else {
					if a, ok := ans.(*dns.A); ok {
						url[0] = a.A.String()
						ipv4Resolved = true
					}
				}
			}
		}(i)
	}
	wg.Wait()

	if ipv6 && !ipv6Resolved {
		return "", errors.New("the AAAA record not resolved")
	}

	if !ipv6 && !ipv4Resolved {
		return "", errors.New("the A record not resolved")
	}

	return strings.Join(url, ":"), nil
}
