package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

type ChannelCapabilitySyncResult struct {
	ChannelID         int      `json:"channel_id"`
	UpdatedModels     int      `json:"updated_models"`
	UnconfirmedModels []string `json:"unconfirmed_models"`
	UpdatedAt         int64    `json:"updated_at"`
}

var capabilitySyncLocks [64]sync.Mutex

func SupportsChannelCapabilitySync(channelType int, baseURL string) bool {
	return common.ChannelCapabilityProvider(channelType, baseURL) != ""
}

func unknownVideoParameters(channelType int) *common.ModelIntegrationParameters {
	p := &common.ModelIntegrationParameters{
		CapabilitiesUnknown: true,
		DurationField:       "duration", RatioField: "ratio", ResolutionField: "resolution",
		ImageField: "images", VideoField: "reference_videos", AudioField: "reference_audios",
	}
	if channelType == constant.ChannelTypeV6K {
		p.DurationField, p.VideoField, p.AudioField = "seconds", "videos", "audios"
	}
	return p
}

// RefreshChannelCapabilities fetches metadata only. It does not submit a
// generation, change mappings/prices, or use cached hints to authorize a relay.
func RefreshChannelCapabilities(ctx context.Context, channelID int) (ChannelCapabilitySyncResult, error) {
	result := ChannelCapabilitySyncResult{ChannelID: channelID, UnconfirmedModels: []string{}}
	lock := &capabilitySyncLocks[uint(channelID)%uint(len(capabilitySyncLocks))]
	lock.Lock()
	defer lock.Unlock()
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return result, err
	}
	provider := common.ChannelCapabilityProvider(channel.Type, channel.GetBaseURL())
	if provider == "" {
		return result, errors.New("capability_sync_unsupported")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var catalog map[string]*common.ModelIntegrationParameters
	source := "upstream_capabilities"
	if provider != "zongheng" {
		catalog, err = fetchPublicVideoCapabilityCatalog(ctx, channel.GetBaseURL(), channel.GetSetting().Proxy, provider)
		source = "upstream_catalog"
	} else {
		catalog, err = fetchChannelZonghengCapabilities(ctx, channel)
	}
	if err != nil {
		return result, errors.New("upstream_capabilities_unavailable")
	}
	// A channel edited during a fetch must not publish a stale catalog under its
	// new origin, credential, or model mapping.
	current, err := model.GetChannelById(channelID, true)
	if err != nil {
		return result, err
	}
	if model.ChannelCapabilityScope(channel) != model.ChannelCapabilityScope(current) || channel.GetModelMapping() != current.GetModelMapping() || channel.Models != current.Models {
		return result, errors.New("channel_changed_during_sync")
	}
	items := make([]model.ChannelModelCapability, 0)
	var previous []model.ChannelModelCapability
	if err := model.DB.Where("channel_id = ? AND scope = ?", channel.Id, model.ChannelCapabilityScope(channel)).Find(&previous).Error; err != nil {
		return result, err
	}
	templates := make(map[string]bool)
	for _, item := range previous {
		templates[item.PublicModel] = item.EndpointTemplate != "null" && item.EndpointTemplate != ""
	}
	seenPublicModels := make(map[string]bool)
	for _, publicModel := range strings.Split(channel.Models, ",") {
		publicModel = strings.TrimSpace(publicModel)
		if publicModel == "" || seenPublicModels[publicModel] {
			continue
		}
		seenPublicModels[publicModel] = true
		upstream, _, err := common.ResolveModelMapping(publicModel, channel.GetModelMapping())
		if err != nil {
			return result, err
		}
		p := catalog[upstream]
		if provider != "v6k" && provider != "zongheng" && p == nil && (common.IsOpenAITextModel(upstream) || common.IsImageGenerationModel(upstream)) {
			continue
		}
		if provider != "v6k" && provider != "zongheng" && p == nil && !isPublicCatalogVideo(upstream) && !isPublicCatalogVideo(publicModel) && !templates[publicModel] {
			continue
		}
		if p == nil {
			p = unknownVideoParameters(channel.Type)
		}
		if p.CapabilitiesUnknown || len(p.UnknownFields) > 0 || len(p.Durations) == 0 || len(p.Resolutions) == 0 {
			result.UnconfirmedModels = append(result.UnconfirmedModels, publicModel)
		}
		item, err := model.NewChannelModelCapability(channel, publicModel, upstream, p, source)
		if err != nil {
			return result, err
		}
		if item.EndpointTemplate == "null" && !common.StringsContains(result.UnconfirmedModels, publicModel) {
			result.UnconfirmedModels = append(result.UnconfirmedModels, publicModel)
		}
		items = append(items, item)
	}
	if err := model.ReplaceChannelModelCapabilities(channelID, items); err != nil {
		return result, err
	}
	result.UpdatedModels, result.UpdatedAt = len(items), common.GetTimestamp()
	sort.Strings(result.UnconfirmedModels)
	return result, nil
}

