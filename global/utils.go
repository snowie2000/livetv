// utils
package global

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/proxy"
)

func GetBaseURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parsedURL.RawQuery = ""

	// Remove the last element (document) from the path
	parsedURL.Path = path.Dir(parsedURL.Path) + "/"

	// Rebuild the URL without the document part
	return parsedURL.String()
}

func IsValidURL(u string) bool {
	_, err := url.ParseRequestURI(u)
	if err == nil {
		uu, err := url.Parse(u)
		return err == nil && uu.Scheme != "" && uu.Host != ""
	}
	return false
}

func MergeUrl(baseUrl string, partialUrl string) string {
	if strings.HasPrefix(partialUrl, "/") {
		u, _ := url.Parse(baseUrl)
		u.Path = ""
		u.RawQuery = ""
		u.Fragment = ""
		return u.String() + partialUrl
	}
	return baseUrl + partialUrl
}

func TransportWithProxy(proxyUrl string) *http.Transport {
	d := &net.Dialer{
		Timeout: HttpClientTimeout,
	}
	tr := &http.Transport{
		Dial:            d.Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if proxyUrl != "" {
		if u, err := url.Parse(proxyUrl); err == nil {
			if p, e := proxy.FromURL(u, d); e == nil {
				tr.Dial = p.Dial
			} else {
				log.Println("Proxy setup error:", e)
			}
		}
	}
	return tr
}
