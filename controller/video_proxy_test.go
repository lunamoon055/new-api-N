package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestIsAsyncGenerationsVideoTaskIncludesLinkskyModels(t *testing.T) {
	for _, modelName := range []string{"sora2", "kling-v3", "video-2.0", "video-2.0-fast", "ko3"} {
		t.Run(modelName, func(t *testing.T) {
			task := &model.Task{
				Properties: model.Properties{
					OriginModelName: modelName,
				},
			}

			require.True(t, isAsyncGenerationsVideoTask(task))
		})
	}
}
