package ddns

import (
	"bytes"
	"log"
	"net/http"
	"strings"
)

type ProviderWebHook struct {
	URL           string
	RequestMethod string
	RequestBody   string
	RequestHeader string
}

func (provider ProviderWebHook) UpdateDomain(domainConfig *DomainConfig) bool {
	if domainConfig == nil {
		return false
	}

	if domainConfig.FullDomain == "" {
		log.Println("NEZHA>> Failed to update an empty domain")
		return false
	}
	updated := false
	client := &http.Client{}
	if domainConfig.EnableIPv4 && domainConfig.Ipv4Addr != "" {
		url := provider.FormatWebhookString(provider.URL, domainConfig, "ipv4")
		body := provider.FormatWebhookString(provider.RequestBody, domainConfig, "ipv4")
		header := provider.FormatWebhookString(provider.RequestHeader, domainConfig, "ipv4")
		headers := strings.Split(header, "\n")
		req, err := http.NewRequest(provider.RequestMethod, url, bytes.NewBufferString(body))
		if err == nil && req != nil {
			SetStringHeadersToRequest(req, headers)
			if _, err := client.Do(req); err != nil {
				log.Printf("NEZHA>> Failed to update a domain: %s. Cause by: %s\n", domainConfig.FullDomain, err.Error())
			} else {
				updated = true
			}
		}
	}
	if domainConfig.EnableIpv6 && domainConfig.Ipv6Addr != "" {
		url := provider.FormatWebhookString(provider.URL, domainConfig, "ipv6")
		body := provider.FormatWebhookString(provider.RequestBody, domainConfig, "ipv6")
		header := provider.FormatWebhookString(provider.RequestHeader, domainConfig, "ipv6")
		headers := strings.Split(header, "\n")
		req, err := http.NewRequest(provider.RequestMethod, url, bytes.NewBufferString(body))
		if err == nil && req != nil {
			SetStringHeadersToRequest(req, headers)
			if _, err := client.Do(req); err != nil {
				log.Printf("NEZHA>> Failed to update a domain: %s. Cause by: %s\n", domainConfig.FullDomain, err.Error())
			} else {
				updated = true
			}
		}
	}
	return updated
}
