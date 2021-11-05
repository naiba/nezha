package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	msg = "msg"
)

type testSt struct {
	url        string
	body       string
	reqType    int
	reqMethod  int
	expectURL  string
	expectBody string
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
}

func TestNotification(t *testing.T) {
	cases := []testSt{
		{
			url:        "https://example.com",
			body:       `{"asd":"dsa"}`,
			reqMethod:  NotificationRequestMethodGET,
			expectURL:  "https://example.com",
			expectBody: "",
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"asd":"dsa"}`,
			reqMethod:  NotificationRequestMethodGET,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: "",
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"asd":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeForm,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: "asd=" + msg,
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"#NEZHA#":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeForm,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: "%23NEZHA%23=" + msg,
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"asd":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeJSON,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: `{"asd":"msg"}`,
		},
		{
			url:        "https://example.com/?m=#NEZHA#",
			body:       `{"#NEZHA#":"#NEZHA#"}`,
			reqMethod:  NotificationRequestMethodPOST,
			reqType:    NotificationRequestTypeJSON,
			expectURL:  "https://example.com/?m=" + msg,
			expectBody: `{"msg":"msg"}`,
		},
	}

	for _, c := range cases {
		execCase(t, c)
	}
}