func fetchChannelZonghengCapabilities(ctx context.Context, channel *model.Channel) (map[string]*common.ModelIntegrationParameters, error) {
	keys := channel.GetKeys()
	if len(keys) == 0 {
		return nil, errors.New("missing_channel_key")
	}
	var result map[string]*common.ModelIntegrationParameters
	for index, key := range keys {
		if status, ok := channel.ChannelInfo.MultiKeyStatusList[index]; ok && status != 1 {
			continue
		}
		models, err := FetchZonghengVideoModels(ctx, channel.GetBaseURL(), strings.TrimSpace(key), channel.GetSetting().Proxy)
		if err != nil {
			return nil, err
		}
		current := make(map[string]*common.ModelIntegrationParameters)
		for _, item := range models {
			current[item.ID] = zonghengDocumentationParameters(item)
		}
		if result == nil {
			result = current
			continue
		}
		// Different keys can expose different contracts. Publish only an agreed
		// profile; differing profiles remain unknown instead of taking key one.
		for name, p := range result {
			left, _ := common.Marshal(p)
			right, _ := common.Marshal(current[name])
			if string(left) != string(right) {
				result[name] = unknownVideoParameters(channel.Type)
			}
		}
	}
	if result == nil {
		return nil, errors.New("missing_enabled_channel_key")
	}
	return result, nil
}

func zonghengDocumentationParameters(item ZonghengVideoModel) *common.ModelIntegrationParameters {
	c := item.Capabilities
	p := unknownVideoParameters(constant.ChannelTypeZongheng)
	p.CapabilitiesUnknown = false
	p.Durations, p.Resolutions, p.Ratios = c.Durations, c.Resolutions, c.Ratios
	for _, limit := range []struct {
		name  string
		value *int
		dest  *int
	}{
		{"max_images", c.MaxImages, &p.MaxImages}, {"max_videos", c.MaxVideos, &p.MaxVideos}, {"max_audios", c.MaxAudios, &p.MaxAudios},
	} {
		if limit.value == nil || *limit.value < 0 {
			p.UnknownFields = append(p.UnknownFields, limit.name)
		} else {
			*limit.dest = *limit.value
		}
	}
	return p
}

func fetchPublicVideoCapabilityCatalog(ctx context.Context, baseURL, proxy, provider string) (map[string]*common.ModelIntegrationParameters, error) {
	client, err := GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, NormalizeV6KBaseURL(baseURL)+"/api/pricing", nil)
	if err != nil {
		return nil, err
	}
	// This is a public catalog. Never forward a channel key to this endpoint.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; NewAPI-CapabilitySync)")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog HTTP %d", resp.StatusCode)
	}
	body, err := common.ReadResponseBodyWithLimit(resp.Body, common.MaxUpstreamResponseBodyBytes)
	if err != nil {
		return nil, err
	}
	var catalog struct {
		Success bool `json:"success"`
		Data    []struct {
			ModelName   string   `json:"model_name"`
			Description string   `json:"description"`
			Endpoints   []string `json:"supported_endpoint_types"`
		} `json:"data"`
	}
	if err := common.Unmarshal(body, &catalog); err != nil {
		return nil, err
	}
	if !catalog.Success {
		return nil, errors.New("catalog_unavailable")
	}
	result := make(map[string]*common.ModelIntegrationParameters)
	for _, item := range catalog.Data {
		video := common.StringsContains(item.Endpoints, "openai-video")
		if provider != "v6k" {
			video = video || isPublicCatalogVideo(item.ModelName)
		}
		if item.ModelName != "" && video {
			note := item.Description
			// Suanliai's /docs explicitly publishes capability annotations in
			// exact catalog IDs. Do not infer undocumented shorthand triples.
			if provider == "suanliai" {
				note += "/" + item.ModelName
			}
			p := parseV6KCapabilityNote(note)
			if provider == "suanliai" {
				supplementSuanliaiReferenceTuple(p, item.ModelName)
			}
			if provider != "v6k" {
				p.DurationField, p.ImageField, p.VideoField, p.AudioField = "duration", "referenceImages", "referenceVideos", "referenceAudios"
			}
			result[item.ModelName] = p
		}
	}
	return result, nil
}

