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
import { useMemo, useState } from 'react'
import {
  Bot,
  FileText,
  History,
  Image,
  MessageSquare,
  Plus,
  Search,
  Settings2,
  Sparkles,
  Video,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { CREATION_MODES } from '../constants'
import { formatCreationModelCost } from '../cost'
import type { CreationMode, CreationModel } from '../types'

type CreationSidebarProps = {
  mode: CreationMode
  models: CreationModel[]
  selectedModel?: CreationModel
  selectedResolution?: string
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
  const [query, setQuery] = useState('')
  const filteredModels = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase()
    if (!normalizedQuery) return props.models
    return props.models.filter((model) =>
      [model.id, model.description, ...(model.tags ?? [])].some((value) =>
        value?.toLocaleLowerCase().includes(normalizedQuery)
      )
    )
  }, [props.models, query])

  return (
    <aside className='flex min-w-0 flex-col border-b border-slate-200 bg-white/95 lg:sticky lg:top-16 lg:h-[calc(100svh-4rem)] lg:border-r lg:border-b-0 dark:border-white/10 dark:bg-[#0b1118]'>
      <div className='flex flex-col gap-3 border-b border-slate-200 p-4 dark:border-white/10'>
        <div className='flex items-start gap-3'>
          <span className='flex size-9 shrink-0 items-center justify-center rounded-md border border-cyan-500/25 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300'>
            <Sparkles className='size-4' />
          </span>
          <div className='min-w-0'>
            <h1 className='text-base font-semibold'>{t('Creation Center')}</h1>
            <p className='text-muted-foreground mt-0.5 text-xs leading-5'>
              {t(
                'Create chat, image, and video tasks with the models configured in your workspace.'
              )}
            </p>
          </div>
        </div>
        <div
          className='grid grid-cols-3 gap-1 rounded-md border border-slate-200 bg-slate-100/80 p-1 dark:border-white/10 dark:bg-white/[0.035]'
          role='tablist'
        >
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
            <History data-icon='inline-start' />
            {t('History')}
          </Button>
          <Button variant='outline' size='sm' onClick={props.onNewSession}>
            <Plus data-icon='inline-start' />
            {t('New session')}
          </Button>
        </div>
      </div>

      <div className='flex min-h-0 flex-1 flex-col p-3 lg:p-4'>
        <div className='mb-2 flex items-center justify-between gap-2'>
          <div className='text-muted-foreground text-xs font-medium'>
            {t('Available models')}
          </div>
          {(props.canManageCategories || props.canManageDescriptions) && (
            <div className='flex justify-end gap-1'>
              {props.canManageCategories && (
                <Button
                  variant='ghost'
                  size='icon-xs'
                  aria-label={t('Manage categories')}
                  onClick={props.onManageCategories}
                >
                  <Settings2 />
                </Button>
              )}
              {props.canManageDescriptions && (
                <Button
                  variant='ghost'
                  size='icon-xs'
                  aria-label={t('Manage descriptions')}
                  onClick={props.onManageDescriptions}
                >
                  <FileText />
                </Button>
              )}
            </div>
          )}
        </div>
        <div className='relative mb-3'>
          <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2' />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t('Search models')}
            aria-label={t('Search models')}
            className='h-9 rounded-md border-slate-200 bg-slate-50 pl-8 text-xs shadow-none dark:border-white/10 dark:bg-white/[0.035]'
          />
        </div>
        <div className='no-scrollbar flex min-h-0 gap-2 overflow-x-auto pb-1 lg:flex-1 lg:flex-col lg:overflow-x-hidden lg:overflow-y-auto lg:pr-1 lg:pb-0'>
          {props.loading ? (
            <ModelSkeletons />
          ) : props.error ? (
            <SidebarNotice>{t('Unable to load model catalog.')}</SidebarNotice>
          ) : filteredModels.length === 0 ? (
            <SidebarNotice>
              {t('No models are configured for this creation type.')}
            </SidebarNotice>
          ) : (
            filteredModels.map((model) => (
              <ModelButton
                key={model.id}
                model={model}
                mode={props.mode}
                active={props.selectedModel?.id === model.id}
                selectedResolution={
                  props.selectedModel?.id === model.id
                    ? props.selectedResolution
                    : undefined
                }
                onClick={() => props.onModelChange(model)}
              />
            ))
          )}
        </div>
      </div>

      <div className='mt-auto hidden border-t border-slate-200 p-4 lg:block dark:border-white/10'>
        <div className='rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-white/10 dark:bg-white/[0.035]'>
          <div className='flex items-center gap-2 text-xs font-medium'>
            <span className='size-1.5 rounded-full bg-cyan-500' />
            {t('Browsing mode')}
          </div>
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
        'flex min-h-11 items-center justify-center gap-1.5 rounded-md px-2 text-xs font-medium transition-colors',
        props.active
          ? 'bg-white text-cyan-700 shadow-sm dark:bg-white/10 dark:text-cyan-300'
          : 'text-muted-foreground hover:text-foreground hover:bg-white/70 dark:hover:bg-white/[0.06]'
      )}
    >
      <Icon className='size-4' />
      <span>{label}</span>
      {props.count > 0 && (
        <span className='bg-muted text-muted-foreground rounded px-1 text-[10px] tabular-nums'>
          {props.count}
        </span>
      )}
    </button>
  )
}

function ModelButton(props: {
  model: CreationModel
  mode: CreationMode
  active: boolean
  selectedResolution?: string
  onClick: () => void
}) {
  const { t } = useTranslation()
  const costLabel = formatCreationModelCost(
    props.model.cost,
    t,
    props.mode,
    props.selectedResolution
  )
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
        'group min-w-[15rem] rounded-md border bg-slate-50 p-3 text-left transition-colors lg:w-full lg:min-w-0 dark:bg-white/[0.035]',
        props.active
          ? 'border-cyan-500/60 bg-cyan-500/[0.07] ring-1 ring-cyan-500/15'
          : 'border-slate-200 hover:border-cyan-500/30 hover:bg-cyan-500/[0.03] dark:border-white/10'
      )}
    >
      <div className='flex items-start gap-2.5'>
        <span
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-lg',
            props.active
              ? 'bg-cyan-600 text-white dark:bg-cyan-500 dark:text-slate-950'
              : 'bg-muted text-muted-foreground group-hover:text-foreground'
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
      <span className='mt-2 block truncate text-[11px] font-medium text-cyan-700 tabular-nums dark:text-cyan-300'>
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
      <Skeleton className='h-[5.25rem] min-w-[15rem] lg:w-full lg:min-w-0' />
      <Skeleton className='h-[5.25rem] min-w-[15rem] lg:w-full lg:min-w-0' />
      <Skeleton className='h-[5.25rem] min-w-[15rem] lg:w-full lg:min-w-0' />
    </>
  )
}

function SidebarNotice(props: { children: React.ReactNode }) {
  return (
    <div className='text-muted-foreground min-w-[15rem] rounded-md border border-dashed px-3 py-5 text-center text-xs leading-5 lg:min-w-0'>
      {props.children}
    </div>
  )
}
