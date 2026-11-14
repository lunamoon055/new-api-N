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
import { Fragment, memo } from 'react'
import { ChevronRight, Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { StatusBadge } from '@/components/status-badge'
import { Separator } from '@/components/ui/separator'
import { DEFAULT_TOKEN_UNIT } from '../constants'
import { parseTags } from '../lib/filters'
import { isTokenBasedModel } from '../lib/model-helpers'
import type { PricingModel, TokenUnit } from '../types'
import { ModelAvailability } from './model-availability'
import { ModelCardCapabilities } from './model-card-capabilities'
import { ModelCardPricing } from './model-card-pricing'
import { ModelDescriptionEditor } from './model-description-editor'
import { ModelPerfBadge, type ModelPerfBadgeData } from './model-perf-badge'

export interface ModelCardProps {
  model: PricingModel
  onClick: () => void
  priceRate?: number
  usdExchangeRate?: number
  tokenUnit?: TokenUnit
  showRechargePrice?: boolean
  perf?: ModelPerfBadgeData
}

export const ModelCard = memo(function ModelCard(props: ModelCardProps) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const tokenUnit = props.tokenUnit ?? DEFAULT_TOKEN_UNIT
  const isVideo =
    props.model.mapped_endpoints?.some(
      (endpoint) => endpoint.kind === 'video'
    ) || Boolean(props.model.video_billing_mode)
  const isTokenBased = !isVideo && isTokenBasedModel(props.model)
  const tokenUnitLabel = tokenUnit === 'K' ? '1K' : '1M'
  const tags = parseTags(props.model.tags)
  const groups = props.model.enable_groups || []
  const endpoints = props.model.supported_endpoint_types || []
  const vendorIcon = props.model.vendor_icon
    ? getLobeIcon(props.model.vendor_icon, 28)
    : null
  const initial = props.model.model_name?.charAt(0).toUpperCase() || '?'
  const isDynamicPricing =
    props.model.billing_mode === 'tiered_expr' &&
    Boolean(props.model.billing_expr)
  const primaryGroup = groups[0]
  const bottomTags = [
    ...new Set([...endpoints.slice(0, 2), ...tags.slice(0, 2)]),
  ]
  const hiddenCount =
    Math.max(groups.length - 1, 0) +
    Math.max(endpoints.length - 2, 0) +
    Math.max(tags.length - 2, 0)
  let billingLabel = t('Per Request')
  if (isTokenBased) billingLabel = t('Token-based')
  if (isVideo) {
    const perSecond =
      props.model.video_billing_mode === 'dynamic' ||
      props.model.video_billing_mode === 'tiered_seconds'
    billingLabel = perSecond ? t('Billed per second') : t('Billed per request')
  }

  const handleCopy = (event: React.MouseEvent): void => {
    event.stopPropagation()
    copyToClipboard(props.model.model_name || '')
  }

  return (
    <div
      className={cn(
        'group relative flex min-w-0 flex-col rounded-xl border p-3 transition-colors sm:p-4',
        'hover:bg-muted/20'
      )}
    >
      <div className='flex items-start justify-between gap-2.5 sm:gap-3'>
        <div className='flex min-w-0 flex-1 items-start gap-2.5 sm:gap-3'>
          <div className='bg-muted/40 flex size-9 shrink-0 items-center justify-center rounded-lg sm:size-10 sm:rounded-xl'>
            {vendorIcon || (
              <span className='text-muted-foreground text-sm font-bold'>
                {initial}
              </span>
            )}
          </div>
          <h3 className='text-foreground min-w-0 font-mono text-[15px] leading-snug font-bold [overflow-wrap:anywhere]'>
            {props.model.model_name
              .split(/(?<=[/_-])|(?<=[A-Za-z])(?=\d)/)
              .map((part, index) => (
                <Fragment key={`${index}-${part}`}>
                  {part}
                  <wbr />
                </Fragment>
              ))}
          </h3>
        </div>
        <div className='flex shrink-0 items-center gap-1.5'>
          <button
            type='button'
            onClick={props.onClick}
            className='text-muted-foreground hover:text-foreground hover:bg-muted focus-visible:ring-ring inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs transition-colors focus-visible:ring-2 sm:px-2.5 sm:py-1.5'
          >
            {t('Details')}
            <ChevronRight className='size-3.5' aria-hidden='true' />
          </button>
          <button
            type='button'
            onClick={handleCopy}
            className='text-muted-foreground hover:text-foreground hover:bg-muted focus-visible:ring-ring rounded-md border p-1.5 transition-colors focus-visible:ring-2'
            title={t('Copy')}
            aria-label={t('Copy model name: {{model}}', {
              model: props.model.model_name,
            })}
          >
            <Copy className='size-3.5' aria-hidden='true' />
          </button>
        </div>
      </div>

      <div className='mt-3'>
        <ModelCardPricing
          model={props.model}
          tokenUnit={tokenUnit}
          priceRate={props.priceRate}
          usdExchangeRate={props.usdExchangeRate}
          showRechargePrice={props.showRechargePrice}
        />
      </div>
      <ModelCardCapabilities model={props.model} />
      {props.model.description?.trim() && (
        <p
          className='text-muted-foreground mt-3 line-clamp-2 text-[13px] leading-relaxed'
          title={props.model.description}
        >
          {props.model.description}
        </p>
      )}

      <div className='mt-auto pt-3'>
        <Separator className='mb-2' />
        <div className='flex flex-wrap items-center justify-between gap-x-2 gap-y-2'>
          <ModelAvailability
            modelName={props.model.model_name}
            available={props.model.display_available}
          />
          <ModelDescriptionEditor
            modelName={props.model.model_name}
            description={props.model.description}
            compact
          />
        </div>
        <div className='mt-1.5 flex min-w-0 flex-wrap items-center justify-between gap-x-3 gap-y-2'>
          <div className='flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1'>
            {primaryGroup && (
              <span className='text-muted-foreground text-xs font-medium whitespace-nowrap'>
                {primaryGroup} {t('Groups')}
              </span>
            )}
            <span className='text-muted-foreground text-xs font-medium whitespace-nowrap'>
              {billingLabel}
            </span>
            {isDynamicPricing && (
              <StatusBadge
                label={t('Dynamic Pricing')}
                variant='warning'
                copyable={false}
                size='sm'
              />
            )}
          </div>
          <ModelPerfBadge perf={props.perf} className='shrink-0' />
        </div>
        <div className='mt-1 flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1 sm:gap-x-3'>
          {bottomTags.map((item) => (
            <span
              key={item}
              title={item}
              className='text-muted-foreground max-w-40 min-w-0 truncate text-xs'
            >
              {item}
            </span>
          ))}
          {isTokenBased && (
            <span className='text-muted-foreground text-xs whitespace-nowrap'>
              {tokenUnitLabel}
            </span>
          )}
          {hiddenCount > 0 && (
            <span className='text-muted-foreground text-xs whitespace-nowrap'>
              +{hiddenCount}
            </span>
          )}
        </div>
      </div>
    </div>
  )
})
