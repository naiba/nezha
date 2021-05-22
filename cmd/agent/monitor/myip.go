package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type geoIP struct {
	CountryCode string `json:"country_code,omitempty"`
	IP          string `json:"ip,omitempty"`
}

var (
	ipv4Servers = []string{
		"https://api-ipv4.ip.sb/geoip",
		"https://ip4.seeip.org/geoip",
	}
	ipv6Servers = []string{
		"https://ip6.seeip.org/geoip",
		"https://api-ipv6.ip.sb/geoip",
	}
	cachedIP, cachedCountry string
	httpClientV4            = newHTTPClient(time.Second*20, time.Second*5, time.Second*10, false)
	httpClientV6            = newHTTPClient(time.Second*20, time.Second*5, time.Second*10, true)
)

func UpdateIP() {
	for {
		ipv4 := fetchGeoIP(ipv4Servers, false)
		ipv6 := fetchGeoIP(ipv6Servers, true)
		cachedIP = fmt.Sprintf("ip(v4:%s,v6:%s)", ipv4.IP, ipv6.IP)
		if ipv4.CountryCode != "" {
			cachedCountry = ipv4.CountryCode
		} else if ipv6.CountryCode != "" {
			cachedCountry = ipv6.CountryCode
		}
		time.Sleep(time.Minute * 10)
	}
}

func fetchGeoIP(servers []string, isV6 bool) geoIP {
	var ip geoIP
	var resp *http.Response
	var err error
	for i := 0; i < len(servers); i++ {
		if isV6 {
			resp, err = httpClientV6.Get(servers[i])
		} else {
			resp, err = httpClientV4.Get(servers[i])
		}
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			resp.Body.Close()
			err = json.Unmarshal(body, &ip)
			if err != nil {
				continue
			}
			return ip
		}
	}
	return ip
}

func newHTTPClient(httpTimeout, dialTimeout, keepAliveTimeout time.Duration, ipv6 bool) *http.Client {
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

	c := new(dns.Client)
	r, _, err := c.Exchange(m, net.JoinHostPort("1.1.1.1", "53"))
	if err != nil {
		return "", err
	}

	var ipv4Resolved, ipv6Resolved bool
	for _, ans := range r.Answer {
		if ipv6 {
			if aaaa, ok := ans.(*dns.AAAA); ok {
				url[0] = aaaa.AAAA.String()
				ipv6Resolved = true
			}
		} else {
			if a, ok := ans.(*dns.A); ok {
				url[0] = a.A.String()
				ipv4Resolved = true
			}
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
