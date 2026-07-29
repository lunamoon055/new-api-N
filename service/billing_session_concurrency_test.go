package service

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newForcedWalletRelayInfo(userId, tokenId int, tokenKey string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:          userId,
		TokenId:         tokenId,
		TokenKey:        tokenKey,
		ForcePreConsume: true,
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
	}
}

func newBillingTestContext() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	return ctx
}

func TestPreConsumeBilling_ConcurrentWalletReservationDoesNotOverdraw(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9001, 9001
	const tokenKey = "sk-concurrent-wallet-reservation"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 100)

	// Atomic pre-consumption must bypass the process-local batch updater.
	common.BatchUpdateEnabled = true
	t.Cleanup(func() { common.BatchUpdateEnabled = false })

	start := make(chan struct{})
	results := make(chan *types.NewAPIError, 2)
	for i := 0; i < 2; i++ {
		ctx := newBillingTestContext()
		relayInfo := newForcedWalletRelayInfo(userId, tokenId, tokenKey)
		go func() {
			<-start
			results <- PreConsumeBilling(ctx, 80, relayInfo)
		}()
	}
	close(start)

	successes := 0
	userQuotaFailures := 0
	for i := 0; i < 2; i++ {
		apiErr := <-results
		if apiErr == nil {
			successes++
			continue
		}
		if apiErr.GetErrorCode() == types.ErrorCodeInsufficientUserQuota {
			userQuotaFailures++
		}
	}

	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, userQuotaFailures)
	assert.Equal(t, 20, getUserQuota(t, userId))
	assert.Equal(t, 20, getTokenRemainQuota(t, tokenId))
	assert.Equal(t, 80, getTokenUsedQuota(t, tokenId))
}

func TestPreConsumeTokenQuota_ConcurrentReservationDoesNotOverdraw(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9007, 9007
	const tokenKey = "sk-concurrent-token-reservation"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 100)

	common.BatchUpdateEnabled = true
	t.Cleanup(func() { common.BatchUpdateEnabled = false })

	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		relayInfo := newForcedWalletRelayInfo(userId, tokenId, tokenKey)
		go func() {
			<-start
			results <- PreConsumeTokenQuota(relayInfo, 80)
		}()
	}
	close(start)

	successes := 0
	tokenQuotaFailures := 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, model.ErrInsufficientTokenQuota) {
			tokenQuotaFailures++
		}
	}

	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, tokenQuotaFailures)
	assert.Equal(t, 20, getTokenRemainQuota(t, tokenId))
	assert.Equal(t, 80, getTokenUsedQuota(t, tokenId))
}

func TestPreConsumeTokenQuota_UnlimitedTokenStillTracksUsage(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9008, 9008
	const tokenKey = "sk-unlimited-token-reservation"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 0)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenId).Update("unlimited_quota", true).Error)

	relayInfo := newForcedWalletRelayInfo(userId, tokenId, tokenKey)
	relayInfo.TokenUnlimited = true
	require.NoError(t, PreConsumeTokenQuota(relayInfo, 80))
	assert.Equal(t, -80, getTokenRemainQuota(t, tokenId))
	assert.Equal(t, 80, getTokenUsedQuota(t, tokenId))
}

func TestPreConsumeBilling_TokenFailureRollsBackWallet(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9002, 9002
	const tokenKey = "sk-token-reservation-rollback"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 50)

	apiErr := PreConsumeBilling(
		newBillingTestContext(),
		80,
		newForcedWalletRelayInfo(userId, tokenId, tokenKey),
	)
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodePreConsumeTokenQuotaFailed, apiErr.GetErrorCode())
	assert.True(t, errors.Is(apiErr, model.ErrInsufficientTokenQuota))
	assert.Equal(t, 100, getUserQuota(t, userId))
	assert.Equal(t, 50, getTokenRemainQuota(t, tokenId))
	assert.Equal(t, 0, getTokenUsedQuota(t, tokenId))
}

