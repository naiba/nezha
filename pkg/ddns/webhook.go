package ddns

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/naiba/nezha/pkg/utils"
)

type ProviderWebHook struct {
	url           string
	requestMethod string
	requestBody   string
	requestHeader string
	domainConfig  *DomainConfig
}

func NewProviderWebHook(s, rm, rb, rh string) *ProviderWebHook {
	return &ProviderWebHook{
		url:           s,
		requestMethod: rm,
		requestBody:   rb,
		requestHeader: rh,
	}
}

func (provider *ProviderWebHook) UpdateDomain(domainConfig *DomainConfig) error {
	if domainConfig == nil {
		return fmt.Errorf("获取 DDNS 配置失败")
	}
	provider.domainConfig = domainConfig

	if provider.domainConfig.FullDomain == "" {
		return fmt.Errorf("failed to update an empty domain")
	}

	if provider.domainConfig.EnableIPv4 && provider.domainConfig.Ipv4Addr != "" {
		req, err := provider.prepareRequest(true)
		if err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.domainConfig.FullDomain, err)
		}
		if _, err := utils.HttpClient.Do(req); err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.domainConfig.FullDomain, err)
		}
	}

	if provider.domainConfig.EnableIpv6 && provider.domainConfig.Ipv6Addr != "" {
		req, err := provider.prepareRequest(false)
		if err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.domainConfig.FullDomain, err)
		}
		if _, err := utils.HttpClient.Do(req); err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.domainConfig.FullDomain, err)
		}
	}
	return nil
}

func (provider *ProviderWebHook) prepareRequest(isIPv4 bool) (*http.Request, error) {
	u, err := url.Parse(provider.url)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url: %v", err)
	}

	// Only handle queries here
	q := u.Query()
	for p, vals := range q {
		for n, v := range vals {
			vals[n] = provider.formatWebhookString(v, isIPv4)
		}
		q[p] = vals
	}

	u.RawQuery = q.Encode()
	body := provider.formatWebhookString(provider.requestBody, isIPv4)
	header := provider.formatWebhookString(provider.requestHeader, isIPv4)
	headers := strings.Split(header, "\n")

	req, err := http.NewRequest(provider.requestMethod, u.String(), bytes.NewBufferString(body))
	if err != nil {
		return nil, fmt.Errorf("failed creating new request: %v", err)
	}

	utils.SetStringHeadersToRequest(req, headers)
	return req, nil
}

func (provider *ProviderWebHook) formatWebhookString(s string, isIPv4 bool) string {
	var ipAddr, ipType string
	if isIPv4 {
		ipAddr = provider.domainConfig.Ipv4Addr
		ipType = "ipv4"
	} else {
		ipAddr = provider.domainConfig.Ipv6Addr
		ipType = "ipv6"
	}

	r := strings.NewReplacer(
		"{ip}", ipAddr,
		"{domain}", provider.domainConfig.FullDomain,
		"{type}", ipType,
		"\r", "",
	)

	result := r.Replace(strings.TrimSpace(s))
	return result
}
