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
import { useTranslation } from 'react-i18next'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { DEFAULT_TOKEN_UNIT } from '../constants'
import {
  getDynamicDisplayGroupRatio,
  getDynamicPricingSummary,
} from '../lib/dynamic-price'
import { isTokenBasedModel } from '../lib/model-helpers'
import {
  formatPrice,
  formatRequestPrice,
  getMediaPriceShortUnitLabelKey,
  getVideoReferenceTierPriceEntries,
  getVideoResolutionTierPriceEntries,
  getVideoResolutionTierShortUnitLabelKey,
  isVideoResolutionTierModel,
} from '../lib/price'
import type { PricingModel, TokenUnit } from '../types'

interface ModelCardPricingProps {
  model: PricingModel
  priceRate?: number
  usdExchangeRate?: number
  tokenUnit?: TokenUnit
  showRechargePrice?: boolean
}

export function ModelCardPricing(props: ModelCardPricingProps) {
  const { t } = useTranslation()
  const tokenUnit = props.tokenUnit ?? DEFAULT_TOKEN_UNIT
  const priceRate = props.priceRate ?? 1
  const usdExchangeRate = props.usdExchangeRate ?? 1
  const showRechargePrice = props.showRechargePrice ?? false
  const tokenUnitLabel = tokenUnit === 'K' ? '1K' : '1M'

  if (props.model.price_configured === false) {
    return <p className='text-sm font-semibold'>{t('Pricing pending')}</p>
  }

  if (isVideoResolutionTierModel(props.model)) {
    return <VideoTierPrices {...props} />
  }

  if (props.model.billing_mode === 'tiered_expr' && props.model.billing_expr) {
    const summary = getDynamicPricingSummary(props.model, {
      tokenUnit,
      showRechargePrice,
      priceRate,
      usdExchangeRate,
      groupRatioMultiplier: getDynamicDisplayGroupRatio(props.model),
    })
    if (!summary) return null
    if (summary.isSpecialExpression) {
      return (
        <div className='flex min-w-0 flex-col gap-1 text-xs'>
          <span>{t('Special billing expression')}</span>
          <code className='text-muted-foreground line-clamp-2 font-mono break-all'>
            {summary.rawExpression}
          </code>
        </div>
      )
    }
    if (summary.primaryEntries.length === 0) {
      return <p className='text-sm'>{t('Dynamic Pricing')}</p>
    }
    return (
      <div className='flex flex-wrap gap-x-4 gap-y-1 text-xs leading-6'>
        {summary.primaryEntries.map((entry) => (
          <span key={entry.key} className='text-muted-foreground'>
            {t(entry.shortLabel)}{' '}
            <span className='text-foreground font-mono font-semibold'>
              {entry.formatted}
            </span>
            /{tokenUnitLabel}
          </span>
        ))}
      </div>
    )
  }

  if (isTokenBasedModel(props.model)) {
    const priceTypes = ['input', 'output'] as const
    return (
      <div className='flex flex-wrap gap-x-4 gap-y-1 text-xs leading-6'>
        {priceTypes.map((type) => (
          <span key={type} className='text-muted-foreground'>
            {t(type === 'input' ? 'Input' : 'Output')}{' '}
            <span className='text-foreground font-mono font-semibold'>
              {formatPrice(
                props.model,
                type,
                tokenUnit,
                showRechargePrice,
                priceRate,
                usdExchangeRate
              )}
            </span>
            /{tokenUnitLabel}
          </span>
        ))}
        {props.model.cache_ratio != null && (
          <span className='text-muted-foreground'>
            {t('Cached')}{' '}
            <span className='font-mono'>
              {formatPrice(
                props.model,
                'cache',
                tokenUnit,
                showRechargePrice,
                priceRate,
                usdExchangeRate
              )}
            </span>
            /{tokenUnitLabel}
          </span>
        )}
      </div>
    )
  }

  return (
    <p className='text-muted-foreground text-sm'>
      <span className='text-foreground font-mono font-semibold'>
        {formatRequestPrice(
          props.model,
          showRechargePrice,
          priceRate,
          usdExchangeRate
        )}
      </span>
      /{t(getMediaPriceShortUnitLabelKey(props.model))}
    </p>
  )
}

function VideoTierPrices(props: ModelCardPricingProps) {
  const { t } = useTranslation()
  const options = {
    showWithRecharge: props.showRechargePrice,
    priceRate: props.priceRate,
    usdExchangeRate: props.usdExchangeRate,
  }
  const generation = getVideoResolutionTierPriceEntries(props.model, options)
  const references = getVideoReferenceTierPriceEntries(props.model, options)
  const referenceByResolution = new Map(
    references.map((entry) => [entry.resolution, entry])
  )
  const resolutions = getVideoResolutionTierPriceEntries(
    {
      ...props.model,
      video_resolution_prices: {
        ...props.model.video_reference_prices,
        ...props.model.video_resolution_prices,
      },
    },
    options
  ).map((entry) => entry.resolution)
  const generationByResolution = new Map(
    generation.map((entry) => [entry.resolution, entry])
  )
  const unit = t(getVideoResolutionTierShortUnitLabelKey(props.model))

  return (
    <div className='flex min-w-0 flex-col gap-1'>
      <Table className='table-fixed text-xs'>
        <caption className='sr-only'>
          {t('Model pricing')} — {props.model.model_name}
        </caption>
        <TableHeader>
          <TableRow>
            <TableHead className='h-auto w-1/4 px-0 pb-1 whitespace-normal'>
              {t('Resolution')}
            </TableHead>
            <TableHead className='h-auto px-1 pb-1 whitespace-normal'>
              {t('Generation price')}
            </TableHead>
            {references.length > 0 && (
              <TableHead className='h-auto px-0 pb-1 text-right whitespace-normal'>
                {t('Reference video')}
              </TableHead>
            )}
          </TableRow>
        </TableHeader>
        <TableBody>
          {resolutions.map((resolution) => {
            const entry = generationByResolution.get(resolution)
            const reference = referenceByResolution.get(resolution)
            return (
              <TableRow key={resolution}>
                <TableCell className='px-0 py-0 font-mono'>
                  {resolution}
                </TableCell>
                <TableCell className='px-1 py-0 font-mono font-semibold'>
                  {entry ? (
                    <>
                      {entry.formatted}/{unit}
                    </>
                  ) : (
                    t('Unconfigured')
                  )}
                </TableCell>
                {references.length > 0 && (
                  <TableCell className='px-0 py-0 text-right font-mono font-semibold'>
                    {reference ? (
                      <>
                        {reference.formatted}/{t('seconds')}
                      </>
                    ) : (
                      t('Unconfigured')
                    )}
                  </TableCell>
                )}
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
      {references.length > 0 && (
        <p className='text-muted-foreground text-xs leading-4'>
          {t('Reference videos are billed separately by input duration.')}
        </p>
      )}
    </div>
  )
}
