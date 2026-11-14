package billing_setting

import (
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

func ParseVideoReferencePrices(raw string) (map[string]map[string]float64, error) {
	var prices map[string]map[string]float64
	if err := common.UnmarshalJsonStr(raw, &prices); err != nil || prices == nil {
		return nil, fmt.Errorf("video reference prices must be a JSON object")
	}
	result := make(map[string]map[string]float64, len(prices))
	for name, tiers := range prices {
		if strings.TrimSpace(name) == "" || tiers == nil {
			return nil, fmt.Errorf("invalid video reference price model")
		}
		result[name] = map[string]float64{}
		for resolution, price := range tiers {
			tier := NormalizeVideoResolution(resolution)
			if tier == "" || math.IsNaN(price) || math.IsInf(price, 0) || price < 0 || price > 1e6 {
				return nil, fmt.Errorf("invalid reference-video price for %s / %s", name, resolution)
			}
			if _, exists := result[name][tier]; exists {
				return nil, fmt.Errorf("duplicate reference-video resolution %s", tier)
			}
			result[name][tier] = price
		}
	}
	return result, nil
}
