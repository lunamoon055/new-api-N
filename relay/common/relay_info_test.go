package common

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestTaskSubmitReqInputMaterialURLsGroupsAndDeduplicatesLinks(t *testing.T) {
	req := TaskSubmitReq{
		Image:          "https://cdn.example.com/image.png",
		Images:         []string{"https://cdn.example.com/image.png", "/api/creation/reference-files/local-image"},
		ImageURL:       "data:image/png;base64,AAAA",
		StartImageURL:  "https://cdn.example.com/start.png",
		VideoURL:       "https://cdn.example.com/video.mp4",
		Videos:         []string{"https://cdn.example.com/video.mp4"},
		VideoReference: []TaskReference{{URL: "https://cdn.example.com/reference.mp4"}},
		AudioURL:       "https://cdn.example.com/audio.wav",
		ReferenceAudios: []string{
			"javascript:alert(1)",
			"https://cdn.example.com/audio.wav",
		},
	}

	images, videos, audios := req.InputMaterialURLs()

	require.Equal(t, []string{
		"https://cdn.example.com/image.png",
		"https://cdn.example.com/start.png",
		"/api/creation/reference-files/local-image",
	}, images)
	require.Equal(t, []string{
		"https://cdn.example.com/video.mp4",
		"https://cdn.example.com/reference.mp4",
	}, videos)
	require.Equal(t, []string{"https://cdn.example.com/audio.wav"}, audios)
}

func TestTaskReferenceAcceptsStringAndObjectLinks(t *testing.T) {
	var direct TaskReference
	require.NoError(t, direct.UnmarshalJSON([]byte(`"https://cdn.example.com/direct.mp4"`)))
	require.Equal(t, "https://cdn.example.com/direct.mp4", direct.URL)

	var object TaskReference
	require.NoError(t, object.UnmarshalJSON([]byte(`{"url":"https://cdn.example.com/object.mp4"}`)))
	require.Equal(t, "https://cdn.example.com/object.mp4", object.URL)
}
