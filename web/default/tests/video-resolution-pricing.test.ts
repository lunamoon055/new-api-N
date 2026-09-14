import { describe, expect, it } from 'bun:test'
import {
  VIDEO_RESOLUTION_PRICE_KEYS,
  buildVideoResolutionPricingData,
  hasAnyVideoResolutionPrice,
  normalizeVideoResolutionPriceInput,
} from '../src/features/system-settings/models/video-resolution-pricing'

describe('video resolution pricing helpers', () => {
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
