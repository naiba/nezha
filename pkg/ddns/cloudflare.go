package ddns

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/naiba/nezha/pkg/utils"
)

const baseEndpoint = "https://api.cloudflare.com/client/v4/zones"

type ProviderCloudflare struct {
	secret       string
	zoneId       string
	recordId     string
	domainConfig *DomainConfig
}

type cfReq struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     uint32 `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type cfResp struct {
	Result []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
}

func NewProviderCloudflare(s string) *ProviderCloudflare {
	return &ProviderCloudflare{
		secret: s,
	}
}

func (provider *ProviderCloudflare) UpdateDomain(domainConfig *DomainConfig) error {
	if domainConfig == nil {
		return fmt.Errorf("获取 DDNS 配置失败")
	}
	provider.domainConfig = domainConfig

	err := provider.getZoneID()
	if err != nil {
		return fmt.Errorf("无法获取 zone ID: %s", err)
	}

	// 当IPv4和IPv6同时成功才算作成功
	if provider.domainConfig.EnableIPv4 {
		if err = provider.addDomainRecord(true); err != nil {
			return err
		}
	}

	if provider.domainConfig.EnableIpv6 {
		if err = provider.addDomainRecord(false); err != nil {
			return err
		}
	}

	return nil
}

func (provider *ProviderCloudflare) addDomainRecord(isIpv4 bool) error {
	err := provider.findDNSRecord(isIpv4)
	if err != nil {
		return fmt.Errorf("查找 DNS 记录时出错: %s", err)
	}

	if provider.recordId == "" {
		// 添加 DNS 记录
		return provider.createDNSRecord(isIpv4)
	} else {
		// 更新 DNS 记录
		return provider.updateDNSRecord(isIpv4)
	}
}

func (provider *ProviderCloudflare) getZoneID() error {
	_, realDomain := splitDomain(provider.domainConfig.FullDomain)
	zu, _ := url.Parse(baseEndpoint)

	q := zu.Query()
	q.Set("name", realDomain)
	zu.RawQuery = q.Encode()

	body, err := provider.sendRequest("GET", zu.String(), nil)
	if err != nil {
		return err
	}

	res := &cfResp{}
	err = utils.Json.Unmarshal(body, res)
	if err != nil {
		return err
	}

	result := res.Result
	if len(result) > 0 {
		provider.zoneId = result[0].ID
		return nil
	}

	return fmt.Errorf("找不到 Zone ID")
}

func (provider *ProviderCloudflare) findDNSRecord(isIPv4 bool) error {
	var ipType string
	if isIPv4 {
		ipType = "A"
	} else {
		ipType = "AAAA"
	}

	de, _ := url.JoinPath(baseEndpoint, provider.zoneId, "dns_records")
	du, _ := url.Parse(de)

	q := du.Query()
	q.Set("name", provider.domainConfig.FullDomain)
	q.Set("type", ipType)
	du.RawQuery = q.Encode()

	body, err := provider.sendRequest("GET", du.String(), nil)
	if err != nil {
		return err
	}

	res := &cfResp{}
	err = utils.Json.Unmarshal(body, res)
	if err != nil {
		return err
	}

	result := res.Result
	if len(result) > 0 {
		provider.recordId = result[0].ID
		return nil
	}

	return nil
}

func (provider *ProviderCloudflare) createDNSRecord(isIPv4 bool) error {
	var ipType, ipAddr string
	if isIPv4 {
		ipType = "A"
		ipAddr = provider.domainConfig.Ipv4Addr
	} else {
		ipType = "AAAA"
		ipAddr = provider.domainConfig.Ipv6Addr
	}

	de, _ := url.JoinPath(baseEndpoint, provider.zoneId, "dns_records")
	data := &cfReq{
		Name:    provider.domainConfig.FullDomain,
		Type:    ipType,
		Content: ipAddr,
		TTL:     60,
		Proxied: false,
	}

	jsonData, _ := utils.Json.Marshal(data)
	_, err := provider.sendRequest("POST", de, jsonData)
	return err
}

func (provider *ProviderCloudflare) updateDNSRecord(isIPv4 bool) error {
	var ipType, ipAddr string
	if isIPv4 {
		ipType = "A"
		ipAddr = provider.domainConfig.Ipv4Addr
	} else {
		ipType = "AAAA"
		ipAddr = provider.domainConfig.Ipv6Addr
	}

	de, _ := url.JoinPath(baseEndpoint, provider.zoneId, "dns_records", provider.recordId)
	data := &cfReq{
		Name:    provider.domainConfig.FullDomain,
		Type:    ipType,
		Content: ipAddr,
		TTL:     60,
		Proxied: false,
	}

	jsonData, _ := utils.Json.Marshal(data)
	_, err := provider.sendRequest("PATCH", de, jsonData)
	return err
}

// 以下为辅助方法，如发送 HTTP 请求等
func (provider *ProviderCloudflare) sendRequest(method string, url string, data []byte) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", provider.secret))
	req.Header.Add("Content-Type", "application/json")

	resp, err := utils.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("NEZHA>> 无法关闭HTTP响应体流: %s", err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
