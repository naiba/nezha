package ddns

import (
	"fmt"
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
func SplitDomain(domain string) (prefix string, topLevelDomain string) {
	// 带有二级TLD的一些常见例子，需要特别处理
	secondLevelTLDs := map[string]bool{
		".co.uk": true, ".com.cn": true, ".gov.cn": true, ".net.cn": true, ".org.cn": true,
	}

	// 分割域名为"."的各部分
	parts := strings.Split(domain, ".")

	// 处理特殊情况，例如 ".co.uk"
	for i := len(parts) - 2; i > 0; i-- {
		potentialTLD := fmt.Sprintf(".%s.%s", parts[i], parts[i+1])
		if secondLevelTLDs[potentialTLD] {
			if i > 1 {
				return strings.Join(parts[:i-1], "."), strings.Join(parts[i-1:], ".")
			}
			return "", domain // 当域名仅为二级TLD时，无前缀
		}
	}

	// 常规处理，查找最后一个"."前的所有内容作为前缀
	if len(parts) > 2 {
		return strings.Join(parts[:len(parts)-2], "."), strings.Join(parts[len(parts)-2:], ".")
	}
	return "", domain // 当域名不包含子域名时，无前缀
}
