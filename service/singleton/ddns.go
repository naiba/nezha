package singleton

import (
	"errors"
	"log"
)

type DDNSDomainConfig struct {
	EnableIPv4 bool
	EnableIpv6 bool
	FullDomain string
	Ipv4Addr   string
	Ipv6Addr   string
}

type DDNSProvider interface {
	// UpdateDomain Return is updated
	UpdateDomain(domainConfig *DDNSDomainConfig) bool
}

type DDNSProviderWebHook struct {
	URL           string
	RequestBody   string
	RequestHeader string
}

func (provider DDNSProviderWebHook) UpdateDomain(domainConfig *DDNSDomainConfig) bool {
	if domainConfig.FullDomain == "" {
		log.Println("NEZHA>> Failed to update an empty domain")
		return false
	}
	updated := false
	if domainConfig.EnableIPv4 {

		updated = true
	}
	if domainConfig.EnableIpv6 {
		updated = true
	}
	return updated
}

type DDNSProviderDummy struct{}

func (provider DDNSProviderDummy) UpdateDomain(domainConfig *DDNSDomainConfig) bool {
	return false
}

func GetDDNSProviderFromString(provider string) (DDNSProvider, error) {
	return DDNSProviderDummy{}, errors.New("")
}
