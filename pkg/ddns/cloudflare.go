package ddns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ProviderCloudflare struct {
	Secret string
}

func (provider ProviderCloudflare) UpdateDomain(domainConfig *DomainConfig) bool {
	if domainConfig == nil {
		return false
	}

	zoneID, err := provider.getZoneID(domainConfig.FullDomain)
	if err != nil {
		log.Printf("无法获取 zone ID: %s\n", err)
		return false
	}

	// 当IPv4和IPv6同时成功才算作成功
	var resultV4 = true
	var resultV6 = true
	if domainConfig.EnableIPv4 {
		if !provider.addDomainRecord(zoneID, domainConfig, true) {
			resultV4 = false
		}
	}

	if domainConfig.EnableIpv6 {
		if !provider.addDomainRecord(zoneID, domainConfig, false) {
			resultV6 = false
		}
	}

	return resultV4 && resultV6
}

func (provider ProviderCloudflare) addDomainRecord(zoneID string, domainConfig *DomainConfig, isIpv4 bool) bool {
	record, err := provider.findDNSRecord(zoneID, domainConfig.FullDomain, isIpv4)
	if err != nil {
		log.Printf("查找 DNS 记录时出错: %s\n", err)
		return false
	}

	if record == nil {
		// 添加 DNS 记录
		return provider.createDNSRecord(zoneID, domainConfig, isIpv4)
	} else {
		// 更新 DNS 记录
		return provider.updateDNSRecord(zoneID, record["id"].(string), domainConfig, isIpv4)
	}
}

func (provider ProviderCloudflare) getZoneID(domain string) (string, error) {
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

func (provider ProviderCloudflare) findDNSRecord(zoneID string, domain string, isIPv4 bool) (map[string]interface{}, error) {
	var ipType = "A"
	if !isIPv4 {
		ipType = "AAAA"
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s", zoneID, ipType, domain)
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

func (provider ProviderCloudflare) createDNSRecord(zoneID string, domainConfig *DomainConfig, isIPv4 bool) bool {
	var ipType = "A"
	var ipAddr = domainConfig.Ipv4Addr
	if !isIPv4 {
		ipType = "AAAA"
		ipAddr = domainConfig.Ipv6Addr
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	data := map[string]interface{}{
		"type":    ipType,
		"name":    domainConfig.FullDomain,
		"content": ipAddr,
		"ttl":     60,
		"proxied": false,
	}
	jsonData, _ := json.Marshal(data)
	_, err := provider.sendRequest("POST", url, jsonData)
	return err == nil
}

func (provider ProviderCloudflare) updateDNSRecord(zoneID string, recordID string, domainConfig *DomainConfig, isIPv4 bool) bool {
	var ipType = "A"
	var ipAddr = domainConfig.Ipv4Addr
	if !isIPv4 {
		ipType = "AAAA"
		ipAddr = domainConfig.Ipv6Addr
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	data := map[string]interface{}{
		"type":    ipType,
		"name":    domainConfig.FullDomain,
		"content": ipAddr,
		"ttl":     60,
		"proxied": false,
	}
	jsonData, _ := json.Marshal(data)
	_, err := provider.sendRequest("PATCH", url, jsonData)
	return err == nil
}

// 以下为辅助方法，如发送 HTTP 请求等
func (provider ProviderCloudflare) sendRequest(method string, url string, data []byte) ([]byte, error) {
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
