package model

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	_ = iota
	NotificationRequestTypeJSON
	NotificationRequestTypeForm
)

const (
	_ = iota
	NotificationRequestMethodGET
	NotificationRequestMethodPOST
)

type Notification struct {
	Common
	Name          string
	URL           string
	RequestMethod int
	RequestType   int
	RequestBody   string `gorm:"type:longtext" `
	VerifySSL     *bool
}

func (n *Notification) Send(message string) error {
	var verifySSL bool

	if n.VerifySSL != nil && *n.VerifySSL {
		verifySSL = true
	}

	var err error
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: verifySSL},
	}
	client := &http.Client{Transport: transCfg, Timeout: time.Minute * 10}
	var reqURL *url.URL
	reqURL, err = url.Parse(n.URL)
	var data map[string]string
	if err == nil && (n.RequestMethod == NotificationRequestMethodGET || n.RequestType == NotificationRequestTypeForm) {
		err = json.Unmarshal([]byte(n.RequestBody), &data)
	}

	var resp *http.Response

	if err == nil {
		if n.RequestMethod == NotificationRequestMethodGET {
			var queryValue = reqURL.Query()
			for k, v := range data {
				queryValue.Set(k, replaceParamsInString(v, message))
			}
			reqURL.RawQuery = queryValue.Encode()
			resp, err = client.Get(reqURL.String())
		} else {
			if n.RequestType == NotificationRequestTypeForm {
				params := url.Values{}
				for k, v := range data {
					params.Add(k, replaceParamsInString(v, message))
				}
				resp, err = client.PostForm(reqURL.String(), params)
			} else {
				jsonValue := replaceParamsInJSON(n.RequestBody, message)
				resp, err = client.Post(reqURL.String(), "application/json", strings.NewReader(jsonValue))
			}
		}
	}

	if err == nil && (resp.StatusCode < 200 || resp.StatusCode > 299) {
		err = fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	return err
}

func replaceParamsInString(str string, message string) string {
	str = strings.ReplaceAll(str, "#NEZHA#", message)
	return str
}

func replaceParamsInJSON(str string, message string) string {
	str = strings.ReplaceAll(str, "#NEZHA#", message)
	return str
}

func jsonEscape(raw interface{}) string {
	b, _ := json.Marshal(raw)
	strb := string(b)
	if strings.HasPrefix(strb, "\"") {
		return strb[1 : len(strb)-1]
	}
	return strb
}
