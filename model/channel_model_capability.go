package model

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChannelModelCapability is documentation metadata, not relay authorization.
// Credentials are included only in the irreversible scope digest, never stored
// in this table or published in a catalog response.
type ChannelModelCapability struct {
	ID               string `gorm:"type:varchar(64);primaryKey"`
	ChannelID        int    `gorm:"index;not null"`
	Scope            string `gorm:"type:varchar(64);not null"`
	ModelName        string `gorm:"type:text;not null"`
	PublicModel      string `gorm:"type:text;not null"`
	EndpointTemplate string `gorm:"type:text;not null"`
	Parameters       string `gorm:"type:text;not null"`
	Source           string `gorm:"type:varchar(32);not null"`
	UpdatedAt        int64  `gorm:"autoUpdateTime:false"`
}

func ChannelCapabilityScope(channel *Channel) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d\n%s\n%s\n%s\n%s",
		channel.Type, strings.TrimRight(channel.GetBaseURL(), "/"), channel.Key,
		channel.GetSetting().Proxy, channel.ChannelInfoJSONForCapabilities()))))
}

// Include enabled-key changes in the scope without advancing a key's polling state.
func (channel *Channel) ChannelInfoJSONForCapabilities() string {
	value, _ := common.Marshal(channel.ChannelInfo.MultiKeyStatusList)
	return string(value)
}

func NewChannelModelCapability(channel *Channel, publicModel, modelName string, parameters *common.ModelIntegrationParameters, source string) (ChannelModelCapability, error) {
	scope := ChannelCapabilityScope(channel)
	encoded, err := common.Marshal(parameters)
	if err != nil {
		return ChannelModelCapability{}, err
	}
	var metadata []Model
	if err := DB.Where("model_name IN ?", []string{publicModel, modelName}).Find(&metadata).Error; err != nil {
		return ChannelModelCapability{}, err
	}
	metaMap := make(map[string]*Model)
	for i := range metadata {
		metaMap[metadata[i].ModelName] = &metadata[i]
	}
	endpoints := getPricingIntegrationEndpointsForModels(channel.Type, modelName, publicModel, metaMap)
	var template *PricingMappedEndpoint
	for i := range endpoints {
		if endpoints[i].Kind == "video" && endpoints[i].Path == "/v1/videos" {
			template = &endpoints[i]
			break
		}
	}
	if template == nil && !common.IsOpenAITextModel(modelName) && !common.IsImageGenerationModel(modelName) {
		var previous ChannelModelCapability
		if DB.Where("channel_id = ? AND scope = ? AND public_model = ?", channel.Id, scope, publicModel).First(&previous).Error == nil {
			var endpoint *PricingMappedEndpoint
			if common.UnmarshalJsonStr(previous.EndpointTemplate, &endpoint) == nil && endpoint != nil && endpoint.Kind == "video" && endpoint.Path == "/v1/videos" {
				template = endpoint
			}
		}
	}
	templateJSON, err := common.Marshal(template)
	if err != nil {
		return ChannelModelCapability{}, err
	}
	return ChannelModelCapability{
		ID:        fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d\n%s\n%s", channel.Id, scope, publicModel)))),
		ChannelID: channel.Id, Scope: scope, ModelName: modelName,
		PublicModel: publicModel, EndpointTemplate: string(templateJSON),
		Parameters: string(encoded), Source: source, UpdatedAt: common.GetTimestamp(),
	}, nil
}

// ReplaceChannelModelCapabilities commits one complete catalog atomically. An
// unavailable upstream never calls this function, retaining the last good data.
func ReplaceChannelModelCapabilities(channelID int, items []ChannelModelCapability) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", channelID).Delete(&ChannelModelCapability{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(items, 100).Error
	})
	if err == nil {
		InvalidatePricingCache()
	}
	return err
}

