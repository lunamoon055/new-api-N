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
import {
  Bot,
  FileText,
  History,
  Image,
  MessageSquare,
  Plus,
  Settings2,
  Video,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { CREATION_MODES } from '../constants'
import { formatCreationModelCost } from '../cost'
import type { CreationMode, CreationModel } from '../types'

type CreationSidebarProps = {
  mode: CreationMode
  models: CreationModel[]
  selectedModel?: CreationModel
  modeCounts: Record<CreationMode, number>
  loading: boolean
  error: boolean
  canManageCategories: boolean
  canManageDescriptions: boolean
  onModeChange: (mode: CreationMode) => void
  onModelChange: (model: CreationModel) => void
  onHistory: () => void
  onNewSession: () => void
  onManageCategories: () => void
  onManageDescriptions: () => void
}

export function CreationSidebar(props: CreationSidebarProps) {
  const { t } = useTranslation()

  return (
    <aside className='border-border/70 bg-background/90 flex flex-col border-r'>
      <div className='space-y-4 border-b p-4'>
        <div>
          <h1 className='text-lg font-semibold'>{t('Creation Center')}</h1>
          <p className='text-muted-foreground mt-1 text-xs leading-5'>
            {t(
              'Create chat, image, and video tasks with the models configured in your workspace.'
            )}
          </p>
        </div>
        <div className='grid grid-cols-3 gap-2' role='tablist'>
          {CREATION_MODES.map((item) => (
            <ModeButton
              key={item}
              mode={item}
              count={props.modeCounts[item]}
              active={props.mode === item}
              onClick={() => props.onModeChange(item)}
            />
          ))}
        </div>
        <div className='grid grid-cols-2 gap-2'>
          <Button variant='outline' size='sm' onClick={props.onHistory}>
            <History />
            {t('History')}
          </Button>
          <Button variant='outline' size='sm' onClick={props.onNewSession}>
            <Plus />
            {t('New session')}
          </Button>
        </div>
      </div>

      <div className='p-4'>
        <div>
          <div className='mb-2 flex items-center justify-between gap-2'>
            <div className='text-muted-foreground text-xs font-medium'>
              {t('Available models')}
            </div>
            {(props.canManageCategories || props.canManageDescriptions) && (
              <div className='flex flex-wrap justify-end gap-1'>
                {props.canManageCategories && (
                  <Button
                    variant='ghost'
                    size='xs'
                    className='h-6 px-1.5 text-[11px]'
                    onClick={props.onManageCategories}
                  >
                    <Settings2 className='size-3.5' />
                    {t('Manage categories')}
                  </Button>
                )}
                {props.canManageDescriptions && (
                  <Button
                    variant='ghost'
                    size='xs'
                    className='h-6 px-1.5 text-[11px]'
                    onClick={props.onManageDescriptions}
                  >
                    <FileText className='size-3.5' />
                    {t('Manage descriptions')}
                  </Button>
                )}
              </div>
            )}
          </div>
          <div className='space-y-2'>
            {props.loading ? (
              <ModelSkeletons />
            ) : props.error ? (
              <SidebarNotice>
                {t('Unable to load model catalog.')}
              </SidebarNotice>
            ) : props.models.length === 0 ? (
              <SidebarNotice>
                {t('No models are configured for this creation type.')}
              </SidebarNotice>
            ) : (
              props.models.map((model) => (
                <ModelButton
                  key={model.id}
                  model={model}
                  mode={props.mode}
                  active={props.selectedModel?.id === model.id}
                  onClick={() => props.onModelChange(model)}
                />
              ))
            )}
          </div>
        </div>
      </div>

      <div className='mt-auto border-t p-4'>
        <div className='bg-muted/50 rounded-lg border p-3'>
          <div className='text-xs font-medium'>{t('Browsing mode')}</div>
          <p className='text-muted-foreground mt-1 text-xs leading-5'>
            {t(
              'The model catalog is synced with live configuration. Sign in before submitting a real task.'
            )}
          </p>
        </div>
      </div>
    </aside>
  )
}

function ModeButton(props: {
  mode: CreationMode
  count: number
  active: boolean
  onClick: () => void
}) {
  const { t } = useTranslation()
  const Icon =
    props.mode === 'chat'
      ? MessageSquare
      : props.mode === 'image'
        ? Image
        : Video
  const label =
    props.mode === 'chat'
      ? t('Chat')
      : props.mode === 'image'
        ? t('Image')
        : t('Video')

  return (
    <button
      type='button'
      role='tab'
      aria-selected={props.active}
      onClick={props.onClick}
      className={cn(
        'flex h-16 flex-col items-center justify-center gap-1 rounded-lg border text-xs transition-colors',
        props.active
          ? 'border-primary/40 bg-primary/10 text-primary'
          : 'border-border bg-card text-muted-foreground hover:bg-muted'
      )}
    >
      <Icon className='size-4' />
      <span>
        {label}
        {props.count > 0 ? ` ${props.count}` : ''}
      </span>
    </button>
  )
}

function ModelButton(props: {
  model: CreationModel
  mode: CreationMode
  active: boolean
  onClick: () => void
}) {
  const { t } = useTranslation()
  const costLabel = formatCreationModelCost(props.model.cost, t, props.mode)
  const tagLabels: Record<string, string> = {
    advanced: t('Advanced'),
    chat: t('Chat'),
    code: t('Code'),
    fast: t('Fast'),
    generation: t('Generation'),
    image: t('Image'),
    reasoning: t('Reasoning'),
    video: t('Video'),
  }
  const visibleTags = props.model.tags?.filter((tag) => tag !== 'async') ?? []

  return (
    <button
      type='button'
      onClick={props.onClick}
      className={cn(
        'w-full rounded-lg border p-3 text-left transition-colors',
        props.active
          ? 'border-primary/45 bg-primary/8'
          : 'border-border bg-card hover:bg-muted'
      )}
    >
      <div className='flex items-start gap-2.5'>
        <span
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-lg',
            props.active
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground'
          )}
        >
          <Bot className='size-4' />
        </span>
        <span className='min-w-0'>
          <span className='block truncate text-xs font-semibold'>
            {props.model.id}
          </span>
          <span className='text-muted-foreground mt-1 line-clamp-2 block text-[11px] leading-4'>
            {props.model.description || t('Ready for creation tasks.')}
          </span>
        </span>
      </div>
      <span className='text-primary mt-2 block truncate text-[11px] font-medium'>
        {t('Consumption')}: {costLabel}
      </span>
      {!!visibleTags.length && (
        <span className='mt-2 flex flex-wrap gap-1'>
          {visibleTags.slice(0, 2).map((tag) => (
            <Badge
              key={tag}
              variant='secondary'
              className='h-4 px-1.5 text-[10px]'
            >
              {tagLabels[tag] ?? tag}
            </Badge>
          ))}
        </span>
      )}
    </button>
  )
}

function ModelSkeletons() {
  return (
    <>
      <Skeleton className='h-[5.25rem] w-full' />
      <Skeleton className='h-[5.25rem] w-full' />
      <Skeleton className='h-[5.25rem] w-full' />
    </>
  )
}

function SidebarNotice(props: { children: React.ReactNode }) {
  return (
    <div className='text-muted-foreground rounded-lg border border-dashed px-3 py-5 text-center text-xs leading-5'>
      {props.children}
    </div>
  )
}
