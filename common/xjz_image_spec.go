package common

// These exact mapped IDs publish the v1 media contract. Neighboring routes
// must keep their own contracts, including the older concurrent variants.
func IsXJZNativeImageModel(name string) bool {
	switch name {
	case "gpt-image-2-原生4K", "gpt-image-2.5-flare-原生4K", "gpt-image-2.5-sunburst-原生4K":
		return true
	}
	return false
}

func XJZNativeImageParameters(edit bool) *ModelIntegrationParameters {
	p := &ModelIntegrationParameters{
		Resolutions:     []string{"1K", "2K", "4K"},
		Ratios:          []string{"16:9", "1:1", "21:9", "2:3", "3:2", "3:4", "4:3", "9:16"},
		ResolutionField: "resolution", RatioField: "ratio",
	}
	if edit {
		p.ImageField, p.MinImages, p.MaxImages = "image[]", 1, 20
	}
	return p
}
