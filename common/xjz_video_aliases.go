package common

// These are complete public IDs currently published by XJZ. Only these exact
// IDs inherit an existing contract; arbitrary suffixes are never stripped.
var XJZVideoAliases = map[string]string{
	"sd2.5-全参过真人-30-10-10-B":     "sd2.5-全参过真人-30-10-10",
	"sd2.0-全参过真人-720p-B":         "sd2.0-全参过真人-720p",
	"grok-1.5-700-B":             "grok-1.5-700",
	"dola-sd-2.5-B":              "dola-sd-2.5",
	"dola-sd-2.0-900-B":          "dola-sd-2.0-900",
	"dola-sd-2.0-B":              "dola-sd-2.0",
	"sd-2.0-933-720-fast-原生真人-B": "sd-2.0-933-720-fast-原生真人",
	"gz稳定sd2.0-903-720P-C":       "gz稳定sd2.0-903-720P",
	"特价sd2.0-720p-933-过脸-C":      "特价sd2.0-720p-933-过脸",
	"特价sd2.0-933-过脸-D":           "特价sd2.0-933-过脸",
}

func init() {
	for alias, target := range XJZVideoAliases {
		XJZVideoSpecs[alias] = XJZVideoSpecs[target]
	}
}
