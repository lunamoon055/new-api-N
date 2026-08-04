package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestCanViewTaskInputMaterialsOnlyAllowsRoot(t *testing.T) {
	require.False(t, canViewTaskInputMaterials(common.RoleCommonUser))
	require.False(t, canViewTaskInputMaterials(common.RoleAdminUser))
	require.True(t, canViewTaskInputMaterials(common.RoleRootUser))
}

func TestTasksToDtoUsesRootOnlyMaterialVisibility(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input:       "prompt remains visible",
			InputImages: []string{"https://cdn.example.com/reference.png"},
		},
	}

	adminDto := tasksToDto(
		[]*model.Task{task},
		false,
		canViewTaskInputMaterials(common.RoleAdminUser),
	)[0]
	adminProperties, ok := adminDto.Properties.(model.Properties)
	require.True(t, ok)
	require.Equal(t, "prompt remains visible", adminProperties.Input)
	require.Empty(t, adminProperties.InputImages)

	rootDto := tasksToDto(
		[]*model.Task{task},
		false,
		canViewTaskInputMaterials(common.RoleRootUser),
	)[0]
	rootProperties, ok := rootDto.Properties.(model.Properties)
	require.True(t, ok)
	require.Equal(t, task.Properties.InputImages, rootProperties.InputImages)
}
