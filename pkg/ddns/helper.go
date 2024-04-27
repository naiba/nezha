package ddns

import (
	"golang.org/x/net/publicsuffix"
	"net/http"
	"strings"
)

func (provider ProviderWebHook) FormatWebhookString(s string, config *DomainConfig, ipType string) string {
	if config == nil {
		return s
	}

	result := strings.TrimSpace(s)
	result = strings.Replace(s, "{ip}", config.Ipv4Addr, -1)
	result = strings.Replace(result, "{domain}", config.FullDomain, -1)
	result = strings.Replace(result, "{type}", ipType, -1)
	// remove \r
	result = strings.Replace(result, "\r", "", -1)
	return result
}

func SetStringHeadersToRequest(req *http.Request, headers []string) {
	if req == nil {
		return
	}
	for _, element := range headers {
		kv := strings.SplitN(element, ":", 2)
		if len(kv) == 2 {
			req.Header.Add(kv[0], kv[1])
		}
	}
}

// SplitDomain 分割域名为前缀和一级域名
func SplitDomain(domain string) (prefix string, realDomain string) {
	realDomain, _ = publicsuffix.EffectiveTLDPlusOne(domain)
	prefix = domain[:len(domain)-len(realDomain)-1]
	return prefix, realDomain
}
