package model

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

// PricingMappedEndpoint exposes the public gateway route inherited from a
// channel's final mapped model without exposing the channel's mapping table.
type PricingMappedEndpoint struct {
	Type                 string                             `json:"type"`
	Kind                 string                             `json:"kind"`
	Path                 string                             `json:"path"`
	Method               string                             `json:"method"`
	QueryPath            string                             `json:"query_path,omitempty"`
	QueryMethod          string                             `json:"query_method,omitempty"`
	ExampleDuration      int                                `json:"example_duration,omitempty"`
	ExampleResolution    string                             `json:"example_resolution,omitempty"`
	ExampleImageRequired bool                               `json:"example_image_required,omitempty"`
	Parameters           *common.ModelIntegrationParameters `json:"parameters,omitempty"`
}

func resolvePricingMappedModel(modelName string, rawMapping *string) (string, bool) {
	if rawMapping == nil || strings.TrimSpace(*rawMapping) == "" {
		return modelName, false
	}
	var mapping map[string]string
	if err := common.UnmarshalJsonStr(*rawMapping, &mapping); err != nil {
		return modelName, false
	}
	current := modelName
	visited := map[string]bool{current: true}
	for {
		next := strings.TrimSpace(mapping[current])
		if next == "" || next == current {
			return current, current != modelName
		}
		if visited[next] {
			return modelName, false
		}
		visited[next] = true
		current = next
	}
}

func pricingEndpointKind(endpointType string, path string) string {
	switch {
	case strings.Contains(path, "/video"), endpointType == string(constant.EndpointTypeOpenAIVideo):
		return "video"
	case strings.Contains(path, "/images/"), endpointType == string(constant.EndpointTypeImageGeneration):
		return "image"
	case endpointType == string(constant.EndpointTypeOpenAI),
		endpointType == string(constant.EndpointTypeOpenAIResponse),
		endpointType == string(constant.EndpointTypeAnthropic),
		endpointType == string(constant.EndpointTypeGemini):
		return "text"
	default:
		return "other"
	}
}

func makePricingMappedEndpoint(endpointType string, info common.EndpointInfo) *PricingMappedEndpoint {
	if !strings.HasPrefix(info.Path, "/") {
		return nil
	}
	method := info.Method
	if method == "" {
		method = "POST"
	}
	queryMethod := info.QueryMethod
	if info.QueryPath != "" && queryMethod == "" {
		queryMethod = "GET"
	}
	return &PricingMappedEndpoint{
		Type:        endpointType,
		Kind:        pricingEndpointKind(endpointType, info.Path),
		Path:        info.Path,
		Method:      method,
		QueryPath:   info.QueryPath,
		QueryMethod: queryMethod,
	}
}

