package common

// ModelIntegrationEndpoint is the public protocol after channel mapping.
// It deliberately excludes channel identities and private upstream model IDs.
type ModelIntegrationEndpoint struct {
	Type                 string                      `json:"type"`
	Kind                 string                      `json:"kind"`
	Path                 string                      `json:"path"`
	Method               string                      `json:"method"`
	QueryPath            string                      `json:"query_path,omitempty"`
	QueryMethod          string                      `json:"query_method,omitempty"`
	ExampleDuration      int                         `json:"example_duration,omitempty"`
	ExampleResolution    string                      `json:"example_resolution,omitempty"`
	ExampleImageRequired bool                        `json:"example_image_required,omitempty"`
	Parameters           *ModelIntegrationParameters `json:"parameters,omitempty"`
}
