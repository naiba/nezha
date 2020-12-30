package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	msg         = "msg"
	reqTypeForm = "application/x-www-form-urlencoded"
	reqTypeJSON = "application/json"
)

type testSt struct {
	url        string
	body       string
	reqType    int
	reqMethod  int
	expectURL  string
	expectBody string
	expectType string
}

func execCase(t *testing.T, item testSt) {
	n := Notification{
		URL:           item.url,
		RequestMethod: item.reqMethod,
		RequestType:   item.reqType,
		RequestBody:   item.body,
	}
	assert.Equal(t, item.expectURL, n.reqURL(msg))
	reqBody, err := n.reqBody(msg)
	assert.Nil(t, err)
	assert.Equal(t, item.expectBody, reqBody)
	assert.Equal(t, item.expectType, n.reqContentType())
}

func TestNotification(t *testing.T) {
	cases := []testSt{
		{
			url:        "https://example.com",
			body:       `{"asd":"dsa"}`,
			reqMethod:  NotificationRequestMethodGET,
			expectURL:  "https://example.com",
			expectBody: "",
			expectType: "",
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"asd":"dsa"}`,
			reqMethod:  NotificationRequestMethodGET,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: "",
			expectType: "",
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"asd":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeForm,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: "asd=" + msg,
			expectType: reqTypeForm,
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"#NEZHA#":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeForm,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: "%23NEZHA%23=" + msg,
			expectType: reqTypeForm,
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"asd":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeJSON,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: `{"asd":"msg"}`,
			expectType: reqTypeJSON,
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"#NEZHA#":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeJSON,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: `{"msg":"msg"}`,
			expectType: reqTypeJSON,
		},
	}

	for _, c := range cases {
		execCase(t, c)
	}
}
