package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sequenceSSRFResolver struct {
	mu        sync.Mutex
	responses [][]net.IPAddr
	calls     int
}

func (r *sequenceSSRFResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	index := r.calls
	r.calls++
	if index >= len(r.responses) {
		index = len(r.responses) - 1
	}
	return r.responses[index], nil
}

func (r *sequenceSSRFResolver) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func configureSSRFHTTPClientTest(t *testing.T) {
	t.Helper()
	fetchSetting := system_setting.GetFetchSetting()
	original := *fetchSetting
	t.Cleanup(func() {
		*fetchSetting = original
	})
	fetchSetting.EnableSSRFProtection = true
	fetchSetting.AllowPrivateIp = false
	fetchSetting.DomainFilterMode = false
	fetchSetting.IpFilterMode = false
	fetchSetting.DomainList = nil
	fetchSetting.IpList = nil
	fetchSetting.AllowedPorts = []string{"80", "443"}
	fetchSetting.ApplyIPFilterForDomain = true
}

func TestSSRFProtectedHTTPClientPinsValidatedDNSAnswer(t *testing.T) {
	configureSSRFHTTPClientTest(t)

	resolver := &sequenceSSRFResolver{responses: [][]net.IPAddr{
		{{IP: net.ParseIP("93.184.216.34")}},
		{{IP: net.ParseIP("127.0.0.1")}},
	}}
	dialAddress := make(chan string, 1)
	requestHost := make(chan string, 1)
	serverErr := make(chan error, 1)
	transport := &http.Transport{
		DialContext: func(_ context.Context, _ string, stringAddress string) (net.Conn, error) {
			dialAddress <- stringAddress
			clientConn, serverConn := net.Pipe()
			go func() {
				defer serverConn.Close()
				request, err := http.ReadRequest(bufio.NewReader(serverConn))
				if err != nil {
					serverErr <- err
					return
				}
				requestHost <- request.Host
				_ = request.Body.Close()
				_, err = fmt.Fprint(
					serverConn,
					"HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nok",
				)
				serverErr <- err
			}()
			return clientConn, nil
		},
	}
	client, err := newSSRFProtectedHTTPClientWithResolver(
		&http.Client{Transport: transport},
		resolver,
	)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "http://rebind.example/resource", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.Equal(t, "ok", string(body))
	assert.Equal(t, "93.184.216.34:80", <-dialAddress)
	assert.Equal(t, "rebind.example", <-requestHost)
	require.NoError(t, <-serverErr)
	assert.Equal(t, 1, resolver.callCount())
	assert.Equal(t, "rebind.example", resp.Request.URL.Host)
}

func TestSSRFProtectedHTTPClientRejectsPrivateAnswerBeforeDial(t *testing.T) {
	configureSSRFHTTPClientTest(t)

	resolver := &sequenceSSRFResolver{responses: [][]net.IPAddr{
		{{IP: net.ParseIP("127.0.0.1")}},
	}}
	var dialCount atomic.Int32
	transport := &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			dialCount.Add(1)
			return nil, fmt.Errorf("unexpected dial")
		},
	}
	client, err := newSSRFProtectedHTTPClientWithResolver(
		&http.Client{Transport: transport},
		resolver,
	)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "http://private.example/resource", nil)
	require.NoError(t, err)
	_, err = client.Do(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private IP address not allowed")
	assert.Zero(t, dialCount.Load())
	assert.Equal(t, 1, resolver.callCount())
}

func TestStripRedirectSecretsAcrossOrigins(t *testing.T) {
	previous, err := http.NewRequest(http.MethodGet, "https://source.example/start", nil)
	require.NoError(t, err)
	redirect, err := http.NewRequest(http.MethodGet, "https://target.example/end", nil)
	require.NoError(t, err)
	for _, header := range []string{
		"Authorization",
		"Cookie",
		"Proxy-Authorization",
		"X-Api-Key",
		"X-Goog-Api-Key",
		"X-Webhook-Signature",
		"Referer",
	} {
		redirect.Header.Set(header, "sensitive")
	}

	stripRedirectSecrets(redirect, []*http.Request{previous})

	for _, header := range []string{
		"Authorization",
		"Cookie",
		"Proxy-Authorization",
		"X-Api-Key",
		"X-Goog-Api-Key",
		"X-Webhook-Signature",
		"Referer",
	} {
		assert.Empty(t, redirect.Header.Get(header), header)
	}
}
