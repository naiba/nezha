package singleton

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	RequestMethod string
	RequestBody   string
	RequestHeader string
}

func (provider DDNSProviderWebHook) UpdateDomain(domainConfig *DDNSDomainConfig) bool {
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
			}
			updated = true
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
			}
			updated = true
		}
	}
	return updated
}

type DDNSProviderDummy struct{}

func (provider DDNSProviderDummy) UpdateDomain(domainConfig *DDNSDomainConfig) bool {
	return false
}

func GetDDNSProviderFromString(provider string) (DDNSProvider, error) {
	switch provider {
	case "webhook":
		return DDNSProviderWebHook{
			URL:           Conf.DDNS.WebhookURL,
			RequestMethod: Conf.DDNS.WebhookMethod,
			RequestBody:   Conf.DDNS.WebhookRequestBody,
			RequestHeader: Conf.DDNS.WebhookHeaders,
		}, nil
	case "dummy":
		return DDNSProviderDummy{}, nil
	}
	return DDNSProviderDummy{}, errors.New(fmt.Sprintf("无法找到配置的DDNS提供者%s", Conf.DDNS.Provider))
}

func (provider DDNSProviderWebHook) FormatWebhookString(s string, config *DDNSDomainConfig, ipType string) string {
	if config == nil {
		return s
	}

	result := strings.TrimSpace(s)
	result = strings.Replace(provider.RequestBody, "{ip}", config.Ipv4Addr, -1)
	result = strings.Replace(result, "{domain}", config.FullDomain, -1)
	result = strings.Replace(result, "{type}", ipType, -1)
	result = strings.Replace(result, "{access_id}", Conf.DDNS.AccessID, -1)
	result = strings.Replace(result, "{access_secret}", Conf.DDNS.AccessSecret, -1)
	// remove \r
	result = strings.Replace(result, "\r", "", -1)
	return result
}

func SetStringHeadersToRequest(req *http.Request, headers []string) {
	if req == nil {
		return
	}
	for _, element := range headers {
		kv := strings.SplitN(element, ":", 1)
		if len(kv) == 2 {
			req.Header.Add(kv[0], kv[1])
		}
	}
}
