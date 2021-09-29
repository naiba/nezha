package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/naiba/nezha/pkg/utils"
)

type geoIP struct {
	CountryCode string `json:"country_code,omitempty"`
	IP          string `json:"ip,omitempty"`
	Query       string `json:"query,omitempty"`
}

var (
	geoIPApiList = []string{
		"https://api.ip.sb/geoip",
		"https://ip.seeip.org/geoip",
		"https://ipapi.co/json",
		"https://freegeoip.app/json/",
		"http://ip-api.com/json/",
		"https://extreme-ip-lookup.com/json/",
	}
	cachedIP, cachedCountry string
	httpClientV4            = utils.NewSingleStackHTTPClient(time.Second*20, time.Second*5, time.Second*10, false)
	httpClientV6            = utils.NewSingleStackHTTPClient(time.Second*20, time.Second*5, time.Second*10, true)
)

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
			err = json.Unmarshal(body, &ip)
			if err != nil {
				continue
			}
			if ip.IP == "" && ip.Query != "" {
				ip.IP = ip.Query
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
