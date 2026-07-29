package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// ReserveWalletQuota conditionally reserves wallet quota and, for normal relay
// requests, token quota in the same database transaction. withToken is false
// only for playground requests, which do not consume token quota.
func ReserveWalletQuota(userId, tokenId int, tokenKey string, quota int, withToken bool) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if quota == 0 {
		return nil
	}

	return reserveWalletQuotaWithPending(userId, tokenId, tokenKey, quota, withToken)
}

func reserveWalletQuotaWithPending(userId, tokenId int, tokenKey string, quota int, withToken bool) error {
	return withReconciledWalletQuota(userId, tokenId, tokenKey, withToken, func(tx *gorm.DB) error {
		if err := decreaseUserQuotaIfAvailable(tx, userId, quota); err != nil {
			return err
		}
		if withToken {
			return decreaseTokenQuotaIfAvailable(tx, userId, tokenId, quota)
		}
		return nil
	})
}

// AdjustWalletQuota persists a settlement delta for a wallet and its token.
// A positive delta consumes quota; a negative delta refunds it.
func AdjustWalletQuota(userId, tokenId int, tokenKey string, delta int, withToken bool) error {
	if delta == 0 {
		return nil
	}
	return withReconciledWalletQuota(userId, tokenId, tokenKey, withToken, func(tx *gorm.DB) error {
		if err := adjustUserQuota(tx, userId, -delta); err != nil {
			return err
		}
		if withToken {
			return adjustTokenQuota(tx, tokenId, -delta)
		}
		return nil
	})
}

func withReconciledWalletQuota(userId, tokenId int, tokenKey string, withToken bool, operation func(*gorm.DB) error) error {
	batchUpdateExecutionLock.Lock()
	batchUpdateLocks[BatchUpdateTypeUserQuota].Lock()
	if withToken {
		batchUpdateLocks[BatchUpdateTypeTokenQuota].Lock()
	}
	defer func() {
		if withToken {
			batchUpdateLocks[BatchUpdateTypeTokenQuota].Unlock()
		}
		batchUpdateLocks[BatchUpdateTypeUserQuota].Unlock()
		batchUpdateExecutionLock.Unlock()
	}()

	pendingUserQuota := batchUpdateStores[BatchUpdateTypeUserQuota][userId]
	pendingTokenQuota := 0
	if withToken {
		pendingTokenQuota = batchUpdateStores[BatchUpdateTypeTokenQuota][tokenId]
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := applyPendingUserQuota(tx, userId, pendingUserQuota); err != nil {
			return err
		}
		if withToken {
			if err := applyPendingTokenQuota(tx, tokenId, pendingTokenQuota); err != nil {
				return err
			}
		}
		return operation(tx)
	})
	if err == nil {
		delete(batchUpdateStores[BatchUpdateTypeUserQuota], userId)
		if withToken {
			delete(batchUpdateStores[BatchUpdateTypeTokenQuota], tokenId)
		}
		invalidateReservedQuotaCaches(userId, tokenKey, withToken)
	}
	return err
}

// ReserveTokenQuota conditionally reserves token quota without going through
// the batch updater. It is used when the funding source is a subscription.
func ReserveTokenQuota(userId, tokenId int, tokenKey string, quota int) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if quota == 0 {
		return nil
	}

	return reserveTokenQuotaWithPending(userId, tokenId, tokenKey, quota)
}

func reserveTokenQuotaWithPending(userId, tokenId int, tokenKey string, quota int) error {
	return withReconciledTokenQuota(tokenId, tokenKey, func(tx *gorm.DB) error {
		return decreaseTokenQuotaIfAvailable(tx, userId, tokenId, quota)
	})
}

func withReconciledTokenQuota(tokenId int, tokenKey string, operation func(*gorm.DB) error) error {
	batchUpdateExecutionLock.Lock()
	batchUpdateLocks[BatchUpdateTypeTokenQuota].Lock()
	defer func() {
		batchUpdateLocks[BatchUpdateTypeTokenQuota].Unlock()
		batchUpdateExecutionLock.Unlock()
	}()

	pendingTokenQuota := batchUpdateStores[BatchUpdateTypeTokenQuota][tokenId]
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := applyPendingTokenQuota(tx, tokenId, pendingTokenQuota); err != nil {
			return err
		}
		return operation(tx)
	})
	if err == nil {
		delete(batchUpdateStores[BatchUpdateTypeTokenQuota], tokenId)
		invalidateTokenQuotaCache(tokenKey)
	}
	return err
}

