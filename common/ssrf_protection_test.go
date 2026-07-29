package common

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticSSRFResolver struct {
	addresses []net.IPAddr
	err       error
}

func (r staticSSRFResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r.addresses, r.err
}

func newTestSSRFProtection() *SSRFProtection {
	return &SSRFProtection{
		AllowPrivateIp:         false,
		DomainFilterMode:       false,
		IpFilterMode:           false,
		AllowedPorts:           []int{80, 443},
		ApplyIPFilterForDomain: true,
	}
}

func TestSSRFProtectionResolveURLReturnsPinnedPublicAddress(t *testing.T) {
	protection := newTestSSRFProtection()
	target, err := protection.ResolveURL(
		context.Background(),
		"https://public.example/video",
		staticSSRFResolver{addresses: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}},
	)
	require.NoError(t, err)
	require.NotNil(t, target)
	assert.Equal(t, "public.example", target.Host)
	assert.Equal(t, 443, target.Port)
	assert.Equal(t, "93.184.216.34:443", target.DialAddress())
}

func TestSSRFProtectionResolveURLRejectsAnyPrivateDNSAnswer(t *testing.T) {
	protection := newTestSSRFProtection()
	_, err := protection.ResolveURL(
		context.Background(),
		"http://rebind.example/file",
		staticSSRFResolver{addresses: []net.IPAddr{
			{IP: net.ParseIP("93.184.216.34")},
			{IP: net.ParseIP("127.0.0.1")},
		}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private IP address not allowed")
}

func TestSSRFProtectionCanonicalizesTrailingDotBeforeDomainFilter(t *testing.T) {
	protection := newTestSSRFProtection()
	protection.DomainList = []string{"blocked.example"}

	err := protection.ValidateURL("http://blocked.example./resource")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain in blacklist")
}

func TestSSRFProtectionRejectsEmptyHostAndScopedIP(t *testing.T) {
	protection := newTestSSRFProtection()

	err := protection.ValidateURL("http:///resource")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host is empty")

	err = protection.ValidateURL("http://[fe80::1%25en0]/resource")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scoped IP")
}
