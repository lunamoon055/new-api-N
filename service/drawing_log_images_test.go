package service

import (
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/require"
)

func TestPrepareDrawingLogImages(t *testing.T) {
	storage, err := url.Parse("https://media.example/upload")
	require.NoError(t, err)
	const upstream = "https://upstream.example/generated.png"
	const stored = "https://media.example/file/generated.png"
	const local = "/api/image-results/generated.png"
	const proxy = "/mj/image/task-1"
	for _, tt := range []struct {
		name       string
		action     string
		primary    string
		properties string
		proxyURL   string
		want       []string
	}{
		{
			name: "legacy row with upstream and transferred copies", primary: upstream,
			properties: `{"image_url":"` + upstream + `","image_urls":["` + stored + `","` + stored + `"]}`,
			want:       []string{stored},
		},
		{
			name: "local preview replaces upstream fallback", primary: local,
			properties: `{"image_urls":["` + upstream + `","` + local + `"]}`,
			want:       []string{local},
		},
		{
			name: "no transferred copy retains upstream fallback", primary: upstream,
			properties: `{"image_url":"` + upstream + `"}`,
			want:       []string{upstream},
		},
		{
			name: "unrelated host with similar name is upstream", primary: upstream,
			properties: `{"image_urls":["https://media.example.invalid/result.png"]}`,
			want:       []string{upstream, "https://media.example.invalid/result.png"},
		},
		{
			name: "new record with separate transferred and fallback results", primary: stored,
			action:     constant.MjActionImageGeneration,
			properties: `{"image_urls_resolved":true,"image_urls":["` + stored + `","` + upstream + `"]}`,
			want:       []string{stored, upstream},
		},
		{
			name: "Midjourney proxy remains usable", primary: upstream,
			action:     constant.MjActionImagine,
			properties: `{"image_url":"` + upstream + `"}`,
			proxyURL:   proxy,
			want:       []string{proxy},
		},
		{
			name: "no phantom proxy without a result", action: constant.MjActionImagine,
			properties: `{"image_url":""}`,
			proxyURL:   proxy,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			task := &model.Midjourney{Action: tt.action, ImageUrl: tt.primary, Properties: tt.properties}
			PrepareDrawingLogImages(task, []*url.URL{storage}, tt.proxyURL)
			var properties struct {
				ImageURL  string   `json:"image_url"`
				ImageURLs []string `json:"image_urls"`
			}
			require.NoError(t, common.UnmarshalJsonStr(task.Properties, &properties))
			require.Empty(t, properties.ImageURL)
			require.Equal(t, tt.want, properties.ImageURLs)
			if len(tt.want) == 0 {
				require.Empty(t, task.ImageUrl)
			} else {
				require.Equal(t, tt.want[0], task.ImageUrl)
			}
		})
	}
}
