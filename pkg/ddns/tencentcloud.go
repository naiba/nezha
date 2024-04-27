package ddns

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	url = "https://dnspod.tencentcloudapi.com"
)

type ProviderTencentCloud struct {
	SecretID  string
	SecretKey string
}

func (provider ProviderTencentCloud) UpdateDomain(domainConfig *DomainConfig) bool {
	if domainConfig == nil {
		return false
	}

	// 当IPv4和IPv6同时成功才算作成功
	var resultV4 = true
	var resultV6 = true
	if domainConfig.EnableIPv4 {
		if !provider.addDomainRecord(domainConfig, true) {
			resultV4 = false
		}
	}

	if domainConfig.EnableIpv6 {
		if !provider.addDomainRecord(domainConfig, false) {
			resultV6 = false
		}
	}

	return resultV4 && resultV6
}

func (provider ProviderTencentCloud) addDomainRecord(domainConfig *DomainConfig, isIpv4 bool) bool {
	record, err := provider.findDNSRecord(domainConfig.FullDomain, isIpv4)
	if err != nil {
		log.Printf("查找 DNS 记录时出错: %s\n", err)
		return false
	}

	if errResponse, ok := record["Error"].(map[string]interface{}); ok {
		if errCode, ok := errResponse["Code"].(string); ok && errCode == "ResourceNotFound.NoDataOfRecord" { // 没有找到 DNS 记录
			// 添加 DNS 记录
			return provider.createDNSRecord(domainConfig.FullDomain, domainConfig, isIpv4)
		} else {
			log.Printf("查询 DNS 记录时出错，错误代码为: %s\n", errCode)
		}
	}

	// 默认情况下更新 DNS 记录
	return provider.updateDNSRecord(domainConfig.FullDomain, record["RecordList"].([]interface{})[0].(map[string]interface{})["RecordId"].(float64), domainConfig, isIpv4)
}

func (provider ProviderTencentCloud) findDNSRecord(domain string, isIPv4 bool) (map[string]interface{}, error) {
	var ipType = "A"
	if !isIPv4 {
		ipType = "AAAA"
	}
	_, realDomain := SplitDomain(domain)
	prefix, _ := SplitDomain(domain)
	data := map[string]interface{}{
		"RecordType": ipType,
		"Domain":     realDomain,
		"RecordLine": "默认",
		"Subdomain":  prefix,
	}
	jsonData, _ := json.Marshal(data)
	body, err := provider.sendRequest("DescribeRecordList", jsonData)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	result := res["Response"].(map[string]interface{})
	return result, nil
}

func (provider ProviderTencentCloud) createDNSRecord(domain string, domainConfig *DomainConfig, isIPv4 bool) bool {
	var ipType = "A"
	var ipAddr = domainConfig.Ipv4Addr
	if !isIPv4 {
		ipType = "AAAA"
		ipAddr = domainConfig.Ipv6Addr
	}
	_, realDomain := SplitDomain(domain)
	prefix, _ := SplitDomain(domain)
	data := map[string]interface{}{
		"RecordType": ipType,
		"RecordLine": "默认",
		"Domain":     realDomain,
		"SubDomain":  prefix,
		"Value":      ipAddr,
		"TTL":        600,
	}
	jsonData, _ := json.Marshal(data)
	_, err := provider.sendRequest("CreateRecord", jsonData)
	return err == nil
}

func (provider ProviderTencentCloud) updateDNSRecord(domain string, recordID float64, domainConfig *DomainConfig, isIPv4 bool) bool {
	var ipType = "A"
	var ipAddr = domainConfig.Ipv4Addr
	if !isIPv4 {
		ipType = "AAAA"
		ipAddr = domainConfig.Ipv6Addr
	}
	_, realDomain := SplitDomain(domain)
	prefix, _ := SplitDomain(domain)
	data := map[string]interface{}{
		"RecordType": ipType,
		"RecordLine": "默认",
		"Domain":     realDomain,
		"SubDomain":  prefix,
		"Value":      ipAddr,
		"TTL":        600,
		"RecordId":   recordID,
	}
	jsonData, _ := json.Marshal(data)
	_, err := provider.sendRequest("ModifyRecord", jsonData)
	return err == nil
}

// 以下为辅助方法，如发送 HTTP 请求等
func (provider ProviderTencentCloud) sendRequest(action string, data []byte) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-Version", "2021-03-23")

	provider.signRequest(provider.SecretID, provider.SecretKey, req, action, string(data))
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

// https://github.com/jeessy2/ddns-go/blob/master/util/tencent_cloud_signer.go

func (provider ProviderTencentCloud) sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func (provider ProviderTencentCloud) hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}

func (provider ProviderTencentCloud) WriteString(strs ...string) string {
	var b strings.Builder
	for _, str := range strs {
		b.WriteString(str)
	}

	return b.String()
}

func (provider ProviderTencentCloud) signRequest(secretId string, secretKey string, r *http.Request, action string, payload string) {
	algorithm := "TC3-HMAC-SHA256"
	service := "dnspod"
	host := provider.WriteString(service, ".tencentcloudapi.com")
	timestamp := time.Now().Unix()
	timestampStr := strconv.FormatInt(timestamp, 10)

	// 步骤 1：拼接规范请求串
	canonicalHeaders := provider.WriteString("content-type:application/json\nhost:", host, "\nx-tc-action:", strings.ToLower(action), "\n")
	signedHeaders := "content-type;host;x-tc-action"
	hashedRequestPayload := provider.sha256hex(payload)
	canonicalRequest := provider.WriteString("POST\n/\n\n", canonicalHeaders, "\n", signedHeaders, "\n", hashedRequestPayload)

	// 步骤 2：拼接待签名字符串
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := provider.WriteString(date, "/", service, "/tc3_request")
	hashedCanonicalRequest := provider.sha256hex(canonicalRequest)
	string2sign := provider.WriteString(algorithm, "\n", timestampStr, "\n", credentialScope, "\n", hashedCanonicalRequest)

	// 步骤 3：计算签名
	secretDate := provider.hmacsha256(date, provider.WriteString("TC3", secretKey))
	secretService := provider.hmacsha256(service, secretDate)
	secretSigning := provider.hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(provider.hmacsha256(string2sign, secretSigning)))

	// 步骤 4：拼接 Authorization
	authorization := provider.WriteString(algorithm, " Credential=", secretId, "/", credentialScope, ", SignedHeaders=", signedHeaders, ", Signature=", signature)

	r.Header.Add("Authorization", authorization)
	r.Header.Set("Host", host)
	r.Header.Set("X-TC-Action", action)
	r.Header.Add("X-TC-Timestamp", timestampStr)
}
