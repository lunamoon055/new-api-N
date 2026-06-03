package dto

import "github.com/QuantumNous/new-api/constant"

type CreationModel struct {
	ID                     string                  `json:"id"`
	Description            string                  `json:"description,omitempty"`
	Icon                   string                  `json:"icon,omitempty"`
	Tags                   []string                `json:"tags,omitempty"`
	VendorID               int                     `json:"vendor_id,omitempty"`
	SupportedEndpointTypes []constant.EndpointType `json:"supported_endpoint_types"`
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
