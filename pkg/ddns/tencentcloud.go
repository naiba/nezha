package ddns

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/naiba/nezha/pkg/utils"
)

const te = "https://dnspod.tencentcloudapi.com"

type ProviderTencentCloud struct {
	isIpv4       bool
	domainConfig *DomainConfig
	recordID     uint64
	recordType   string
	secretID     string
	secretKey    string
	errCode      string
	ipAddr       string
}

type tcReq struct {
	RecordType string `json:"RecordType"`
	Domain     string `json:"Domain"`
	RecordLine string `json:"RecordLine"`
	Subdomain  string `json:"Subdomain,omitempty"`
	SubDomain  string `json:"SubDomain,omitempty"` // As is
	Value      string `json:"Value,omitempty"`
	TTL        uint32 `json:"TTL,omitempty"`
	RecordId   uint64 `json:"RecordId,omitempty"`
}

func NewProviderTencentCloud(id, key string) *ProviderTencentCloud {
	return &ProviderTencentCloud{
		secretID:  id,
		secretKey: key,
	}
}

func (provider *ProviderTencentCloud) UpdateDomain(domainConfig *DomainConfig) error {
	if domainConfig == nil {
		return fmt.Errorf("获取 DDNS 配置失败")
	}
	provider.domainConfig = domainConfig

	// 当IPv4和IPv6同时成功才算作成功
	var err error
	if provider.domainConfig.EnableIPv4 {
		provider.isIpv4 = true
		provider.recordType = getRecordString(provider.isIpv4)
		provider.ipAddr = provider.domainConfig.Ipv4Addr
		if err = provider.addDomainRecord(); err != nil {
			return err
		}
	}

	if provider.domainConfig.EnableIpv6 {
		provider.isIpv4 = false
		provider.recordType = getRecordString(provider.isIpv4)
		provider.ipAddr = provider.domainConfig.Ipv6Addr
		if err = provider.addDomainRecord(); err != nil {
			return err
		}
	}

	return err
}

func (provider *ProviderTencentCloud) addDomainRecord() error {
	err := provider.findDNSRecord()
	if err != nil {
		return fmt.Errorf("查找 DNS 记录时出错: %s", err)
	}

	if provider.errCode == "ResourceNotFound.NoDataOfRecord" { // 没有找到 DNS 记录
		return provider.createDNSRecord()
	} else if provider.errCode != "" {
		return fmt.Errorf("查询 DNS 记录时出错，错误代码为: %s", provider.errCode)
	}

	// 默认情况下更新 DNS 记录
	return provider.updateDNSRecord()
}

func (provider *ProviderTencentCloud) findDNSRecord() error {
	prefix, realDomain := splitDomain(provider.domainConfig.FullDomain)
	data := &tcReq{
		RecordType: provider.recordType,
		Domain:     realDomain,
		RecordLine: "默认",
		Subdomain:  prefix,
	}

	jsonData, _ := utils.Json.Marshal(data)
	body, err := provider.sendRequest("DescribeRecordList", jsonData)
	if err != nil {
		return err
	}

	result, err := utils.GjsonGet(body, "Response.RecordList.0.RecordId")
	if err != nil {
		if errors.Is(err, utils.ErrGjsonNotFound) {
			if errCode, err := utils.GjsonGet(body, "Response.Error.Code"); err == nil {
				provider.errCode = errCode.String()
				return nil
			}
		}
		return err
	}

	provider.recordID = result.Uint()
	return nil
}

func (provider *ProviderTencentCloud) createDNSRecord() error {
	prefix, realDomain := splitDomain(provider.domainConfig.FullDomain)
	data := &tcReq{
		RecordType: provider.recordType,
		RecordLine: "默认",
		Domain:     realDomain,
		SubDomain:  prefix,
		Value:      provider.ipAddr,
		TTL:        600,
	}

	jsonData, _ := utils.Json.Marshal(data)
	_, err := provider.sendRequest("CreateRecord", jsonData)
	return err
}

func (provider *ProviderTencentCloud) updateDNSRecord() error {
	prefix, realDomain := splitDomain(provider.domainConfig.FullDomain)
	data := &tcReq{
		RecordType: provider.recordType,
		RecordLine: "默认",
		Domain:     realDomain,
		SubDomain:  prefix,
		Value:      provider.ipAddr,
		TTL:        600,
		RecordId:   provider.recordID,
	}

	jsonData, _ := utils.Json.Marshal(data)
	_, err := provider.sendRequest("ModifyRecord", jsonData)
	return err
}

// 以下为辅助方法，如发送 HTTP 请求等
func (provider *ProviderTencentCloud) sendRequest(action string, data []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", te, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-Version", "2021-03-23")

	provider.signRequest(provider.secretID, provider.secretKey, req, action, string(data))
	resp, err := utils.HttpClient.Do(req)
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

func (provider *ProviderTencentCloud) sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func (provider *ProviderTencentCloud) hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}

func (provider *ProviderTencentCloud) WriteString(strs ...string) string {
	var b strings.Builder
	for _, str := range strs {
		b.WriteString(str)
	}

	return b.String()
}

func (provider *ProviderTencentCloud) signRequest(secretId string, secretKey string, r *http.Request, action string, payload string) {
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
