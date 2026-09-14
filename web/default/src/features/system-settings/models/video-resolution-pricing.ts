export const VIDEO_RESOLUTION_PRICE_KEYS = [
  '480p',
  '720p',
  '1080p',
  '1k',
  '2k',
  '4k',
] as const

export type VideoResolutionPriceKey =
  (typeof VIDEO_RESOLUTION_PRICE_KEYS)[number]

export type VideoResolutionPriceInput = Record<VideoResolutionPriceKey, string>
export type VideoResolutionPriceMap = Partial<
  Record<VideoResolutionPriceKey, number>
>

export type VideoBillingMode =
  | 'dynamic'
  | 'fixed'
  | 'tiered_seconds'
  | 'tiered_request'

export type VideoResolutionPricingMode = Extract<
  VideoBillingMode,
  'tiered_seconds' | 'tiered_request'
>

export const EMPTY_VIDEO_RESOLUTION_PRICE_INPUT: VideoResolutionPriceInput = {
  '480p': '',
  '720p': '',
  '1080p': '',
  '1k': '',
  '2k': '',
  '4k': '',
}

export function isVideoResolutionPricingMode(
  value: unknown
): value is VideoResolutionPricingMode {
  return value === 'tiered_seconds' || value === 'tiered_request'
}

export function normalizeVideoBillingMode(value: unknown): VideoBillingMode {
  if (value === 'fixed') return 'fixed'
  if (isVideoResolutionPricingMode(value)) return value
  return 'dynamic'
}

export function normalizeVideoResolutionPriceInput(
  input: Partial<Record<VideoResolutionPriceKey, string | number>>
): VideoResolutionPriceMap {
  return VIDEO_RESOLUTION_PRICE_KEYS.reduce<VideoResolutionPriceMap>(
    (acc, resolution) => {
      const raw = input[resolution]
      if (raw === '' || raw === null || raw === undefined) return acc

      const value = Number(raw)
      if (Number.isFinite(value)) {
        acc[resolution] = value
      }
      return acc
    },
    {}
  )
}

export function formatVideoResolutionPriceInput(
  prices?: Partial<Record<VideoResolutionPriceKey, number | string>>
): VideoResolutionPriceInput {
  return VIDEO_RESOLUTION_PRICE_KEYS.reduce<VideoResolutionPriceInput>(
    (acc, resolution) => {
      const value = prices?.[resolution]
      acc[resolution] =
        value === null || value === undefined || value === ''
          ? ''
          : String(value)
      return acc
    },
    { ...EMPTY_VIDEO_RESOLUTION_PRICE_INPUT }
  )
}

export function hasAnyVideoResolutionPrice(
  input: Partial<Record<VideoResolutionPriceKey, string | number>>
): boolean {
  return VIDEO_RESOLUTION_PRICE_KEYS.some((resolution) => {
    const raw = input[resolution]
    if (raw === null || raw === undefined || String(raw).trim() === '') {
      return false
    }
    const value = Number(raw)
    return Number.isFinite(value) && value >= 0
  })
}

export function buildVideoResolutionPricingData(
  name: string,
  mode: VideoResolutionPricingMode,
  prices: Partial<Record<VideoResolutionPriceKey, string | number>>
) {
  return {
    name,
    billingMode: 'per-request' as const,
    videoBillingMode: mode,
    videoResolutionPrices: normalizeVideoResolutionPriceInput(prices),
  }
}
