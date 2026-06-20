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
  mode?: CreationMode
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
      if (mode === 'video' && cost.request_quota != null) {
        return formatQuota(cost.request_quota)
      }
      const requestPrice = formatCurrencyFromUSD(cost.request_price, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })
      const requestQuota = cost.request_quota
        ? ` · ${formatQuota(cost.request_quota)}`
        : ''
      return `${requestPrice} ${t('per request')}${requestQuota}${groupSuffix}`
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
  }
}

function formatCostNumber(value: number) {
  return Number.parseFloat(value.toFixed(6)).toString()
}
