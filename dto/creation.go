package dto

import "github.com/QuantumNous/new-api/constant"

type CreationModel struct {
	ID                     string                  `json:"id"`
	Description            string                  `json:"description,omitempty"`
	ManualDescription      string                  `json:"manual_description,omitempty"`
	Icon                   string                  `json:"icon,omitempty"`
	Tags                   []string                `json:"tags,omitempty"`
	VendorID               int                     `json:"vendor_id,omitempty"`
	Cost                   *CreationModelCost      `json:"cost,omitempty"`
	SupportedEndpointTypes []constant.EndpointType `json:"supported_endpoint_types"`
}

type CreationModelCost struct {
	BillingMode           string   `json:"billing_mode"`
	InputPricePerMillion  *float64 `json:"input_price_per_million,omitempty"`
	OutputPricePerMillion *float64 `json:"output_price_per_million,omitempty"`
	RequestPrice          *float64 `json:"request_price,omitempty"`
	RequestQuota          *int     `json:"request_quota,omitempty"`
	GroupRatio            float64  `json:"group_ratio,omitempty"`
}

type CreationModelGroup struct {
	Mode   string          `json:"mode"`
	Models []CreationModel `json:"models"`
}

type CreationVendor struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

type CreationModelCatalog struct {
	Modes   []CreationModelGroup `json:"modes"`
	Vendors []CreationVendor     `json:"vendors"`
}
