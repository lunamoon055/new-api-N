package middleware

import (
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// ConfigureTrustedProxies configures which direct peers may supply proxy
// headers used by gin.Context.ClientIP. An empty value intentionally trusts no
// proxies. Trusting an all-network CIDR would make client-controlled forwarded
// headers authoritative, so it is rejected.
func ConfigureTrustedProxies(engine *gin.Engine, raw string) error {
	if engine == nil {
		return fmt.Errorf("gin engine is nil")
	}

	trusted := make([]string, 0)
	seen := make(map[string]struct{})
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if strings.Contains(value, "/") {
			_, network, err := net.ParseCIDR(value)
			if err != nil {
				return fmt.Errorf("invalid trusted proxy CIDR %q: %w", value, err)
			}
			ones, _ := network.Mask.Size()
			if ones == 0 {
				return fmt.Errorf("all-network trusted proxy CIDR %q is forbidden", value)
			}
		} else if net.ParseIP(value) == nil {
			return fmt.Errorf("invalid trusted proxy IP %q", value)
		}

		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		trusted = append(trusted, value)
	}

	if err := engine.SetTrustedProxies(trusted); err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}
	return nil
}
