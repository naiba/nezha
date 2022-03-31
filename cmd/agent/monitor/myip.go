package monitor

import (
	"fmt"
	"io/ioutil"
	"math/rand"
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
		"https://ipapi.co/json",
		"https://freegeoip.app/json/",
		"http://ip-api.com/json/",
		"https://extreme-ip-lookup.com/json/",
		// "https://ip.seeip.org/geoip",
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
	if isV6 {
		resp, err = httpGetWithUA(httpClientV6, servers[rand.Intn(len(servers))])
	} else {
		resp, err = httpGetWithUA(httpClientV4, servers[rand.Intn(len(servers))])
	}
	if err != nil {
		return ip
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ip
	}
	resp.Body.Close()
	err = utils.Json.Unmarshal(body, &ip)
	if err != nil {
		return ip
	}
	if ip.IP == "" && ip.Query != "" {
		ip.IP = ip.Query
	}
	// 没取到 v6 IP
	if isV6 && !strings.Contains(ip.IP, ":") {
		return ip
	}
	// 没取到 v4 IP
	if !isV6 && !strings.Contains(ip.IP, ".") {
		return ip
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
