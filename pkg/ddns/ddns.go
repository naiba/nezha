package ddns

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
)

var dnsTimeOut = 10 * time.Second

type IP struct {
	Ipv4Addr string
	Ipv6Addr string
}

type Provider interface {
	UpdateDomain()
}

type provider struct {
	IsIpv4      bool
	DDNSProfile *model.DDNSProfile
	IPAddrs     *IP
	IPAddr      string
	RecordType  string
	Domain      string
}

func splitDomain(domain string) (prefix string, zone string) {
	zone, _ = publicsuffix.EffectiveTLDPlusOne(domain)
	prefix = domain[:len(domain)-len(zone)-1]
	return
}

func splitDomainSOA(domain string) (prefix string, zone string, err error) {
	c := &dns.Client{Timeout: dnsTimeOut}

	domain += "."
	indexes := dns.Split(domain)

	var r *dns.Msg
	for _, idx := range indexes {
		m := new(dns.Msg)
		m.SetQuestion(domain[idx:], dns.TypeSOA)

		for _, server := range utils.DNSServers {
			r, _, err = c.Exchange(m, server)
			if err != nil {
				return
			}
			if len(r.Answer) > 0 {
				if soa, ok := r.Answer[0].(*dns.SOA); ok {
					zone = strings.TrimSuffix(soa.Hdr.Name, ".")
					prefix = domain[:len(domain)-len(zone)-2]
					return
				}
			}
		}
	}

	return "", "", fmt.Errorf("SOA record not found for domain: %s", domain)
}

func getRecordString(isIpv4 bool) string {
	if isIpv4 {
		return "A"
	}
	return "AAAA"
}