func TestPreConsumeBilling_ReconcilesPendingBatchQuota(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9004, 9004
	const tokenKey = "sk-pending-batch-reservation"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 100)

	common.BatchUpdateEnabled = true
	t.Cleanup(func() { common.BatchUpdateEnabled = false })
	require.NoError(t, model.DecreaseUserQuota(userId, 30, false))
	require.NoError(t, model.DecreaseTokenQuota(tokenId, tokenKey, 30))

	apiErr := PreConsumeBilling(
		newBillingTestContext(),
		80,
		newForcedWalletRelayInfo(userId, tokenId, tokenKey),
	)
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeInsufficientUserQuota, apiErr.GetErrorCode())
	assert.True(t, errors.Is(apiErr, model.ErrInsufficientUserQuota))

	// A smaller reservation atomically persists both the pending batch debit and
	// the new reservation, proving the failed attempt did not drop queued quota.
	apiErr = PreConsumeBilling(
		newBillingTestContext(),
		60,
		newForcedWalletRelayInfo(userId, tokenId, tokenKey),
	)
	require.Nil(t, apiErr)
	assert.Equal(t, 10, getUserQuota(t, userId))
	assert.Equal(t, 10, getTokenRemainQuota(t, tokenId))
	assert.Equal(t, 90, getTokenUsedQuota(t, tokenId))
}

func TestBillingSessionReserve_TokenFailureRollsBackWallet(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9003, 9003
	const tokenKey = "sk-reserve-token-rollback"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 50)

	relayInfo := newForcedWalletRelayInfo(userId, tokenId, tokenKey)
	apiErr := PreConsumeBilling(newBillingTestContext(), 20, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)

	err := relayInfo.Billing.Reserve(60)
	require.Error(t, err)
	var reserveErr *types.NewAPIError
	require.True(t, errors.As(err, &reserveErr))
	assert.Equal(t, types.ErrorCodePreConsumeTokenQuotaFailed, reserveErr.GetErrorCode())
	assert.True(t, errors.Is(err, model.ErrInsufficientTokenQuota))

	assert.Equal(t, 20, relayInfo.Billing.GetPreConsumedQuota())
	assert.Equal(t, 80, getUserQuota(t, userId))
	assert.Equal(t, 30, getTokenRemainQuota(t, tokenId))
	assert.Equal(t, 20, getTokenUsedQuota(t, tokenId))
}

func TestBillingSessionSettle_PersistsDirectlyWithBatchEnabled(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9005, 9005
	const tokenKey = "sk-direct-settlement"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 100)

	common.BatchUpdateEnabled = true
	t.Cleanup(func() { common.BatchUpdateEnabled = false })

	relayInfo := newForcedWalletRelayInfo(userId, tokenId, tokenKey)
	apiErr := PreConsumeBilling(newBillingTestContext(), 20, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	require.NoError(t, relayInfo.Billing.Settle(10))

	assert.Equal(t, 90, getUserQuota(t, userId))
	assert.Equal(t, 90, getTokenRemainQuota(t, tokenId))
	assert.Equal(t, 10, getTokenUsedQuota(t, tokenId))
}

func TestBillingSessionRefund_PersistsDirectlyWithBatchEnabled(t *testing.T) {
	truncate(t)

	const userId, tokenId = 9006, 9006
	const tokenKey = "sk-direct-refund"
	seedUser(t, userId, 100)
	seedToken(t, tokenId, userId, tokenKey, 100)

	common.BatchUpdateEnabled = true
	t.Cleanup(func() { common.BatchUpdateEnabled = false })

	relayInfo := newForcedWalletRelayInfo(userId, tokenId, tokenKey)
	ctx := newBillingTestContext()
	apiErr := PreConsumeBilling(ctx, 20, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	relayInfo.Billing.Refund(ctx)

	require.Eventually(t, func() bool {
		var user model.User
		var token model.Token
		if err := model.DB.Select("quota").Where("id = ?", userId).First(&user).Error; err != nil {
			return false
		}
		if err := model.DB.Select("remain_quota", "used_quota").Where("id = ?", tokenId).First(&token).Error; err != nil {
			return false
		}
		return user.Quota == 100 && token.RemainQuota == 100 && token.UsedQuota == 0
	}, time.Second, 10*time.Millisecond)
}
