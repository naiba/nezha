package ddns

import "golang.org/x/net/publicsuffix"

type DomainConfig struct {
	EnableIPv4 bool
	EnableIpv6 bool
	FullDomain string
	Ipv4Addr   string
	Ipv6Addr   string
}

type Provider interface {
	// UpdateDomain Return is updated
	UpdateDomain(*DomainConfig) error
}

func splitDomain(domain string) (prefix string, realDomain string) {
	realDomain, _ = publicsuffix.EffectiveTLDPlusOne(domain)
	prefix = domain[:len(domain)-len(realDomain)-1]
	return prefix, realDomain
}

func getRecordString(isIpv4 bool) string {
	if isIpv4 {
		return "A"
	}
	return "AAAA"
}
