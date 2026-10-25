package operation_setting

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const PurchaseLinksOptionKey = "payment_setting.purchase_links"

var PurchaseTierIDs = [...]string{"10", "20", "50", "100", "200", "other"}

func validPurchaseURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") &&
		parsed.Hostname() != "" && parsed.User == nil &&
		!strings.ContainsAny(value, "\r\n\t ") && len(value) <= 2048
}

// ParsePurchaseLinks accepts only the six storefront destinations. Empty values
// explicitly disable a tier; missing "other" retains the legacy storefront.
func ParsePurchaseLinks(raw string) (map[string]string, error) {
	if len(raw) > 16384 {
		return nil, fmt.Errorf("充值档位链接配置过长")
	}
	var links map[string]string
	if err := common.UnmarshalJsonStr(raw, &links); err != nil || links == nil {
		return nil, fmt.Errorf("充值档位链接必须为 JSON 对象")
	}
	for tier, value := range links {
		known := false
		for _, allowed := range PurchaseTierIDs {
			if tier == allowed {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("不支持的充值档位：%s", tier)
		}
		value = strings.TrimSpace(value)
		if value != "" && !validPurchaseURL(value) {
			return nil, fmt.Errorf("充值档位 %s 请输入有效的 http 或 https 购买链接", tier)
		}
		links[tier] = value
	}
	return links, nil
}

func GetPurchaseLinks() map[string]string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	configured := GetPaymentSetting().PurchaseLinks
	links := make(map[string]string, len(PurchaseTierIDs))
	for _, tier := range PurchaseTierIDs {
		value := strings.TrimSpace(configured[tier])
		if tier == "other" {
			if _, present := configured[tier]; !present {
				value = strings.TrimSpace(common.TopUpLink)
			}
		}
		if value != "" && !validPurchaseURL(value) {
			value = ""
		}
		links[tier] = value
	}
	return links
}
