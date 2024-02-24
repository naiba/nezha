package singleton

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type DDNSProviderCloudflare struct {
	Secret string
}

func (provider DDNSProviderCloudflare) UpdateDomain(domainConfig *DDNSDomainConfig) bool {
	if domainConfig == nil {
		return false
	}

	zoneID, err := provider.getZoneID(domainConfig.FullDomain)
	if err != nil {
		log.Printf("无法获取 zone ID: %s\n", err)
		return false
	}

	record, err := provider.findDNSRecord(zoneID, domainConfig.FullDomain)
	if err != nil {
		log.Printf("查找 DNS 记录时出错: %s\n", err)
		return false
	}

	if record == nil {
		// 添加 DNS 记录
		return provider.createDNSRecord(zoneID, domainConfig)
	} else {
		// 更新 DNS 记录
		return provider.updateDNSRecord(zoneID, record["id"].(string), domainConfig)
	}
}

func (provider DDNSProviderCloudflare) getZoneID(domain string) (string, error) {
	_, realDomain := SplitDomain(domain)
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", realDomain)
	body, err := provider.sendRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	var res map[string]interface{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return "", err
	}

	result := res["result"].([]interface{})
	if len(result) > 0 {
		zoneID := result[0].(map[string]interface{})["id"].(string)
		return zoneID, nil
	}

	return "", fmt.Errorf("找不到 Zone ID")
}

func (provider DDNSProviderCloudflare) findDNSRecord(zoneID string, domain string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s", zoneID, domain)
	body, err := provider.sendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	result := res["result"].([]interface{})
	if len(result) > 0 {
		return result[0].(map[string]interface{}), nil
	}

	return nil, nil // 没有找到 DNS 记录
}

func (provider DDNSProviderCloudflare) createDNSRecord(zoneID string, domainConfig *DDNSDomainConfig) bool {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	data := map[string]interface{}{
		"type":    "A",
		"name":    domainConfig.FullDomain,
		"content": domainConfig.Ipv4Addr,
		"ttl":     3600,
		"proxied": false,
	}
	jsonData, _ := json.Marshal(data)
	_, err := provider.sendRequest("POST", url, jsonData)
	return err == nil
}

func (provider DDNSProviderCloudflare) updateDNSRecord(zoneID string, recordID string, domainConfig *DDNSDomainConfig) bool {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	data := map[string]interface{}{
		"type":    "A",
		"name":    domainConfig.FullDomain,
		"content": domainConfig.Ipv4Addr,
		"ttl":     3600,
		"proxied": false,
	}
	jsonData, _ := json.Marshal(data)
	_, err := provider.sendRequest("PATCH", url, jsonData)
	return err == nil
}

// 以下为辅助方法，如发送 HTTP 请求等
func (provider DDNSProviderCloudflare) sendRequest(method string, url string, data []byte) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", provider.Secret))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("NEZHA>> 无法关闭HTTP响应体流: %s\n", err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (provider DDNSProviderWebHook) FormatWebhookString(s string, config *DDNSDomainConfig, ipType string) string {
	if config == nil {
		return s
	}

	result := strings.TrimSpace(s)
	result = strings.Replace(s, "{ip}", config.Ipv4Addr, -1)
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

func RetryableUpdateDomain(provider DDNSProvider, config *DDNSDomainConfig, maxRetries int) bool {
	if nil == config {
		return false
	}
	for retries := 0; retries < maxRetries; retries++ {
		log.Printf("NEZHA>> 正在尝试更新域名(%s)DDNS(%d/%d)\n", config.FullDomain, retries+1, maxRetries)
		if provider.UpdateDomain(config) {
			log.Printf("NEZHA>> 尝试更新域名(%s)DDNS成功\n", config.FullDomain)
			return true
		}
	}
	log.Printf("NEZHA>> 尝试更新域名(%s)DDNS失败\n", config.FullDomain)
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
	case "cloudflare":
		return DDNSProviderCloudflare{
			Secret: Conf.DDNS.AccessSecret,
		}, nil
	}
	return DDNSProviderDummy{}, errors.New(fmt.Sprintf("无法找到配置的DDNS提供者%s", Conf.DDNS.Provider))
}
