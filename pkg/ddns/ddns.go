package ddns

import (
	"github.com/naiba/nezha/model"
	"golang.org/x/net/publicsuffix"
)

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

func splitDomain(domain string) (prefix string, tld string) {
	tld, _ = publicsuffix.EffectiveTLDPlusOne(domain)
	prefix = domain[:len(domain)-len(tld)-1]
	return
}

func getRecordString(isIpv4 bool) string {
	if isIpv4 {
		return "A"
	}
	return "AAAA"
}
