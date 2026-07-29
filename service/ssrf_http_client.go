package service

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

type ssrfPinnedTransport struct {
	base     *http.Transport
	resolver common.SSRFResolver
}

type closeIdleReadCloser struct {
	io.ReadCloser
	once      sync.Once
	closeIdle func()
}

func (c *closeIdleReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.once.Do(c.closeIdle)
	return err
}

func (t *ssrfPinnedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("invalid SSRF-protected request")
	}
	fetchSetting := system_setting.GetFetchSetting()
	if fetchSetting == nil {
		return nil, fmt.Errorf("fetch setting is not available")
	}
	if !fetchSetting.EnableSSRFProtection {
		return t.base.RoundTrip(req)
	}

	target, err := common.ResolveURLWithFetchSetting(
		req.Context(),
		req.URL.String(),
		fetchSetting.EnableSSRFProtection,
		fetchSetting.AllowPrivateIp,
		fetchSetting.DomainFilterMode,
		fetchSetting.IpFilterMode,
		fetchSetting.DomainList,
		fetchSetting.IpList,
		fetchSetting.AllowedPorts,
		fetchSetting.ApplyIPFilterForDomain,
		t.resolver,
	)
	if err != nil {
		return nil, fmt.Errorf("request reject: %w", err)
	}
	if target == nil || target.DialAddress() == "" {
		return nil, fmt.Errorf("request reject: URL did not resolve to a usable endpoint")
	}

	transport := t.base.Clone()
	if transport.Proxy != nil {
		proxyURL, proxyErr := transport.Proxy(req)
		if proxyErr != nil {
			return nil, fmt.Errorf("resolve outbound proxy: %w", proxyErr)
		}
		if proxyURL != nil && strings.EqualFold(proxyURL.Scheme, "https") {
			return nil, fmt.Errorf("SSRF-protected requests do not support HTTPS forward proxies")
		}
		if proxyURL == nil {
			transport.Proxy = nil
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	pinnedReq := req.Clone(req.Context())
	pinnedURL := *req.URL
	originalAuthority := req.URL.Host
	pinnedURL.Host = target.DialAddress()
	pinnedReq.URL = &pinnedURL
	if pinnedReq.Host == "" {
		pinnedReq.Host = originalAuthority
	}

	if strings.EqualFold(req.URL.Scheme, "https") {
		tlsConfig := transport.TLSClientConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		} else {
			tlsConfig = tlsConfig.Clone()
		}
		// The TCP connection is pinned to an IP literal, while certificate
		// verification and SNI must continue using the original hostname.
		tlsConfig.ServerName = target.Host
		transport.TLSClientConfig = tlsConfig
	}

	resp, err := transport.RoundTrip(pinnedReq)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	if resp == nil {
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("SSRF-protected transport returned an empty response")
	}
	resp.Request = req
	if resp.Body == nil {
		transport.CloseIdleConnections()
		return resp, nil
	}
	resp.Body = &closeIdleReadCloser{
		ReadCloser: resp.Body,
		closeIdle:  transport.CloseIdleConnections,
	}
	return resp, nil
}

func cloneHTTPTransport(roundTripper http.RoundTripper) (*http.Transport, error) {
	if roundTripper == nil {
		roundTripper = http.DefaultTransport
	}
	transport, ok := roundTripper.(*http.Transport)
	if !ok || transport == nil {
		return nil, fmt.Errorf("SSRF protection requires an HTTP transport")
	}
	return transport, nil
}

func sameHTTPOrigin(left *url.URL, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	if !strings.EqualFold(left.Scheme, right.Scheme) {
		return false
	}
	if !strings.EqualFold(strings.TrimSuffix(left.Hostname(), "."), strings.TrimSuffix(right.Hostname(), ".")) {
		return false
	}
	return effectiveHTTPPort(left) == effectiveHTTPPort(right)
}

func effectiveHTTPPort(value *url.URL) string {
	if value == nil {
		return ""
	}
	if port := value.Port(); port != "" {
		return port
	}
	if strings.EqualFold(value.Scheme, "https") {
		return "443"
	}
	if strings.EqualFold(value.Scheme, "http") {
		return "80"
	}
	return ""
}

func stripRedirectSecrets(req *http.Request, via []*http.Request) {
	if req == nil || len(via) == 0 || sameHTTPOrigin(req.URL, via[len(via)-1].URL) {
		return
	}
	for _, header := range []string{
		"Authorization",
		"Cookie",
		"Proxy-Authorization",
		"X-Api-Key",
		"X-Goog-Api-Key",
		"X-Webhook-Signature",
	} {
		req.Header.Del(header)
	}
	req.Header.Del("Referer")
}

func newSSRFProtectedHTTPClientWithResolver(base *http.Client, resolver common.SSRFResolver) (*http.Client, error) {
	if base == nil {
		base = http.DefaultClient
	}
	transport, err := cloneHTTPTransport(base.Transport)
	if err != nil {
		return nil, err
	}

	client := *base
	client.Transport = &ssrfPinnedTransport{
		base:     transport,
		resolver: resolver,
	}
	baseCheckRedirect := base.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		stripRedirectSecrets(req, via)
		if baseCheckRedirect != nil {
			return baseCheckRedirect(req, via)
		}
		return nil
	}
	return &client, nil
}

// DoSSRFProtectedRequest validates every redirect target, resolves it once,
// and pins the actual connection to that validated IP address.
func DoSSRFProtectedRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	protectedClient, err := newSSRFProtectedHTTPClientWithResolver(client, nil)
	if err != nil {
		return nil, err
	}
	return protectedClient.Do(req)
}