func mappedEndpointsFromMetadata(meta *Model) []PricingMappedEndpoint {
	if meta == nil || strings.TrimSpace(meta.Endpoints) == "" {
		return nil
	}
	var configured map[string]interface{}
	if err := common.UnmarshalJsonStr(meta.Endpoints, &configured); err != nil {
		return nil
	}
	keys := make([]string, 0, len(configured))
	for key := range configured {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var endpoints []PricingMappedEndpoint
	for _, key := range keys {
		info := common.EndpointInfo{Method: "POST"}
		switch value := configured[key].(type) {
		case string:
			info.Path = value
		case map[string]interface{}:
			info.Path, _ = value["path"].(string)
			if method, ok := value["method"].(string); ok && method != "" {
				info.Method = strings.ToUpper(method)
			}
			info.QueryPath, _ = value["query_path"].(string)
			info.QueryMethod, _ = value["query_method"].(string)
		}
		if endpoint := makePricingMappedEndpoint(key, info); endpoint != nil {
			endpoints = appendPricingEndpoint(endpoints, *endpoint)
		}
	}
	return endpoints
}

// Provider catalog prefixes are part of the mapped ID, but do not change the
// gateway video contract. Normalize only the prefixes used by these catalogs.
func normalizePricingVideoModelName(modelName string) string {
	return common.NormalizeGatewayVideoModelName(modelName)
}

func pricingVideoEndpoint(modelName string) *PricingMappedEndpoint {
	name := normalizePricingVideoModelName(modelName)
	var path string
	switch {
	case common.IsXJZVideoModel(modelName):
		path = "/v1/videos"
	case strings.HasPrefix(name, "官转"):
		path = "/v1/video/generations"
	case name == "grok-imagine-video-1.5",
		name == "sd-mini",
		name == "sd-2-c1",
		name == "sd-2.5",
		name == "sd-2.5-c1",
		name == "官方h3-1080p",
		name == "官方h3-2k",
		strings.HasPrefix(name, "004系列/"),
		strings.HasPrefix(name, "omni-video-"),
		strings.HasPrefix(name, "videos-"),
		strings.HasPrefix(name, "sd2"),
		strings.HasPrefix(name, "sd-2-"),
		strings.HasPrefix(name, "seedance2."),
		strings.HasPrefix(name, "seedance-2."),
		strings.HasPrefix(name, "wan3.0-video"),
		strings.HasPrefix(name, "wan3.0-image"),
		strings.HasPrefix(name, "wan3.0("),
		strings.HasPrefix(name, "wan3.0（"):
		path = "/v1/videos"
	case name == "grok-imagine-video",
		name == "ko3",
		name == "wan3.0",
		strings.HasPrefix(name, "wan3.0-"),
		strings.HasPrefix(name, "sora"),
		strings.HasPrefix(name, "veo"),
		strings.HasPrefix(name, "kling"),
		strings.HasPrefix(name, "minimax-h3"),
		strings.HasPrefix(name, "video-2."):
		path = "/v1/video/async-generations"
	default:
		return nil
	}
	endpoint := makePricingMappedEndpoint(string(constant.EndpointTypeOpenAIVideo), common.EndpointInfo{
		Path: path, Method: "POST", QueryPath: path + "/{task_id}", QueryMethod: "GET",
	})
	if endpoint != nil && common.IsXJZVideoModel(modelName) {
		endpoint.ExampleDuration, endpoint.ExampleResolution, endpoint.ExampleImageRequired = pricingXJZVideoExample(modelName)
	}
	if endpoint != nil {
		endpoint.Parameters = common.GetVideoIntegrationParameters(modelName)
	}
	return endpoint
}

func pricingXJZVideoExample(modelName string) (int, string, bool) {
	switch strings.TrimSpace(modelName) {
	case "特价dola2.5-900-720P-固定30s":
		return 30, "720p", true
	case "特价dola-sd2.0-900-720p":
		return 5, "720p", true
	case "TTP-grok":
		return 6, "720p", false
	case "gz稳定sd2.0-903-720P", "特价sd2.0-720p-933-过脸", "特价稳sd2.0-933-720p":
		return 5, "720p", false
	case "Xminimex-h3-2K-933":
		return 5, "2k", false
	case "特价稳sd-2.5-30s-720P可10图":
		return 30, "720p", false
	case "gz稳定sd2.5-720全参过脸", "特价sd2.0-933-过脸":
		return 4, "720p", false
	case "gz稳定sd2.5-480全参过脸":
		return 4, "480p", false
	case "gz稳定sd2.5-1080全参过脸":
		return 4, "1080p", false
	case "MiniMaxH3":
		return 4, "2K", false
	default:
		return 0, "", false
	}
}

func getPricingIntegrationEndpoints(channelType int, modelName string, metadata map[string]*Model) []PricingMappedEndpoint {
	return getPricingIntegrationEndpointsForModels(channelType, modelName, modelName, metadata)
}

func getPricingIntegrationEndpointsForModels(channelType int, modelName, originalModelName string, metadata map[string]*Model) []PricingMappedEndpoint {
	if endpoints := mappedEndpointsFromMetadata(metadata[modelName]); len(endpoints) > 0 {
		for index := range endpoints {
			if endpoints[index].Kind == "video" {
				endpoints[index].Parameters = common.GetVideoIntegrationParameters(modelName)
			}
			if endpoints[index].Kind == "image" {
				endpoints[index].Parameters = common.GetImageIntegrationParameters(modelName)
				if parameters := endpoints[index].Parameters; parameters != nil && strings.Contains(endpoints[index].Path, "async-generations") {
					parameters.ResolutionField, parameters.ImageField = "output_resolution", "image_urls"
				}
			}
		}
		return endpoints
	}
	if endpoint := pricingVideoEndpoint(modelName); endpoint != nil {
		return []PricingMappedEndpoint{*endpoint}
	}
	// An opaque upstream alias must not turn a known public video model into
	// the OpenAI channel's default chat endpoint. Explicit upstream metadata
	// and a recognized upstream protocol above still take precedence.
	if originalModelName != modelName {
		if endpoint := pricingVideoEndpoint(originalModelName); endpoint != nil {
			return []PricingMappedEndpoint{*endpoint}
		}
	}
	if parameters := common.GetImageIntegrationParameters(modelName); parameters != nil && channelType != constant.ChannelTypeGemini && channelType != constant.ChannelTypeVertexAi {
		endpoint := makePricingMappedEndpoint(string(constant.EndpointTypeImageGeneration), common.EndpointInfo{Path: "/v1/images/generations", Method: "POST"})
		endpoint.Parameters = parameters
		return []PricingMappedEndpoint{*endpoint}
	}
	var endpoints []PricingMappedEndpoint
	for _, endpointType := range common.GetEndpointTypesByChannelType(channelType, modelName) {
		if info, ok := common.GetDefaultEndpointInfo(endpointType); ok {
			if endpoint := makePricingMappedEndpoint(string(endpointType), info); endpoint != nil {
				endpoints = appendPricingEndpoint(endpoints, *endpoint)
			}
		}
	}
	return endpoints
}

func appendPricingEndpoint(endpoints []PricingMappedEndpoint, candidate PricingMappedEndpoint) []PricingMappedEndpoint {
	for _, existing := range endpoints {
		if existing.Type == candidate.Type && existing.Path == candidate.Path && existing.Method == candidate.Method && existing.QueryPath == candidate.QueryPath {
			return endpoints
		}
	}
	return append(endpoints, candidate)
}
