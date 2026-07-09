import { describe, expect, it } from 'bun:test'
import {
  QUOTA_TYPES,
  QUOTA_TYPE_VALUES,
} from '../src/features/pricing/constants'
import { filterByQuotaType } from '../src/features/pricing/lib/filters'
import { isTokenBasedModel } from '../src/features/pricing/lib/model-helpers'
import {
  getVideoResolutionTierPriceEntries,
  getVideoResolutionTierShortUnitLabelKey,
  isVideoResolutionTierModel,
} from '../src/features/pricing/lib/price'
import type { PricingModel } from '../src/features/pricing/types'

const tieredVideoModel: PricingModel = {
  id: 1,
  model_name: 'video-2.0-mini',
  quota_type: QUOTA_TYPE_VALUES.TOKEN,
  model_ratio: 337.5,
  completion_ratio: 1,
  enable_groups: ['default', 'vip'],
  group_ratio: {
    default: 1,
    vip: 0.9,
  },
  video_billing_mode: 'tiered_request',
  video_resolution_prices: {
    '480p': 0.05,
    '720p': 0.1,
  },
}

describe('pricing video resolution tier helpers', () => {
  it('treats video resolution tier models as non-token pricing', () => {
    expect(isVideoResolutionTierModel(tieredVideoModel)).toBe(true)
    expect(isTokenBasedModel(tieredVideoModel)).toBe(false)
  })

  it('formats resolution tier prices with the same group multiplier rules as marketplace cards', () => {
    expect(
      getVideoResolutionTierPriceEntries(tieredVideoModel).map((entry) => ({
        resolution: entry.resolution,
        formatted: entry.formatted,
      }))
    ).toEqual([
      { resolution: '480p', formatted: '$0.045' },
      { resolution: '720p', formatted: '$0.09' },
    ])
  })

  it('uses compact unit labels for marketplace tier prices', () => {
    expect(getVideoResolutionTierShortUnitLabelKey(tieredVideoModel)).toBe(
      'times'
    )
  })

  it('keeps tiered video pricing out of token and request marketplace filters', () => {
    expect(filterByQuotaType([tieredVideoModel], QUOTA_TYPES.TOKEN)).toEqual([])
    expect(filterByQuotaType([tieredVideoModel], QUOTA_TYPES.REQUEST)).toEqual(
      []
    )
    expect(
      filterByQuotaType([tieredVideoModel], QUOTA_TYPES.TIERED_REQUEST)
    ).toEqual([tieredVideoModel])
  })
})
