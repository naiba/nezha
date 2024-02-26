package ddns

type DomainConfig struct {
	EnableIPv4 bool
	EnableIpv6 bool
	FullDomain string
	Ipv4Addr   string
	Ipv6Addr   string
}

type Provider interface {
	// UpdateDomain Return is updated
	UpdateDomain(domainConfig *DomainConfig) bool
}
