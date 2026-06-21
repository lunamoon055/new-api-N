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
import { Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatCreationModelCost } from '../cost'
import type { CreationMode, CreationModel } from '../types'

type ModelHeroProps = {
  mode: CreationMode
  model?: CreationModel
}

export function ModelHero(props: ModelHeroProps) {
  const { t } = useTranslation()
  const title = props.model?.id || t('Select a model')
  const costLabel = props.model
    ? formatCreationModelCost(props.model.cost, t, props.mode)
    : undefined
  const fallback =
    props.mode === 'chat'
      ? t('Choose a configured chat model for writing, coding, and analysis.')
      : props.mode === 'image'
        ? t(
            'Choose an image model and add references before composing a prompt.'
          )
        : t('Choose a video model and prepare a prompt for the next step.')

  return (
    <section className='bg-card flex min-h-[22rem] items-center justify-center rounded-lg border p-6 text-center'>
      <div className='max-w-lg'>
        <div className='bg-primary text-primary-foreground mx-auto flex size-16 items-center justify-center rounded-lg shadow-sm'>
          <Sparkles className='size-7' />
        </div>
        <div className='text-primary mt-5 flex items-center justify-center gap-2 text-[11px] font-medium'>
          <span className='bg-primary/30 h-px w-10' />
          {t('Current model')}
          <span className='bg-primary/30 h-px w-10' />
        </div>
        <h2 className='mt-3 text-2xl font-semibold break-words'>{title}</h2>
        <p className='text-muted-foreground mt-3 text-sm leading-6'>
          {props.model?.description || fallback}
        </p>
        {costLabel && (
          <div className='text-primary bg-primary/10 mx-auto mt-4 inline-flex max-w-full items-center rounded-full px-3 py-1 text-xs font-medium'>
            <span className='truncate'>
              {t('Consumption')}: {costLabel}
            </span>
          </div>
        )}
      </div>
    </section>
  )
}