// A single bulk read is shared by every public catalog consumer. An old scope
// (different origin, credential or proxy) can never supply a new channel's hints.
func pricingChannelCapabilities() map[int]map[string]ChannelModelCapability {
	result := make(map[int]map[string]ChannelModelCapability)
	if !DB.Migrator().HasTable(&ChannelModelCapability{}) {
		return result
	}
	var rows []ChannelModelCapability
	if err := DB.Find(&rows).Error; err != nil {
		return result
	}
	if len(rows) == 0 {
		return result
	}
	ids := make([]int, 0)
	seen := make(map[int]bool)
	for _, row := range rows {
		if !seen[row.ChannelID] {
			ids = append(ids, row.ChannelID)
			seen[row.ChannelID] = true
		}
	}
	var channels []Channel
	if err := DB.Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return result
	}
	scopes := make(map[int]string, len(channels))
	for i := range channels {
		scopes[channels[i].Id] = ChannelCapabilityScope(&channels[i])
	}
	for _, row := range rows {
		if scopes[row.ChannelID] != row.Scope {
			continue
		}
		if result[row.ChannelID] == nil {
			result[row.ChannelID] = make(map[string]ChannelModelCapability)
		}
		result[row.ChannelID][row.PublicModel] = row
	}
	return result
}

func applyPricingChannelCapability(endpoints []PricingMappedEndpoint, item ChannelModelCapability, upstreamModel string) []PricingMappedEndpoint {
	var parameters common.ModelIntegrationParameters
	if common.UnmarshalJsonStr(item.Parameters, &parameters) != nil {
		return endpoints
	}
	if item.ModelName != upstreamModel {
		parameters = common.ModelIntegrationParameters{CapabilitiesUnknown: true}
	}
	var template *PricingMappedEndpoint
	_ = common.UnmarshalJsonStr(item.EndpointTemplate, &template)
	if template != nil && template.Kind == "video" && template.Path == "/v1/videos" && !common.IsOpenAITextModel(upstreamModel) && !common.IsImageGenerationModel(upstreamModel) {
		if len(endpoints) == 0 || (len(endpoints) == 1 && endpoints[0].Path == "/v1/chat/completions") {
			endpoints = []PricingMappedEndpoint{*template}
		}
	}
	for i := range endpoints {
		ep := &endpoints[i]
		if ep.Kind != "video" || ep.Path != "/v1/videos" {
			continue
		}
		// Reuse implemented field names across renames, never old limits.
		if ep.Parameters != nil {
			parameters.DurationField, parameters.RatioField, parameters.ResolutionField = ep.Parameters.DurationField, ep.Parameters.RatioField, ep.Parameters.ResolutionField
			parameters.ImageField, parameters.VideoField, parameters.AudioField = ep.Parameters.ImageField, ep.Parameters.VideoField, ep.Parameters.AudioField
		}
		if len(parameters.Ratios) == 0 && item.ModelName == upstreamModel {
			if exact := common.GetVideoIntegrationParameters(upstreamModel); exact != nil {
				parameters.Ratios = exact.Ratios
			}
		}
		if exact := common.GetVideoIntegrationParameters(upstreamModel); exact != nil && item.ModelName == upstreamModel {
			parameters.AllowFrames, parameters.MinImages = exact.AllowFrames, exact.MinImages
			parameters.AudioNeedsImage, parameters.AudioRequiresImage = exact.AudioNeedsImage, exact.AudioRequiresImage
		}
		ep.Parameters = common.WithVideoRatioDefaults(&parameters)
		if !slices.Contains(parameters.Durations, ep.ExampleDuration) {
			ep.ExampleDuration = 0
		}
		if !common.StringsContains(parameters.Resolutions, ep.ExampleResolution) {
			ep.ExampleResolution = ""
		}
		ep.ExampleImageRequired = parameters.MinImages > 0
		doc := *ep.Parameters
		doc.DurationField, doc.RatioField, doc.ResolutionField = "duration", "ratio", "resolution"
		doc.ImageField, doc.VideoField, doc.AudioField = "referenceImages", "referenceVideos", "referenceAudios"
		ep.DocumentationFields = &doc
		ep.CapabilitySource, ep.CapabilitiesUpdatedAt = item.Source, item.UpdatedAt
		if item.ModelName != upstreamModel {
			ep.CapabilitySource = "protocol_template"
			ep.CapabilitiesUpdatedAt = 0
		}
	}
	return endpoints
}
