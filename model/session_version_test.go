package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func createSessionVersionTestUser(t *testing.T, id int) *User {
	t.Helper()

	user := &User{
		Id:             id,
		Username:       fmt.Sprintf("session_version_%d", id),
		Password:       "existing-hash",
		Role:           common.RoleCommonUser,
		Status:         common.UserStatusEnabled,
		Email:          fmt.Sprintf("session-version-%d@example.com", id),
		Group:          "default",
		SessionVersion: 1,
	}
	require.NoError(t, DB.Create(user).Error)
	t.Cleanup(func() {
		DB.Unscoped().Delete(&User{}, id)
	})
	return user
}

func requireSessionVersion(t *testing.T, userID int, expected int64) {
	t.Helper()

	state, err := GetUserAuthState(userID)
	require.NoError(t, err)
	require.Equal(t, expected, state.SessionVersion)
}

func TestUserSecurityChangesBumpSessionVersion(t *testing.T) {
	t.Run("password update", func(t *testing.T) {
		user := createSessionVersionTestUser(t, 8101)
		user.Password = "new-password"
		require.NoError(t, user.Update(true))
		requireSessionVersion(t, user.Id, 2)
	})

	t.Run("status update", func(t *testing.T) {
		user := createSessionVersionTestUser(t, 8102)
		user.Status = common.UserStatusDisabled
		require.NoError(t, user.Update(false))
		requireSessionVersion(t, user.Id, 2)
	})

	t.Run("role update", func(t *testing.T) {
		user := createSessionVersionTestUser(t, 8103)
		user.Role = common.RoleAdminUser
		require.NoError(t, user.Update(false))
		requireSessionVersion(t, user.Id, 2)
	})

	t.Run("benign update", func(t *testing.T) {
		user := createSessionVersionTestUser(t, 8104)
		user.DisplayName = "new display name"
		require.NoError(t, user.Update(false))
		requireSessionVersion(t, user.Id, 1)
	})

	t.Run("admin edit", func(t *testing.T) {
		user := createSessionVersionTestUser(t, 8105)
		user.Status = common.UserStatusDisabled
		require.NoError(t, user.Edit(false))
		requireSessionVersion(t, user.Id, 2)
	})
}

func TestPasswordResetBumpsSessionVersion(t *testing.T) {
	user := createSessionVersionTestUser(t, 8110)

	require.NoError(t, ResetUserPasswordByEmail(user.Email, "reset-password"))
	requireSessionVersion(t, user.Id, 2)
}

func TestTwoFAStateChangesBumpSessionVersion(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&TwoFA{}, &TwoFABackupCode{}))
	user := createSessionVersionTestUser(t, 8120)
	twoFA := &TwoFA{
		UserId:    user.Id,
		Secret:    "test-secret",
		IsEnabled: false,
	}
	require.NoError(t, DB.Create(twoFA).Error)

	require.NoError(t, twoFA.Enable())
	requireSessionVersion(t, user.Id, 2)

	require.NoError(t, CreateBackupCodes(user.Id, []string{"backup-code"}))
	requireSessionVersion(t, user.Id, 3)

	require.NoError(t, DisableTwoFA(user.Id))
	requireSessionVersion(t, user.Id, 4)
}

func TestPasskeySecurityMaterialChangesBumpSessionVersion(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&PasskeyCredential{}))
	user := createSessionVersionTestUser(t, 8130)

	credential := &PasskeyCredential{
		UserID:       user.Id,
		CredentialID: "credential-a",
		PublicKey:    "public-key-a",
	}
	require.NoError(t, UpsertPasskeyCredential(credential))
	requireSessionVersion(t, user.Id, 2)

	// Assertion metadata changes during login must not revoke every session.
	credential.SignCount++
	require.NoError(t, UpsertPasskeyCredential(credential))
	requireSessionVersion(t, user.Id, 2)

	credential.CredentialID = "credential-b"
	credential.PublicKey = "public-key-b"
	require.NoError(t, UpsertPasskeyCredential(credential))
	requireSessionVersion(t, user.Id, 3)

	require.NoError(t, DeletePasskeyByUserID(user.Id))
	requireSessionVersion(t, user.Id, 4)
}
