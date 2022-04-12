package monitor

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/naiba/nezha/pkg/utils"
)

type geoIP struct {
	CountryCode  string `json:"country_code,omitempty"`
	CountryCode2 string `json:"countryCode,omitempty"`
	IP           string `json:"ip,omitempty"`
	Query        string `json:"query,omitempty"`
	Location     struct {
		CountryCode string `json:"country_code,omitempty"`
	} `json:"location,omitempty"`
}

func (ip *geoIP) Unmarshal(body []byte) error {
	if err := utils.Json.Unmarshal(body, ip); err != nil {
		return err
	}
	if ip.IP == "" && ip.Query != "" {
		ip.IP = ip.Query
	}
	if ip.CountryCode == "" && ip.CountryCode2 != "" {
		ip.CountryCode = ip.CountryCode2
	}
	if ip.CountryCode == "" && ip.Location.CountryCode != "" {
		ip.CountryCode = ip.Location.CountryCode
	}
	return nil
}

var (
	geoIPApiList = []string{
		"https://api.myip.la/en?json",
		"https://api.ip.sb/geoip",
		"https://ipapi.co/json",
		"https://freegeoip.app/json/",
		"http://ip-api.com/json/",
		// "https://extreme-ip-lookup.com/json/",
		// "https://ip.seeip.org/geoip",
	}
	cachedIP, cachedCountry string
	httpClientV4            = utils.NewSingleStackHTTPClient(time.Second*20, time.Second*5, time.Second*10, false)
	httpClientV6            = utils.NewSingleStackHTTPClient(time.Second*20, time.Second*5, time.Second*10, true)
)

// UpdateIP 每30分钟更新一次IP地址与国家码的缓存
func UpdateIP() {
	for {
		ipv4 := fetchGeoIP(geoIPApiList, false)
		ipv6 := fetchGeoIP(geoIPApiList, true)
		if ipv4.IP == "" && ipv6.IP == "" {
			time.Sleep(time.Minute)
			continue
		}
		if ipv4.IP == "" || ipv6.IP == "" {
			cachedIP = fmt.Sprintf("%s%s", ipv4.IP, ipv6.IP)
		} else {
			cachedIP = fmt.Sprintf("%s/%s", ipv4.IP, ipv6.IP)
		}
		if ipv4.CountryCode != "" {
			cachedCountry = ipv4.CountryCode
		} else if ipv6.CountryCode != "" {
			cachedCountry = ipv6.CountryCode
		}
		time.Sleep(time.Minute * 30)
	}
}

func fetchGeoIP(servers []string, isV6 bool) geoIP {
	var ip geoIP
	var resp *http.Response
	var err error
	// 双栈支持参差不齐，不能随机请求，有些 IPv6 取不到 IP
	for i := 0; i < len(servers); i++ {
		if isV6 {
			resp, err = httpClientV6.Get(servers[i])
		} else {
			resp, err = httpClientV4.Get(servers[i])
		}
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if err := ip.Unmarshal(body); err != nil {
				continue
			}
			// 没取到 v6 IP
			if isV6 && !strings.Contains(ip.IP, ":") {
				continue
			}
			// 没取到 v4 IP
			if !isV6 && !strings.Contains(ip.IP, ".") {
				continue
			}
			return ip
		}
	}
	return ip
}

func httpGetWithUA(client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.84 Safari/537.36")
	return client.Do(req)
}
