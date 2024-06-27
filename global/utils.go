// utils
package global

import (
	"net/url"
	"path"
	"strings"
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
