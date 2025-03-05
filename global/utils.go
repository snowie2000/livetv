// utils
package global

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	freq "github.com/imroc/req/v3"
	"golang.org/x/net/proxy"
)

var (
	dialer = &net.Dialer{
		Timeout: HttpClientTimeout,
	}
	DefaultTransport = &http.Transport{
		Dial:                dialer.Dial,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
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

func removeDocumentPart(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	p := u.Path
	if p == "" || strings.HasSuffix(p, "/") {
		return u.String(), nil // Already a directory, or empty path.
	}

	dir := path.Dir(p)
	fmt.Println("dir", dir)
	if dir == "." {
		u.Path = "/" // if file is in root, set path to "/"
	} else {
		if !strings.HasSuffix(dir, "/") {
			dir = dir + "/"
		}
		u.Path = dir // Add trailing slash to create a directory URL.
	}

	baseUrl := u.String()
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl = baseUrl + "/"
	}
	return baseUrl, nil
}

func MergeUrl(baseUrl string, partialUrl string) string {
	if baseUrl == "" {
		return partialUrl
	}

	if strings.HasPrefix(partialUrl, "/") {
		u, _ := url.Parse(baseUrl)
		u.Path = ""
		u.RawQuery = ""
		u.Fragment = ""
		return u.String() + partialUrl
	} else {
		baseUrl, _ = removeDocumentPart(baseUrl)
		return baseUrl + partialUrl
	}
}

func TransportWithProxy(proxyUrl string) *http.Transport {
	tr := DefaultTransport
	if proxyUrl != "" {
		tr = DefaultTransport.Clone()
		tr.DisableKeepAlives = true
		if u, err := url.Parse(proxyUrl); err == nil {
			if p, e := proxy.FromURL(u, dialer); e == nil {
				tr.Dial = p.Dial
			} else {
				log.Println("Proxy setup error:", e)
			}
		}
	}
	return tr
}

func CloseBody(resp any) {
	if resp, ok := resp.(*http.Response); ok {
		if resp != nil && resp.Body != nil {
			// if the body is already read in some scenarios, the below operation becomes a no-op
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			fmt.Println("http response closed")
		}
	}
	if resp, ok := resp.(*freq.Response); ok {
		if resp != nil && resp.Body != nil {
			// if the body is already read in some scenarios, the below operation becomes a no-op
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			fmt.Println("freq response closed")
		}
	}
}

type lazybuf struct {
	s   string
	buf []byte
	w   int
}

func (b *lazybuf) index(i int) byte {
	if b.buf != nil {
		return b.buf[i]
	}
	return b.s[i]
}

func (b *lazybuf) append(c byte) {
	if b.buf == nil {
		if b.w < len(b.s) && b.s[b.w] == c {
			b.w++
			return
		}
		b.buf = make([]byte, len(b.s))
		copy(b.buf, b.s[:b.w])
	}
	b.buf[b.w] = c
	b.w++
}

func (b *lazybuf) string() string {
	if b.buf == nil {
		return b.s[:b.w]
	}
	return string(b.buf[:b.w])
}

func cleanPathSimple(path string) string {
	if path == "" {
		return "."
	}

	rooted := path[0] == '/'
	n := len(path)

	// Invariants:
	//	reading from path; r is index of next byte to process.
	//	writing to buf; w is index of next byte to write.
	//	dotdot is index in buf where .. must stop, either because
	//		it is the leading slash or it is a leading ../../.. prefix.
	out := lazybuf{s: path}
	r, dotdot := 0, 0
	if rooted {
		out.append('/')
		r, dotdot = 1, 1
	}

	for r < n {
		switch {
		case path[r] == '.' && (r+1 == n || path[r+1] == '/'):
			// . element
			r++
		case path[r] == '.' && path[r+1] == '.' && (r+2 == n || path[r+2] == '/'):
			// .. element: remove to last /
			r += 2
			switch {
			case out.w > dotdot:
				// can backtrack
				out.w--
				for out.w > dotdot && out.index(out.w) != '/' {
					out.w--
				}
			case !rooted:
				// cannot backtrack, but not rooted, so append .. element.
				if out.w > 0 {
					out.append('/')
				}
				out.append('.')
				out.append('.')
				dotdot = out.w
			}
		default:
			if path[r] == '/' {
				r++
			}
			// real path element.
			// add slash if needed
			if rooted && out.w != 1 || !rooted && out.w != 0 {
				out.append('/')
			}
			// copy element
			for ; r < n && path[r] != '/'; r++ {
				out.append(path[r])
			}
		}
	}

	// Turn empty string into "."
	if out.w == 0 {
		return "."
	}

	return out.string()
}

func CleanUrl(Url string) string {
	parsedURL, err := url.Parse(Url)
	if err != nil {
		return Url
	}

	// Resolve the path using path resolution
	parsedURL.Path = cleanPathSimple(parsedURL.Path) // Remove trailing segments

	// Get the final clean URL as a string
	cleanURL := parsedURL.String()

	return cleanURL
}