var (
	v6kReferenceNote       = regexp.MustCompile(`([0-9]+)\s*(图|音|视)`)
	v6kDurationRangeNote   = regexp.MustCompile(`([0-9]+)\s*[-–~至]\s*([0-9]+)\s*(?:秒|[sS]\b)`)
	v6kFixedDurationNote   = regexp.MustCompile(`固定\s*([0-9]+)\s*秒`)
	v6kDurationListNote    = regexp.MustCompile(`(?:^|/)\s*([0-9]+(?:\s*-\s*[0-9]+){2,})\s*(?:秒)?\s*(?:/|$)`)
	v6kResolutionNote      = regexp.MustCompile(`(?i)([0-9]{3,4}p|[124]k)`)
	videoSlashDurationList = regexp.MustCompile(`([0-9]+(?:/[0-9]+)+)\s*(?:秒|[sS]\b)`)
	suanliaiReferenceTuple = regexp.MustCompile(`([0-9]+)-([0-9]+)-([0-9]+)(?:[/)）]|$)`)
	publicCatalogVideoName = regexp.MustCompile(`(?i)(sd-?2(?:\.|[^0-9]|$)|seedance|wan3|w3-(?:c|p)|mx-h3|video|sora|veo|海螺|可灵)`)
	publicCatalogImageName = regexp.MustCompile(`(?i)(image|banana|gpt-|gemini|claude|deepseek)`)
	publicCatalogRatios    = regexp.MustCompile(`[0-9]+:[0-9]+`)
)

func isPublicCatalogVideo(name string) bool {
	return !publicCatalogImageName.MatchString(name) && publicCatalogVideoName.MatchString(name)
}

func supplementSuanliaiReferenceTuple(p *common.ModelIntegrationParameters, name string) {
	// Equal audio/video counts are unambiguous in the supplier's published
	// 30-10-10 and 9-3-3 notation. Unequal triples remain unconfirmed.
	for _, m := range suanliaiReferenceTuple.FindAllStringSubmatch(name, -1) {
		a, e1 := strconv.Atoi(m[1])
		b, e2 := strconv.Atoi(m[2])
		c, e3 := strconv.Atoi(m[3])
		if e1 != nil || e2 != nil || e3 != nil || a > 1000 || b > 1000 || b != c {
			continue
		}
		for _, entry := range []struct {
			name  string
			value int
			dest  *int
		}{{"max_images", a, &p.MaxImages}, {"max_videos", b, &p.MaxVideos}, {"max_audios", c, &p.MaxAudios}} {
			if !common.IsIntegrationCapabilityUnknown(p, entry.name) {
				continue
			}
			*entry.dest = entry.value
			unknown := make([]string, 0, len(p.UnknownFields))
			for _, field := range p.UnknownFields {
				if field != entry.name {
					unknown = append(unknown, field)
				}
			}
			p.UnknownFields = unknown
		}
		return
	}
}

