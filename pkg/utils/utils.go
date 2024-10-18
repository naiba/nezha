package utils

import (
	"crypto/rand"
	"math/big"
	"os"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

var (
	Json = jsoniter.ConfigCompatibleWithStandardLibrary

	DNSServers = []string{"1.1.1.1:53", "223.5.5.5:53"}
)

func IsWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

var ipv4Re = regexp.MustCompile(`(\d*\.).*(\.\d*)`)

func ipv4Desensitize(ipv4Addr string) string {
	return ipv4Re.ReplaceAllString(ipv4Addr, "$1****$2")
}

var ipv6Re = regexp.MustCompile(`(\w*:\w*:).*(:\w*:\w*)`)

func ipv6Desensitize(ipv6Addr string) string {
	return ipv6Re.ReplaceAllString(ipv6Addr, "$1****$2")
}

func IPDesensitize(ipAddr string) string {
	ipAddr = ipv4Desensitize(ipAddr)
	ipAddr = ipv6Desensitize(ipAddr)
	return ipAddr
}

// SplitIPAddr 传入/分割的v4v6混合地址，返回v4和v6地址与有效地址
func SplitIPAddr(v4v6Bundle string) (string, string, string) {
	ipList := strings.Split(v4v6Bundle, "/")
	ipv4 := ""
	ipv6 := ""
	validIP := ""
	if len(ipList) > 1 {
		// 双栈
		ipv4 = ipList[0]
		ipv6 = ipList[1]
		validIP = ipv4
	} else if len(ipList) == 1 {
		// 仅ipv4|ipv6
		if strings.Contains(ipList[0], ":") {
			ipv6 = ipList[0]
			validIP = ipv6
		} else {
			ipv4 = ipList[0]
			validIP = ipv4
		}
	}
	return ipv4, ipv6, validIP
}

func IsFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	lettersLength := big.NewInt(int64(len(letters)))
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, lettersLength)
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}

func Uint64SubInt64(a uint64, b int64) uint64 {
	if b < 0 {
		return a + uint64(-b)
	}
	if a < uint64(b) {
		return 0
	}
	return a - uint64(b)
}
