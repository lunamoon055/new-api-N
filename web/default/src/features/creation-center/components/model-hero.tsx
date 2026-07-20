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
import { Bot, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { formatCreationModelCost } from '../cost'
import type { CreationMode, CreationModel } from '../types'

type ModelHeroProps = {
  mode: CreationMode
  model?: CreationModel
  selectedResolution?: string
}

export function ModelHero(props: ModelHeroProps) {
  const { t } = useTranslation()
  const title = props.model?.id || t('Select a model')
  const costLabel = props.model
    ? formatCreationModelCost(
        props.model.cost,
        t,
        props.mode,
        props.selectedResolution
      )
    : undefined
  const fallback =
    props.mode === 'chat'
      ? t('Choose a configured chat model for writing, coding, and analysis.')
      : props.mode === 'image'
        ? t(
            'Choose an image model and add references before composing a prompt.'
          )
        : t('Choose a video model and prepare a prompt for the next step.')
  const modeLabel =
    props.mode === 'chat'
      ? t('Chat')
      : props.mode === 'image'
        ? t('Image')
        : t('Video')
  const visibleTags =
    props.model?.tags?.filter((tag) => tag !== 'async').slice(0, 3) ?? []

  return (
    <section className='min-w-0 overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm dark:border-white/10 dark:bg-[#101820]'>
      <div className='flex min-w-0 flex-col gap-3 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between'>
        <div className='flex min-w-0 items-center gap-3'>
          <div className='relative flex size-9 shrink-0 items-center justify-center rounded-md border border-cyan-500/25 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300'>
            <Bot className='size-4' />
            <span className='absolute -right-1 -bottom-1 flex size-4 items-center justify-center rounded-sm border-2 border-white bg-cyan-600 text-white dark:border-[#101820] dark:bg-cyan-400 dark:text-slate-950'>
              <Sparkles className='size-2.5' />
            </span>
          </div>
          <div className='min-w-0'>
            <div className='flex min-w-0 items-center gap-2'>
              <span className='text-muted-foreground shrink-0 text-[11px] font-medium'>
                {t('Current model')}
              </span>
              <span className='h-3 w-px bg-slate-200 dark:bg-white/10' />
              <h2 className='truncate text-sm font-semibold'>{title}</h2>
            </div>
            <p className='text-muted-foreground mt-0.5 line-clamp-1 max-w-3xl text-[11px] leading-4'>
              {props.model?.description || fallback}
            </p>
          </div>
        </div>

        <div className='flex shrink-0 flex-wrap items-center gap-2 sm:justify-end'>
          <Badge
            variant='outline'
            className='border-cyan-500/25 bg-cyan-500/[0.06] text-cyan-700 dark:text-cyan-300'
          >
            {modeLabel}
          </Badge>
          {visibleTags.map((tag) => (
            <Badge key={tag} variant='outline'>
              {tag}
            </Badge>
          ))}
          {costLabel && (
            <Badge variant='outline' className='max-w-full tabular-nums'>
              <span className='truncate'>
                {t('Consumption')}: {costLabel}
              </span>
            </Badge>
          )}
        </div>
      </div>
    </section>
  )
}