// Parse only the supplier's explicit capability notation. Omitted media types,
// unrecognized prose, prices and success percentages do not imply limits.
func parseV6KCapabilityNote(note string) *common.ModelIntegrationParameters {
	p := unknownVideoParameters(constant.ChannelTypeV6K)
	p.CapabilitiesUnknown = false
	known := make(map[string]bool)
	for _, match := range v6kReferenceNote.FindAllStringSubmatch(note, -1) {
		count, err := strconv.Atoi(match[1])
		if err != nil || count > 1000 {
			continue
		}
		switch match[2] {
		case "图":
			p.MaxImages = count
			known["max_images"] = true
		case "音":
			p.MaxAudios = count
			known["max_audios"] = true
		case "视":
			p.MaxVideos = count
			known["max_videos"] = true
		}
	}
	for _, field := range []struct {
		name, unsupported string
		dest              *int
	}{
		{"max_images", "不支持图片参考", &p.MaxImages}, {"max_videos", "不支持视频参考", &p.MaxVideos}, {"max_audios", "不支持音频参考", &p.MaxAudios},
	} {
		if strings.Contains(note, field.unsupported) {
			*field.dest = 0
			known[field.name] = true
		}
		if !known[field.name] {
			p.UnknownFields = append(p.UnknownFields, field.name)
		}
	}
	if m := v6kFixedDurationNote.FindStringSubmatch(note); len(m) > 0 {
		n, _ := strconv.Atoi(m[1])
		if n > 0 && n <= 600 {
			p.Durations = []int{n}
		}
	} else if m := videoSlashDurationList.FindStringSubmatch(note); len(m) > 0 {
		for _, value := range strings.Split(m[1], "/") {
			n, _ := strconv.Atoi(value)
			if n > 0 && n <= 600 {
				p.Durations = append(p.Durations, n)
			}
		}
	} else if m := v6kDurationListNote.FindStringSubmatch(note); len(m) > 0 {
		for _, value := range strings.Split(m[1], "-") {
			n, _ := strconv.Atoi(strings.TrimSpace(value))
			if n > 0 && n <= 600 {
				p.Durations = append(p.Durations, n)
			}
		}
	} else if m := v6kDurationRangeNote.FindStringSubmatch(note); len(m) > 0 {
		lo, _ := strconv.Atoi(m[1])
		hi, _ := strconv.Atoi(m[2])
		if lo > 0 && hi >= lo && hi <= 600 {
			for n := lo; n <= hi; n++ {
				p.Durations = append(p.Durations, n)
			}
		}
	}
	seen := make(map[string]bool)
	// Lookahead is unavailable in Go regex; scan slash-separated segments so
	// adjacent resolution entries can both be matched.
	for _, segment := range strings.Split(note, "/") {
		for _, m := range v6kResolutionNote.FindAllStringSubmatch(segment, -1) {
			value := strings.ToLower(m[1])
			if !seen[value] {
				p.Resolutions = append(p.Resolutions, value)
				seen[value] = true
			}
		}
	}
	for _, ratio := range publicCatalogRatios.FindAllString(note, -1) {
		if common.StringsContains(common.DefaultVideoAspectRatios(), ratio) && !common.StringsContains(p.Ratios, ratio) {
			p.Ratios = append(p.Ratios, ratio)
		}
	}
	if strings.Contains(note, "只能竖屏") {
		p.Ratios = []string{"9:16"}
	}
	return p
}

var capabilityQueue = struct {
	sync.Mutex
	pending map[int]bool
}{pending: make(map[int]bool)}
var capabilityWorkers = make(chan struct{}, 2)

// Coalesce repeated saves while always rerunning for an edit made mid-fetch.
func QueueChannelCapabilitySync(channelID int) {
	capabilityQueue.Lock()
	_, exists := capabilityQueue.pending[channelID]
	capabilityQueue.pending[channelID] = true
	capabilityQueue.Unlock()
	if exists {
		return
	}
	go func() {
		capabilityWorkers <- struct{}{}
		defer func() { <-capabilityWorkers }()
		for {
			capabilityQueue.Lock()
			capabilityQueue.pending[channelID] = false
			capabilityQueue.Unlock()
			_, err := RefreshChannelCapabilities(context.Background(), channelID)
			if err != nil {
				common.SysLog(fmt.Sprintf("channel capability sync %d: %s", channelID, capabilitySyncErrorCode(err)))
			}
			capabilityQueue.Lock()
			if !capabilityQueue.pending[channelID] {
				delete(capabilityQueue.pending, channelID)
				capabilityQueue.Unlock()
				return
			}
			capabilityQueue.Unlock()
		}
	}()
}

func capabilitySyncErrorCode(err error) string {
	// Never log HTTP client errors containing an upstream URL or credential.
	if err.Error() == "channel_changed_during_sync" {
		return "channel_changed_during_sync"
	}
	return "capability_sync_failed"
}

var capabilityTaskOnce sync.Once

func StartChannelCapabilitySyncTask() {
	if !common.IsMasterNode {
		return
	}
	capabilityTaskOnce.Do(func() {
		go func() {
			interval := common.GetEnvOrDefault("CHANNEL_CAPABILITY_SYNC_INTERVAL_MINUTES", 30)
			if interval <= 0 {
				return
			}
			ticker := time.NewTicker(time.Duration(interval) * time.Minute)
			defer ticker.Stop()
			for {
				var channels []model.Channel
				if err := model.DB.Select("id", "type", "base_url").Where("status = ?", 1).Find(&channels).Error; err == nil {
					for i := range channels {
						if SupportsChannelCapabilitySync(channels[i].Type, channels[i].GetBaseURL()) {
							QueueChannelCapabilitySync(channels[i].Id)
						}
					}
				}
				<-ticker.C
			}
		}()
	})
}
