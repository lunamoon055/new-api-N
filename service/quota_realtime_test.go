package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingBillingSettler struct {
	preConsumed int
	reserved    []int
	settled     []int
}

func (r *recordingBillingSettler) Settle(actualQuota int) error {
	r.settled = append(r.settled, actualQuota)
	return nil
}

func (r *recordingBillingSettler) Refund(*gin.Context) {}

func (r *recordingBillingSettler) NeedsRefund() bool { return false }

func (r *recordingBillingSettler) GetPreConsumedQuota() int { return r.preConsumed }

func (r *recordingBillingSettler) Reserve(targetQuota int) error {
	r.reserved = append(r.reserved, targetQuota)
	return nil
}

func TestReserveWssQuotaDefersChargingToFinalSettlement(t *testing.T) {
	recorder := &recordingBillingSettler{preConsumed: 100}
	info := &relaycommon.RelayInfo{
		Billing: recorder,
		PriceData: types.PriceData{
			UsePrice:   true,
			ModelPrice: 0.01,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 2,
			},
		},
	}
	usage := &dto.RealtimeUsage{TotalTokens: 150, InputTokens: 100, OutputTokens: 50}
	ctx := newBillingTestContext()

	wantQuota, _, _ := calculateWssQuota(info, usage)
	require.NoError(t, ReserveWssQuota(ctx, info, usage))
	assert.Equal(t, []int{wantQuota}, recorder.reserved)
	assert.Empty(t, recorder.settled)

	require.NoError(t, SettleBilling(ctx, info, wantQuota))
	assert.Equal(t, []int{wantQuota}, recorder.settled)
}

func TestCalculateWssQuotaIncludesTieredUsageDimensions(t *testing.T) {
	const expression = "p + c + cr * 2 + ai * 3 + ao * 4"
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:  "tiered_expr",
			ExprString:   expression,
			ExprHash:     billingexpr.ExprHashString(expression),
			GroupRatio:   1,
			QuotaPerUnit: 1_000_000,
			ExprVersion:  1,
		},
	}
	usage := &dto.RealtimeUsage{
		TotalTokens:  150,
		InputTokens:  100,
		OutputTokens: 50,
		InputTokenDetails: dto.InputTokenDetails{
			CachedTokens: 10,
			TextTokens:   70,
			AudioTokens:  20,
		},
		OutputTokenDetails: dto.OutputTokenDetails{
			TextTokens:  40,
			AudioTokens: 10,
		},
	}

	quota, result, tiered := calculateWssQuota(info, usage)
	require.True(t, tiered)
	require.NotNil(t, result)
	assert.Equal(t, 230, quota)
}
