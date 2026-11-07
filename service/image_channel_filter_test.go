package service

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStandardImageOptionsSelectACompatibleMappedChannel(t *testing.T) {
	mega := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://newapi.megabyai.cc"), ModelMapping: common.GetPointer(`{"alias":"gpt-image-2.5-flare"}`)}
	xjz := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://api.xjzapi.cc"), ModelMapping: common.GetPointer(`{"alias":"gpt-image-2.5-flare-原生4K"}`)}
	for _, tc := range []struct {
		body   string
		native bool
	}{
		{`{"model":"alias","n":1,"resolution":"4K","ratio":"16:9"}`, true},
		{`{"model":"alias","size":"1536x864"}`, false},
		{`{"model":"alias","n":2}`, false},
		{`{"model":"alias","background":"transparent"}`, false},
		{`{"model":"alias","output_compression":0}`, false},
		{`{"model":"alias","quality":"max"}`, false},
		{`{"model":"alias","quality":"standard"}`, true},
		{`{"model":"alias","ratio":"5:4"}`, false},
		{`{"model":"alias","mask":{"image_url":"https://example.com/mask.png"}}`, false},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(tc.body))
		c.Request.Header.Set("Content-Type", "application/json")
		filter := SynchronousImageChannelFilter(c, "alias")
		require.True(t, filter(mega), tc.body)
		require.Equal(t, tc.native, filter(xjz), tc.body)
	}
}

func TestJSONEditsAndLargeImageFilesSelectTheStandardTransport(t *testing.T) {
	mega := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://newapi.megabyai.cc"), ModelMapping: common.GetPointer(`{"alias":"gpt-image-2.5-flare"}`)}
	xjz := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://api.xjzapi.cc"), ModelMapping: common.GetPointer(`{"alias":"gpt-image-2.5-flare-原生4K"}`)}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/edits", strings.NewReader(`{"model":"alias","images":[{"image_url":"https://example.com/reference.png"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	filter := SynchronousImageChannelFilter(c, "alias")
	require.True(t, filter(mega))
	require.False(t, filter(xjz))
	for _, size := range []int64{20 << 20, 30 << 20, 51 << 20} {
		c.Request = httptest.NewRequest("POST", "/v1/images/edits", nil)
		c.Request.MultipartForm = &multipart.Form{Value: map[string][]string{"model": {"alias"}}, File: map[string][]*multipart.FileHeader{"image[]": {{Filename: "ref.png", Size: size}}}}
		filter = SynchronousImageChannelFilter(c, "alias")
		require.Equal(t, size <= 50<<20, filter(mega))
		require.Equal(t, size <= 20<<20, filter(xjz))
	}
}

func TestImageReferenceLimitsAreSelectedPerChannelAndCombinedOptions(t *testing.T) {
	mega := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://newapi.megabyai.cc"), ModelMapping: common.GetPointer(`{"alias":"gpt-image-2.5-flare"}`)}
	xjz := &model.Channel{Type: constant.ChannelTypeOpenAI, BaseURL: common.GetPointer("https://api.xjzapi.cc"), ModelMapping: common.GetPointer(`{"alias":"gpt-image-2.5-flare-原生4K"}`)}
	for _, tc := range []struct {
		count        int
		n            string
		mega, native bool
	}{{16, "1", true, true}, {17, "1", false, true}, {20, "1", false, true}, {21, "1", false, false}, {17, "2", false, false}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/images/edits", nil)
		files := make([]*multipart.FileHeader, tc.count)
		for i := range files {
			files[i] = &multipart.FileHeader{Filename: fmt.Sprint(i) + ".png", Size: 100}
		}
		c.Request.MultipartForm = &multipart.Form{Value: map[string][]string{"model": {"alias"}, "n": {tc.n}}, File: map[string][]*multipart.FileHeader{"image[]": files}}
		filter := SynchronousImageChannelFilter(c, "alias")
		require.Equal(t, tc.mega, filter(mega))
		require.Equal(t, tc.native, filter(xjz))
	}
}
