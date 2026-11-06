package service

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

// DrawingLogMediaOrigins returns the origins used to recognize stored copies
// in older rows, which did not record whether a URL was transferred.
func DrawingLogMediaOrigins() []*url.URL {
	origins := make([]*url.URL, 0)
	add := func(raw string) {
		if origin, err := url.Parse(raw); err == nil && origin.Host != "" {
			origins = append(origins, origin)
		}
	}
	add(system_setting.ServerAddress)
	// Keep recognizing existing GHLINK copies after a storage provider change.
	add("https://media.ghlink.top")
	for _, provider := range GetMediaStorageProviders() {
		add(provider.UploadURL)
	}
	return origins
}

// PrepareDrawingLogImages projects the chosen URLs onto the existing log
// response fields. It does not change the persisted row or the relay result.
func PrepareDrawingLogImages(task *model.Midjourney, origins []*url.URL, proxyURL string) {
	if task == nil {
		return
	}
	var properties map[string]json.RawMessage
	if common.UnmarshalJsonStr(task.Properties, &properties) != nil {
		properties = nil
	}
	var storedURLs []string
	_ = common.Unmarshal(properties["image_urls"], &storedURLs)
	var propertyURL string
	_ = common.Unmarshal(properties["image_url"], &propertyURL)
	var resolved bool
	_ = common.Unmarshal(properties["image_urls_resolved"], &resolved)

	primary, _ := normalizeImageGenerationResultURL(task.ImageUrl)
	candidates := append([]string{primary, propertyURL}, storedURLs...)

	seen := make(map[string]struct{}, len(candidates))
	selected := make([]string, 0, len(candidates))
	siteURLs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		result, ok := normalizeImageGenerationResultURL(candidate)
		if !ok {
			continue
		}
		if _, exists := seen[result]; exists {
			continue
		}
		seen[result] = struct{}{}
		selected = append(selected, result)
		if isDrawingLogSiteURL(result, origins) {
			siteURLs = append(siteURLs, result)
		}
	}
	if task.Action != constant.MjActionImageGeneration || !resolved {
		// Legacy rows have no association between alternate URLs. Prefer
		// known site copies; use upstream URLs only if no copy is present.
		if len(siteURLs) > 0 {
			selected = siteURLs
		} else if proxyURL != "" && len(selected) > 0 && selected[0] == primary {
			selected[0] = proxyURL
		}
	}

	task.ImageUrl = ""
	if len(selected) > 0 {
		task.ImageUrl = selected[0]
	}
	if properties == nil {
		return
	}
	if _, hasURL := properties["image_url"]; !hasURL {
		if _, hasURLs := properties["image_urls"]; !hasURLs {
			return
		}
	}
	delete(properties, "image_url")
	delete(properties, "image_urls")
	// Preserve the complete result array for clients using this field.
	if len(selected) > 0 {
		imageURLs, err := common.Marshal(selected)
		if err != nil {
			return
		}
		properties["image_urls"] = imageURLs
	}
	if encoded, err := common.Marshal(properties); err == nil {
		task.Properties = string(encoded)
	}
}

func isDrawingLogSiteURL(raw string, origins []*url.URL) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Host == "" {
		return strings.HasPrefix(parsed.Path, "/") && !strings.Contains(raw, "\\")
	}
	for _, origin := range origins {
		if sameHTTPOrigin(parsed, origin) {
			return true
		}
	}
	return false
}
