import { describe, expect, it } from 'bun:test'
import {
  VIDEO_RESOLUTION_PRICE_KEYS,
  buildVideoResolutionPricingData,
  hasAnyVideoResolutionPrice,
  normalizeVideoResolutionPriceInput,
} from '../src/features/system-settings/models/video-resolution-pricing'
import { getVideoReferenceTierPriceEntries } from '../src/features/pricing/lib/price'
import type { PricingModel } from '../src/features/pricing/types'

describe('video resolution pricing helpers', () => {
  it('keeps reference prices separate, applies group rates, and preserves explicit zero', () => {
    const model = {
      model_name: 'public-sd',
      video_billing_mode: 'tiered_seconds',
      video_resolution_prices: { '720p': 0.1 },
      video_reference_prices: { '720p': 0.2, '1080p': 0 },
      enable_groups: ['default'],
      group_ratio: { default: 2 },
    } as PricingModel
    const entries = getVideoReferenceTierPriceEntries(model)
    expect(entries.map((entry) => [entry.resolution, entry.price])).toEqual([
      ['720p', 0.4],
      ['1080p', 0],
    ])
    expect(model.video_resolution_prices).toEqual({ '720p': 0.1 })
    expect(
      getVideoReferenceTierPriceEntries({
        ...model,
        video_reference_prices: undefined,
      })
    ).toEqual([])
  })
  it('keeps the supported resolution order used by the pricing sheet', () => {
    expect(VIDEO_RESOLUTION_PRICE_KEYS).toEqual([
      '480p',
      '720p',
      '1080p',
      '1k',
      '2k',
      '4k',
    ])
  })

  it('normalizes editable resolution price inputs into numeric settings', () => {
    expect(
      normalizeVideoResolutionPriceInput({
        '480p': '0.01',
        '720p': '0.02',
        '1080p': '',
        '1k': '0.05',
        '2k': '0.06',
        '4k': '0.08',
      })
    ).toEqual({
      '480p': 0.01,
      '720p': 0.02,
      '1k': 0.05,
      '2k': 0.06,
      '4k': 0.08,
    })
  })

  it('builds both tier modes with only the filled resolution prices', () => {
    for (const mode of ['tiered_seconds', 'tiered_request'] as const) {
      expect(
        buildVideoResolutionPricingData('video-2.0-fast', mode, {
          '720p': '0.02',
          '2k': '0.06',
        })
      ).toEqual({
        name: 'video-2.0-fast',
        billingMode: 'per-request',
        videoBillingMode: mode,
        videoResolutionPrices: {
          '720p': 0.02,
          '2k': 0.06,
        },
      })
    }
  })

  it('allows saving any non-empty subset of resolution prices', () => {
    expect(hasAnyVideoResolutionPrice({ '720p': '0.02' })).toBe(true)
    expect(hasAnyVideoResolutionPrice({ '2k': 0 })).toBe(true)
    expect(hasAnyVideoResolutionPrice({ '480p': '', '2k': '  ' })).toBe(false)
  })
})
