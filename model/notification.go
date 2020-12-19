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

type NotificatonSender struct {
	Rule   *Rule
	Server *Server
	State  *State
}

type Notification struct {
	Common
	Name          string
	URL           string
	RequestMethod int
	RequestType   int
	RequestBody   string `gorm:"type:longtext" `
	VerifySSL     *bool
}

func (n *Notification) Send(sender *NotificatonSender, message string) {
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

	if err == nil {
		if n.RequestMethod == NotificationRequestMethodGET {
			for k, v := range data {
				reqURL.Query().Set(k, replaceParamsInString(v, sender))
			}
			client.Get(reqURL.String())
		} else {
			if n.RequestType == NotificationRequestTypeForm {
				params := url.Values{}
				for k, v := range data {
					params.Add(k, replaceParamsInString(v, sender))
				}
				client.PostForm(reqURL.String(), params)
			} else {
				jsonValue := replaceParamsInJSON(n.RequestBody, sender)
				if err == nil {
					client.Post(reqURL.String(), "application/json", strings.NewReader(jsonValue))
				}
			}
		}
	}
}

func replaceParamsInString(str string, sender *NotificatonSender) string {
	str = strings.ReplaceAll(str, "#CPU#", fmt.Sprintf("%2f%%", sender.State.CPU))
	return str
}

func replaceParamsInJSON(str string, sender *NotificatonSender) string {
	str = strings.ReplaceAll(str, "#CPU#", fmt.Sprintf("%2f%%", sender.State.CPU))
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
