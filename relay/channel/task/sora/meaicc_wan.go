package sora

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

func isMeaiccWanModel(baseURL, upstreamModel string) bool {
	return common.IsBaseURLHost(baseURL, "api.meaicc.com") &&
		strings.TrimSpace(upstreamModel) == "w3-c1"
}

// The official creation client sends input.media and parameters to /v1/videos.
// Keep the gateway's flat reference fields public and normalize only this model.
func buildMeaiccWanRequestBody(body []byte, upstreamModel string) ([]byte, error) {
	encoded, err := buildSeedance2RequestBody(body, upstreamModel)
	if err != nil {
		return nil, err
	}
	var request seedance2Request
	if err := common.Unmarshal(encoded, &request); err != nil {
		return nil, err
	}
	request.Parameters.Resolution = strings.ToUpper(request.Parameters.Resolution)
	return common.Marshal(request)
}
