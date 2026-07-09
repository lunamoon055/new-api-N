/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { formatCurrencyFromUSD } from '@/lib/currency'
import { formatQuota } from '@/lib/format'
import type { CreationMode, CreationModelCost } from './types'

export function formatCreationModelCost(
  cost: CreationModelCost | undefined,
  t: (key: string) => string,
  mode?: CreationMode,
  selectedResolution?: string
) {
  if (!cost) return t('Pricing pending')
  const groupSuffix =
    cost.group_ratio && cost.group_ratio !== 1
      ? ` · ${t('Group')} x${formatCostNumber(cost.group_ratio)}`
      : ''

  switch (cost.billing_mode) {
    case 'dynamic':
      return `${t('Dynamic pricing')}${groupSuffix}`
    case 'per_request': {
      const unit = t('times')
      if (cost.request_quota != null) {
        return `${formatQuota(cost.request_quota)}/${unit}`
      }
      const requestPrice = formatCurrencyFromUSD(cost.request_price, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })
      return `${requestPrice}/${unit}`
    }
    case 'per_token': {
      const inputPrice = formatCurrencyFromUSD(cost.input_price_per_million, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })
      const outputPrice = formatCurrencyFromUSD(cost.output_price_per_million, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })
      return `${t('Input')} ${inputPrice}/1M · ${t('Output')} ${outputPrice}/1M${groupSuffix}`
    }
    case 'tiered_seconds':
    case 'tiered_request':
      return formatVideoResolutionTierCost(cost, t, mode, selectedResolution)
  }
}

function formatCostNumber(value: number) {
  return Number.parseFloat(value.toFixed(6)).toString()
}

const videoResolutionOrder = ['480p', '720p', '1080p', '4k']

function formatVideoResolutionTierCost(
  cost: CreationModelCost,
  t: (key: string) => string,
  mode: CreationMode | undefined,
  selectedResolution: string | undefined
) {
  const normalizedResolution = normalizeVideoResolution(selectedResolution)
  const unit = getVideoResolutionCostUnit(cost, t)
  if (mode === 'video' && normalizedResolution) {
    const quota = cost.video_resolution_quotas?.[normalizedResolution]
    if (quota != null) {
      return `${formatQuota(quota)}/${unit} · ${normalizedResolution}`
    }

    const price = cost.video_resolution_prices?.[normalizedResolution]
    if (price != null) {
      return `${formatCurrencyFromUSD(price, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })}/${unit} · ${normalizedResolution}`
    }
  }

  const summary = getVideoResolutionCostEntries(cost)
    .map(({ resolution, quota, price }) => {
      if (quota != null) return `${resolution} ${formatQuota(quota)}/${unit}`
      return `${resolution} ${formatCurrencyFromUSD(price, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })}/${unit}`
    })
    .join(' · ')

  return summary || t('Pricing pending')
}

function getVideoResolutionCostUnit(
  cost: CreationModelCost,
  t: (key: string) => string
) {
  return cost.billing_mode === 'tiered_seconds' ? t('seconds') : t('times')
}

function getVideoResolutionCostEntries(cost: CreationModelCost) {
  const resolutions = [
    ...videoResolutionOrder,
    ...Object.keys(cost.video_resolution_quotas ?? {}),
    ...Object.keys(cost.video_resolution_prices ?? {}),
  ]
  const seen = new Set<string>()

  return resolutions.flatMap((resolution) => {
    const normalizedResolution = normalizeVideoResolution(resolution)
    if (!normalizedResolution || seen.has(normalizedResolution)) return []
    seen.add(normalizedResolution)

    const quota = cost.video_resolution_quotas?.[normalizedResolution]
    const price = cost.video_resolution_prices?.[normalizedResolution]
    if (quota == null && price == null) return []

    return [{ resolution: normalizedResolution, quota, price }]
  })
}

function normalizeVideoResolution(value?: string) {
  const normalized = value?.trim().toLowerCase().replace(/\s+/g, '')
  switch (normalized) {
    case '480':
    case '480p':
      return '480p'
    case '720':
    case '720p':
      return '720p'
    case '1080':
    case '1080p':
      return '1080p'
    case '2160':
    case '2160p':
    case '4k':
      return '4k'
    default:
      return undefined
  }
}
