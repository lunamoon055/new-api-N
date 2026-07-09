import { describe, expect, it } from 'bun:test'
import {
  VIDEO_RESOLUTION_PRICE_KEYS,
  buildVideoResolutionPricingData,
  normalizeVideoResolutionPriceInput,
} from '../src/features/system-settings/models/video-resolution-pricing'

describe('video resolution pricing helpers', () => {
  it('keeps the supported resolution order used by the pricing sheet', () => {
    expect(VIDEO_RESOLUTION_PRICE_KEYS).toEqual(['480p', '720p', '1080p', '4k'])
  })

  it('normalizes editable resolution price inputs into numeric settings', () => {
    expect(
      normalizeVideoResolutionPriceInput({
        '480p': '0.01',
        '720p': '0.02',
        '1080p': '',
        '4k': '0.08',
      })
    ).toEqual({
      '480p': 0.01,
      '720p': 0.02,
      '4k': 0.08,
    })
  })

  it('builds model pricing data for tiered seconds mode', () => {
    expect(
      buildVideoResolutionPricingData('video-2.0-fast', 'tiered_seconds', {
        '480p': '0.01',
        '720p': '0.02',
        '1080p': '0.04',
        '4k': '0.08',
      })
    ).toEqual({
      name: 'video-2.0-fast',
      billingMode: 'per-request',
      videoBillingMode: 'tiered_seconds',
      videoResolutionPrices: {
        '480p': 0.01,
        '720p': 0.02,
        '1080p': 0.04,
        '4k': 0.08,
      },
    })
  })
})
