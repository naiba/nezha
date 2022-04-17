package model

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	msg         = "msg"
	reqTypeForm = "application/x-www-form-urlencoded"
	reqTypeJSON = "application/json"
)

type testSt struct {
	url               string
	body              string
	header            string
	reqType           int
	reqMethod         int
	expectURL         string
	expectBody        string
	expectMethod      string
	expectContentType string
	expectHeader      map[string]string
}

func execCase(t *testing.T, item testSt) {
	n := Notification{
		URL:           item.url,
		RequestMethod: item.reqMethod,
		RequestType:   item.reqType,
		RequestBody:   item.body,
		RequestHeader: item.header,
	}
	server := Server{
		Common:                Common{},
		Name:                  "ServerName",
		Tag:                   "",
		Secret:                "",
		Note:                  "",
		DisplayIndex:          0,
		Host:                  nil,
		State:                 nil,
		LastActive:            time.Time{},
		TaskClose:             nil,
		TaskStream:            nil,
		PrevHourlyTransferIn:  0,
		PrevHourlyTransferOut: 0,
	}
	ns := NotificationServerBundle{
		Notification: &n,
		Server:       &server,
	}
	assert.Equal(t, item.expectURL, ns.reqURL(msg))
	reqBody, err := ns.reqBody(msg)
	assert.Nil(t, err)
	assert.Equal(t, item.expectBody, reqBody)
	reqMethod, err := n.reqMethod()
	assert.Nil(t, err)
	assert.Equal(t, item.expectMethod, reqMethod)

	req, err := http.NewRequest("", "", strings.NewReader(""))
	assert.Nil(t, err)
	n.setContentType(req)
	assert.Equal(t, item.expectContentType, req.Header.Get("Content-Type"))
	n.setRequestHeader(req)
	for k, v := range item.expectHeader {
		assert.Equal(t, v, req.Header.Get(k))
	}
}

func TestNotification(t *testing.T) {
	cases := []testSt{
		{
			url:               "https://example.com",
			body:              `{"asd":"dsa"}`,
			header:            `{"asd":"dsa"}`,
			reqMethod:         NotificationRequestMethodGET,
			expectURL:         "https://example.com",
			expectMethod:      http.MethodGet,
			expectContentType: "",
			expectHeader:      map[string]string{"asd": "dsa"},
			expectBody:        "",
		},
		{
			url:               "https://example.com/?m=#NEZHA#",
			body:              `{"asd":"dsa"}`,
			reqMethod:         NotificationRequestMethodGET,
			expectURL:         "https://example.com/?m=" + msg,
			expectMethod:      http.MethodGet,
			expectContentType: "",
			expectBody:        "",
		},
		{
			url:               "https://example.com/?m=#NEZHA#",
			body:              `{"asd":"#NEZHA#"}`,
			reqMethod:         NotificationRequestMethodPOST,
			reqType:           NotificationRequestTypeForm,
			expectURL:         "https://example.com/?m=" + msg,
			expectMethod:      http.MethodPost,
			expectContentType: reqTypeForm,
			expectBody:        "asd=" + msg,
		},
		{
			url:               "https://example.com/?m=#NEZHA#",
			body:              `{"#NEZHA#":"#NEZHA#"}`,
			reqMethod:         NotificationRequestMethodPOST,
			reqType:           NotificationRequestTypeForm,
			expectURL:         "https://example.com/?m=" + msg,
			expectMethod:      http.MethodPost,
			expectContentType: reqTypeForm,
			expectBody:        "%23NEZHA%23=" + msg,
		},
		{
			url:               "https://example.com/?m=#NEZHA#",
			body:              `{"asd":"#NEZHA#"}`,
			reqMethod:         NotificationRequestMethodPOST,
			reqType:           NotificationRequestTypeJSON,
			expectURL:         "https://example.com/?m=" + msg,
			expectMethod:      http.MethodPost,
			expectContentType: reqTypeJSON,
			expectBody:        `{"asd":"msg"}`,
		},
		{
			url:               "https://example.com/?m=#NEZHA#",
			body:              `{"#NEZHA#":"#NEZHA#"}`,
			reqMethod:         NotificationRequestMethodPOST,
			header:            `{"asd":"dsa11"}`,
			reqType:           NotificationRequestTypeJSON,
			expectURL:         "https://example.com/?m=" + msg,
			expectMethod:      http.MethodPost,
			expectContentType: reqTypeJSON,
			expectBody:        `{"msg":"msg"}`,
			expectHeader:      map[string]string{"asd": "dsa11"},
		},
		{
			url:               "https://example.com/?m=#NEZHA#",
			body:              `{"Server":"#SERVER#"}`,
			reqMethod:         NotificationRequestMethodPOST,
			header:            `{"asd":"dsa11"}`,
			reqType:           NotificationRequestTypeJSON,
			expectURL:         "https://example.com/?m=" + msg,
			expectMethod:      http.MethodPost,
			expectContentType: reqTypeJSON,
			expectBody:        `{"Server":"ServerName"}`,
			expectHeader:      map[string]string{"asd": "dsa11"},
		},
	}

	for _, c := range cases {
		execCase(t, c)
	}
}
