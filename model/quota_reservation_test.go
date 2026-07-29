package model

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func clearQuotaBatchEntry(type_, id int) {
	batchUpdateExecutionLock.Lock()
	defer batchUpdateExecutionLock.Unlock()
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	delete(batchUpdateStores[type_], id)
}

func pendingQuotaBatchValue(type_, id int) int {
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	return batchUpdateStores[type_][id]
}

func TestAdjustWalletQuotaReconcilesPendingBatchOnce(t *testing.T) {
	truncateTables(t)
	const userId, tokenId = 9301, 9301
	const tokenKey = "sk-adjust-wallet-pending"
	require.NoError(t, DB.Create(&User{Id: userId, Username: "adjust_pending", Quota: 100}).Error)
	require.NoError(t, DB.Create(&Token{Id: tokenId, UserId: userId, Key: tokenKey, RemainQuota: 100}).Error)

	common.BatchUpdateEnabled = true
	t.Cleanup(func() {
		common.BatchUpdateEnabled = false
		clearQuotaBatchEntry(BatchUpdateTypeUserQuota, userId)
		clearQuotaBatchEntry(BatchUpdateTypeTokenQuota, tokenId)
	})
	require.NoError(t, DecreaseUserQuota(userId, 30, false))
	require.NoError(t, DecreaseTokenQuota(tokenId, tokenKey, 30))

	// Refund 10 while 30 of consumption is still queued: net persisted delta is -20.
	require.NoError(t, AdjustWalletQuota(userId, tokenId, tokenKey, -10, true))

	var user User
	var token Token
	require.NoError(t, DB.Select("quota").First(&user, userId).Error)
	require.NoError(t, DB.Select("remain_quota", "used_quota").First(&token, tokenId).Error)
	assert.Equal(t, 80, user.Quota)
	assert.Equal(t, 80, token.RemainQuota)
	assert.Equal(t, 20, token.UsedQuota)
	assert.Zero(t, pendingQuotaBatchValue(BatchUpdateTypeUserQuota, userId))
	assert.Zero(t, pendingQuotaBatchValue(BatchUpdateTypeTokenQuota, tokenId))

	batchUpdate()
	require.NoError(t, DB.Select("quota").First(&user, userId).Error)
	require.NoError(t, DB.Select("remain_quota", "used_quota").First(&token, tokenId).Error)
	assert.Equal(t, 80, user.Quota)
	assert.Equal(t, 80, token.RemainQuota)
	assert.Equal(t, 20, token.UsedQuota)
}

func TestAdjustWalletQuotaRollsBackUserWhenTokenUpdateFails(t *testing.T) {
	truncateTables(t)
	const userId, tokenId = 9302, 9302
	const tokenKey = "sk-adjust-wallet-rollback"
	require.NoError(t, DB.Create(&User{Id: userId, Username: "adjust_rollback", Quota: 100}).Error)
	require.NoError(t, DB.Create(&Token{Id: tokenId, UserId: userId, Key: tokenKey, RemainQuota: 100}).Error)

	const callbackName = "test:fail-adjust-wallet-token"
	var failToken atomic.Bool
	failToken.Store(true)
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if failToken.Load() && tx.Statement.Schema != nil && tx.Statement.Schema.Table == "tokens" {
			failToken.Store(false)
			tx.AddError(errors.New("injected token update failure"))
		}
	}))
	callbackRegistered := true
	t.Cleanup(func() {
		if callbackRegistered {
			_ = DB.Callback().Update().Remove(callbackName)
		}
	})

	require.Error(t, AdjustWalletQuota(userId, tokenId, tokenKey, 20, true))

	var user User
	var token Token
	require.NoError(t, DB.Select("quota").First(&user, userId).Error)
	require.NoError(t, DB.Select("remain_quota", "used_quota").First(&token, tokenId).Error)
	assert.Equal(t, 100, user.Quota)
	assert.Equal(t, 100, token.RemainQuota)
	assert.Equal(t, 0, token.UsedQuota)
}

func TestBatchUpdateRequeuesFailedQuotaDelta(t *testing.T) {
	truncateTables(t)
	const userId = 9303
	require.NoError(t, DB.Create(&User{Id: userId, Username: "batch_requeue", Quota: 100}).Error)

	common.BatchUpdateEnabled = true
	t.Cleanup(func() {
		common.BatchUpdateEnabled = false
		clearQuotaBatchEntry(BatchUpdateTypeUserQuota, userId)
	})
	require.NoError(t, DecreaseUserQuota(userId, 30, false))

	const callbackName = "test:fail-batch-user-quota"
	var failUser atomic.Bool
	failUser.Store(true)
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if failUser.Load() && tx.Statement.Schema != nil && tx.Statement.Schema.Table == "users" {
			failUser.Store(false)
			tx.AddError(errors.New("injected batch update failure"))
		}
	}))
	callbackRegistered := true
	t.Cleanup(func() {
		if callbackRegistered {
			_ = DB.Callback().Update().Remove(callbackName)
		}
	})

	batchUpdate()
	assert.Equal(t, -30, pendingQuotaBatchValue(BatchUpdateTypeUserQuota, userId))
	var user User
	require.NoError(t, DB.Select("quota").First(&user, userId).Error)
	assert.Equal(t, 100, user.Quota)

	require.NoError(t, DB.Callback().Update().Remove(callbackName))
	callbackRegistered = false
	batchUpdate()
	assert.Zero(t, pendingQuotaBatchValue(BatchUpdateTypeUserQuota, userId))
	require.NoError(t, DB.Select("quota").First(&user, userId).Error)
	assert.Equal(t, 70, user.Quota)
}
