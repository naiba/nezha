package model

import (
	"reflect"
	"testing"
)

func TestIPStringToBinary(t *testing.T) {
	cases := []struct {
		ip          string
		want        []byte
		expectError bool
	}{
		// 有效的 IPv4 地址
		{
			ip: "192.168.1.1",
			want: []byte{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 1,
			},
			expectError: false,
		},
		// 有效的 IPv6 地址
		{
			ip: "2001:db8::68",
			want: []byte{
				32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 104,
			},
			expectError: false,
		},
		// 无效的 IP 地址
		{
			ip:          "invalid_ip",
			want:        []byte{},
			expectError: true,
		},
	}

	for _, c := range cases {
		got, err := ipStringToBinary(c.ip)
		if (err != nil) != c.expectError {
			t.Errorf("IPStringToBinary(%q) error = %v, expect error = %v", c.ip, err, c.expectError)
			continue
		}
		if err == nil && !reflect.DeepEqual(got, c.want) {
			t.Errorf("IPStringToBinary(%q) = %v, want %v", c.ip, got, c.want)
		}
	}
}

func TestBinaryToIPString(t *testing.T) {
	cases := []struct {
		binary []byte
		want   string
	}{
		// IPv4 地址（IPv4 映射的 IPv6 地址格式）
		{
			binary: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 1, 1},
			want:   "192.168.1.1",
		},
		// 其他测试用例
		{
			binary: []byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 104},
			want:   "2001:db8::68",
		},
		// 全零值
		{
			binary: []byte{},
			want:   "::",
		},
		// IPv4 映射的 IPv6 地址
		{
			binary: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 127, 0, 0, 1},
			want:   "127.0.0.1",
		},
	}

	for _, c := range cases {
		got := binaryToIPString(c.binary)
		if got != c.want {
			t.Errorf("BinaryToIPString(%v) = %q, 期望 %q", c.binary, got, c.want)
		}
	}
}
