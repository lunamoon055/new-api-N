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
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Separator } from '@/components/ui/separator'
import { isVideoResolutionTierModel } from '../lib/price'
import {
  getVideoDocumentationParameters,
  isVideoCapabilityUnknown,
} from '../lib/video-documentation'
import type { IntegrationParameters, PricingModel } from '../types'

export function ModelCardCapabilities(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const p = getVideoDocumentationParameters(props.model)
  if (
    !props.model.mapped_endpoints?.some((endpoint) => endpoint.kind === 'video')
  )
    return null
  if (!p || p.capabilities_unknown) {
    return (
      <div className='mt-3'>
        <Separator className='mb-2' />
        <p className='text-muted-foreground text-xs'>
          {t('Model capabilities are unconfirmed.')}
        </p>
      </div>
    )
  }

  const durations = [
    ...new Set(
      p.durations?.filter((value) => Number.isFinite(value) && value > 0) ?? []
    ),
  ].sort((a, b) => a - b)
  let duration = t('Not published')
  if (!isVideoCapabilityUnknown(p, 'durations') && durations.length > 0) {
    const contiguous =
      durations.length > 1 &&
      durations.every(
        (value, index) => index === 0 || value === durations[index - 1] + 1
      )
    const value = contiguous
      ? `${durations[0]}–${durations[durations.length - 1]}`
      : durations.join(' / ')
    duration = `${value} ${t('seconds')}`
  }
  const siteVideoLimit =
    isVideoCapabilityUnknown(p, 'max_videos') &&
    Boolean(p.reference_video_billed_per_second) &&
    p.max_videos > 0
  const references = [
    {
      field: 'max_images',
      value: referenceLimit(p, 'max_images'),
      label: t('Images'),
    },
    {
      field: 'max_audios',
      value: referenceLimit(p, 'max_audios'),
      label: t('Audio'),
    },
    {
      field: 'max_videos',
      value: referenceLimit(p, 'max_videos'),
      label: t('Video'),
    },
  ].filter((item) => !siteVideoLimit || item.field !== 'max_videos')

  return (
    <div className='mt-3'>
      <Separator className='mb-2' />
      <dl className='flex flex-col gap-1 text-xs leading-4'>
        {!isVideoResolutionTierModel(props.model) && (
          <CapabilityRow label={t('Resolution')}>
            {p.resolutions?.length &&
            !isVideoCapabilityUnknown(p, 'resolutions')
              ? p.resolutions.join(' · ')
              : t('Not published')}
          </CapabilityRow>
        )}
        <CapabilityRow label={t('Video duration')}>{duration}</CapabilityRow>
        <CapabilityRow label={t('Reference materials')}>
          <span className='flex flex-wrap gap-x-2 gap-y-0.5'>
            {references.map((item) => (
              <span key={item.field}>
                <ReferenceCount
                  field={item.field}
                  value={item.value}
                  label={item.label}
                />
              </span>
            ))}
          </span>
        </CapabilityRow>
        {siteVideoLimit && (
          <CapabilityRow label={t('Reference video')}>
            {t('Site limit: {{count}} videos', { count: p.max_videos })}
          </CapabilityRow>
        )}
      </dl>
    </div>
  )
}

function CapabilityRow(props: { label: string; children: ReactNode }) {
  return (
    <div className='grid grid-cols-[4.5rem_minmax(0,1fr)] items-baseline gap-x-2'>
      <dt className='text-muted-foreground'>{props.label}</dt>
      <dd className='min-w-0 break-words'>{props.children}</dd>
    </div>
  )
}

function referenceLimit(
  p: IntegrationParameters,
  field: 'max_images' | 'max_audios' | 'max_videos'
): number | undefined {
  if (isVideoCapabilityUnknown(p, field) || !Number.isFinite(p[field]))
    return undefined
  return Math.max(0, p[field])
}

function ReferenceCount(props: {
  field: string
  value?: number
  label: string
}) {
  const { t } = useTranslation()
  if (props.value === undefined)
    return (
      <>
        {props.label} {t('Not published')}
      </>
    )
  if (props.value === 0)
    return (
      <>
        {props.label} {t('Not supported')}
      </>
    )
  if (props.field === 'max_images')
    return <>{t('{{count}} image references', { count: props.value })}</>
  if (props.field === 'max_audios')
    return <>{t('{{count}} audio references', { count: props.value })}</>
  return <>{t('{{count}} video references', { count: props.value })}</>
}
