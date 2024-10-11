package ddns

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
)

const (
	_ = iota
	methodGET
	methodPOST
	methodPATCH
	methodDELETE
	methodPUT
)

const (
	_ = iota
	requestTypeJSON
	requestTypeForm
)

var requestTypes = map[uint8]string{
	methodGET:    "GET",
	methodPOST:   "POST",
	methodPATCH:  "PATCH",
	methodDELETE: "DELETE",
	methodPUT:    "PUT",
}

type ProviderWebHook struct {
	provider
	ipType string
}

func NewProviderWebHook(profile *model.DDNSProfile, ip *IP) *ProviderWebHook {
	return &ProviderWebHook{
		provider: provider{DDNSProfile: profile, IPAddrs: ip},
	}
}

func (provider *ProviderWebHook) UpdateDomain() {
	for _, domain := range provider.DDNSProfile.Domains {
		for retries := 0; retries < int(provider.DDNSProfile.MaxRetries); retries++ {
			provider.Domain = domain
			log.Printf("NEZHA>> 正在尝试更新域名(%s)DDNS(%d/%d)", provider.Domain, retries+1, provider.DDNSProfile.MaxRetries)
			if err := provider.updateDomain(); err != nil {
				log.Printf("NEZHA>> 尝试更新域名(%s)DDNS失败: %v", provider.Domain, err)
			} else {
				log.Printf("NEZHA>> 尝试更新域名(%s)DDNS成功", provider.Domain)
				break
			}
		}
	}
}

func (provider *ProviderWebHook) updateDomain() error {
	if *provider.DDNSProfile.EnableIPv4 && provider.IPAddrs.Ipv4Addr != "" {
		provider.IsIpv4 = true
		provider.RecordType = getRecordString(provider.IsIpv4)
		provider.ipType = "ipv4"
		provider.IPAddr = provider.IPAddrs.Ipv4Addr
		req, err := provider.prepareRequest()
		if err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.Domain, err)
		}
		if _, err := utils.HttpClient.Do(req); err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.Domain, err)
		}
	}

	if *provider.DDNSProfile.EnableIPv6 && provider.IPAddrs.Ipv6Addr != "" {
		provider.IsIpv4 = false
		provider.RecordType = getRecordString(provider.IsIpv4)
		provider.ipType = "ipv6"
		provider.IPAddr = provider.IPAddrs.Ipv6Addr
		req, err := provider.prepareRequest()
		if err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.Domain, err)
		}
		if _, err := utils.HttpClient.Do(req); err != nil {
			return fmt.Errorf("failed to update a domain: %s. Cause by: %v", provider.Domain, err)
		}
	}
	return nil
}

func (provider *ProviderWebHook) prepareRequest() (*http.Request, error) {
	u, err := provider.reqUrl()
	if err != nil {
		return nil, err
	}

	body, err := provider.reqBody()
	if err != nil {
		return nil, err
	}

	headers, err := utils.GjsonParseStringMap(
		provider.formatWebhookString(provider.DDNSProfile.WebhookHeaders))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(requestTypes[provider.DDNSProfile.WebhookMethod], u.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	provider.setContentType(req)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func (provider *ProviderWebHook) setContentType(req *http.Request) {
	if provider.DDNSProfile.WebhookMethod == methodGET {
		return
	}
	if provider.DDNSProfile.WebhookRequestType == requestTypeForm {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
}

func (provider *ProviderWebHook) reqUrl() (*url.URL, error) {
	formattedUrl := strings.ReplaceAll(provider.DDNSProfile.WebhookURL, "#", "%23")

	u, err := url.Parse(formattedUrl)
	if err != nil {
		return nil, err
	}

	// Only handle queries here
	q := u.Query()
	for p, vals := range q {
		for n, v := range vals {
			vals[n] = provider.formatWebhookString(v)
		}
		q[p] = vals
	}

	u.RawQuery = q.Encode()
	return u, nil
}

func (provider *ProviderWebHook) reqBody() (string, error) {
	if provider.DDNSProfile.WebhookMethod == methodGET ||
		provider.DDNSProfile.WebhookMethod == methodDELETE {
		return "", nil
	}

	switch provider.DDNSProfile.WebhookRequestType {
	case requestTypeJSON:
		return provider.formatWebhookString(provider.DDNSProfile.WebhookRequestBody), nil
	case requestTypeForm:
		data, err := utils.GjsonParseStringMap(provider.DDNSProfile.WebhookRequestBody)
		if err != nil {
			return "", err
		}
		params := url.Values{}
		for k, v := range data {
			params.Add(k, provider.formatWebhookString(v))
		}
		return params.Encode(), nil
	default:
		return "", errors.New("request type not supported")
	}
}

func (provider *ProviderWebHook) formatWebhookString(s string) string {
	r := strings.NewReplacer(
		"#ip#", provider.IPAddr,
		"#domain#", provider.Domain,
		"#type#", provider.ipType,
		"#record#", provider.RecordType,
		"\r", "",
	)

	result := r.Replace(strings.TrimSpace(s))
	return result
}
