package model

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/naiba/nezha/pkg/utils"
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

type NotificationServerBundle struct {
	Notification *Notification
	Server       *Server
}

type Notification struct {
	Common
	Name          string
	Tag           string // 分组名
	URL           string
	RequestMethod int
	RequestType   int
	RequestHeader string `gorm:"type:longtext" `
	RequestBody   string `gorm:"type:longtext" `
	VerifySSL     *bool
}

func (ns *NotificationServerBundle) reqURL(message string) string {
	n := ns.Notification
	return replaceParamsInString(ns.Server, n.URL, message, func(msg string) string {
		return url.QueryEscape(msg)
	})
}

func (n *Notification) reqMethod() (string, error) {
	switch n.RequestMethod {
	case NotificationRequestMethodPOST:
		return http.MethodPost, nil
	case NotificationRequestMethodGET:
		return http.MethodGet, nil
	}
	return "", errors.New("不支持的请求方式")
}

func (ns *NotificationServerBundle) reqBody(message string) (string, error) {
	n := ns.Notification
	if n.RequestMethod == NotificationRequestMethodGET || message == "" {
		return "", nil
	}
	switch n.RequestType {
	case NotificationRequestTypeJSON:
		return replaceParamsInString(ns.Server, n.RequestBody, message, func(msg string) string {
			msgBytes, _ := utils.Json.Marshal(msg)
			return string(msgBytes)[1 : len(msgBytes)-1]
		}), nil
	case NotificationRequestTypeForm:
		var data map[string]string
		if err := utils.Json.Unmarshal([]byte(n.RequestBody), &data); err != nil {
			return "", err
		}
		params := url.Values{}
		for k, v := range data {
			params.Add(k, replaceParamsInString(ns.Server, v, message, nil))
		}
		return params.Encode(), nil
	}
	return "", errors.New("不支持的请求类型")
}

func (n *Notification) setContentType(req *http.Request) {
	if n.RequestMethod == NotificationRequestMethodGET {
		return
	}
	if n.RequestType == NotificationRequestTypeForm {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
}

func (n *Notification) setRequestHeader(req *http.Request) error {
	if n.RequestHeader == "" {
		return nil
	}
	var m map[string]string
	if err := utils.Json.Unmarshal([]byte(n.RequestHeader), &m); err != nil {
		return err
	}
	for k, v := range m {
		req.Header.Set(k, v)
	}
	return nil
}

func (ns *NotificationServerBundle) Send(message string) error {
	var verifySSL bool
	n := ns.Notification
	if n.VerifySSL != nil && *n.VerifySSL {
		verifySSL = true
	}

	/* #nosec */
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: verifySSL},
	}

	client := &http.Client{Transport: transCfg, Timeout: time.Minute * 10}
	reqBody, err := ns.reqBody(message)
	if err != nil {
		return err
	}

	reqMethod, err := n.reqMethod()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(reqMethod, ns.reqURL(message), strings.NewReader(reqBody))
	if err != nil {
		return err
	}

	n.setContentType(req)

	if err := n.setRequestHeader(req); err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("%d@%s %s", resp.StatusCode, resp.Status, string(body))
	}

	return nil
}

func replaceParamsInString(s *Server, str string, message string, mod func(string) string) string {
	if mod != nil {
		str = strings.ReplaceAll(str, "#NEZHA#", mod(message))
	} else {
		str = strings.ReplaceAll(str, "#NEZHA#", message)
	}
	if s != nil {
		str = strings.ReplaceAll(str, "#SERVER#", s.Name)
	}
	return str
}