func applyPendingUserQuota(tx *gorm.DB, userId, delta int) error {
	if delta == 0 {
		return nil
	}
	result := tx.Unscoped().Model(&User{}).
		Where("id = ?", userId).
		Update("quota", gorm.Expr("quota + ?", delta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func applyPendingTokenQuota(tx *gorm.DB, tokenId, delta int) error {
	if delta == 0 {
		return nil
	}
	result := tx.Unscoped().Model(&Token{}).
		Where("id = ?", tokenId).
		Updates(map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota + ?", delta),
			"used_quota":    gorm.Expr("used_quota - ?", delta),
			"accessed_time": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func adjustUserQuota(tx *gorm.DB, userId, delta int) error {
	result := tx.Unscoped().Model(&User{}).
		Where("id = ?", userId).
		Update("quota", gorm.Expr("quota + ?", delta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func adjustTokenQuota(tx *gorm.DB, tokenId, delta int) error {
	result := tx.Unscoped().Model(&Token{}).
		Where("id = ?", tokenId).
		Updates(map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota + ?", delta),
			"used_quota":    gorm.Expr("used_quota - ?", delta),
			"accessed_time": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func decreaseUserQuotaIfAvailable(tx *gorm.DB, userId, quota int) error {
	result := tx.Model(&User{}).
		Where("id = ? AND quota >= ?", userId, quota).
		Update("quota", gorm.Expr("quota - ?", quota))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrInsufficientUserQuota
	}
	return nil
}

func decreaseTokenQuotaIfAvailable(tx *gorm.DB, userId, tokenId, quota int) error {
	result := tx.Model(&Token{}).
		Where("id = ? AND user_id = ? AND (unlimited_quota = ? OR remain_quota >= ?)", tokenId, userId, true, quota).
		Updates(map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota - ?", quota),
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"accessed_time": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrInsufficientTokenQuota
	}
	return nil
}

func invalidateReservedQuotaCaches(userId int, tokenKey string, withToken bool) {
	if err := invalidateUserCache(userId); err != nil {
		common.SysLog("failed to invalidate user quota cache: " + err.Error())
	}
	if !withToken || !common.RedisEnabled || tokenKey == "" {
		return
	}
	if err := cacheDeleteToken(tokenKey); err != nil {
		common.SysLog("failed to invalidate token quota cache: " + err.Error())
	}
}

// IncreaseUserQuotaDirect persists a wallet refund without using the batch updater.
func IncreaseUserQuotaDirect(userId, quota int) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	return adjustUserQuotaWithPending(userId, quota)
}

// DecreaseUserQuotaDirect persists a wallet settlement without using the batch updater.
func DecreaseUserQuotaDirect(userId, quota int) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	return adjustUserQuotaWithPending(userId, -quota)
}

// IncreaseTokenQuotaDirect persists a token refund without using the batch updater.
func IncreaseTokenQuotaDirect(tokenId int, tokenKey string, quota int) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	return adjustTokenQuotaWithPending(tokenId, tokenKey, quota)
}

// DecreaseTokenQuotaDirect persists a token settlement without using the batch updater.
func DecreaseTokenQuotaDirect(tokenId int, tokenKey string, quota int) error {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	return adjustTokenQuotaWithPending(tokenId, tokenKey, -quota)
}

func invalidateTokenQuotaCache(tokenKey string) {
	if !common.RedisEnabled || tokenKey == "" {
		return
	}
	if err := cacheDeleteToken(tokenKey); err != nil {
		common.SysLog("failed to invalidate token quota cache: " + err.Error())
	}
}

func adjustUserQuotaWithPending(userId, delta int) error {
	batchUpdateExecutionLock.Lock()
	batchUpdateLocks[BatchUpdateTypeUserQuota].Lock()
	defer func() {
		batchUpdateLocks[BatchUpdateTypeUserQuota].Unlock()
		batchUpdateExecutionLock.Unlock()
	}()

	pendingQuota := batchUpdateStores[BatchUpdateTypeUserQuota][userId]
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := applyPendingUserQuota(tx, userId, pendingQuota); err != nil {
			return err
		}
		return adjustUserQuota(tx, userId, delta)
	})
	if err == nil {
		delete(batchUpdateStores[BatchUpdateTypeUserQuota], userId)
		if err := invalidateUserCache(userId); err != nil {
			common.SysLog("failed to invalidate user quota cache: " + err.Error())
		}
	}
	return err
}

func adjustTokenQuotaWithPending(tokenId int, tokenKey string, delta int) error {
	return withReconciledTokenQuota(tokenId, tokenKey, func(tx *gorm.DB) error {
		return adjustTokenQuota(tx, tokenId, delta)
	})
}
