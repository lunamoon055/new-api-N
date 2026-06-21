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
  Clock3,
  FileImage,
  Image,
  MessageSquare,
  RefreshCw,
  Timer,
  Trash2,
  Video,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  formatCreationCountdown,
  getCreationCountdownSeconds,
  getCreationTimedOut,
  type CreationHistoryItem,
} from '../session'
import type {
  CreationAsset,
  CreationMode,
  CreationModel,
  CreationResult,
  CreationView,
} from '../types'

type CreationPreviewProps = {
  mode: CreationMode
  model?: CreationModel
  view: CreationView
  assets: CreationAsset[]
  historyItems: CreationHistoryItem[]
  result?: CreationResult
  now: number
  submitting: boolean
  refreshingTask: boolean
  onViewChange: (view: CreationView) => void
  onSelectHistory: (item: CreationHistoryItem) => void
  onClearHistory: () => void
  onRefreshTask: () => void
  onRemoveAsset: (index: number) => void
}

export function CreationPreview(props: CreationPreviewProps) {
  const { t } = useTranslation()

  return (
    <section className='bg-card flex min-h-[22rem] min-w-0 flex-col rounded-lg border'>
      <div className='flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('Creation workspace')}</h2>
        <div className='flex gap-1'>
          {(['preview', 'assets', 'history'] as CreationView[]).map((item) => (
            <Button
              key={item}
              variant={props.view === item ? 'secondary' : 'ghost'}
              size='xs'
              onClick={() => props.onViewChange(item)}
            >
              {item === 'preview'
                ? t('Preview')
                : item === 'assets'
                  ? t('Assets')
                  : t('History')}
            </Button>
          ))}
        </div>
      </div>

      <div className='flex min-h-0 flex-1 items-center justify-center p-4'>
        {props.view === 'assets' ? (
          <AssetPreview
            assets={props.assets}
            onRemoveAsset={props.onRemoveAsset}
          />
        ) : props.view === 'history' ? (
          <HistoryPreview
            items={props.historyItems}
            onClearHistory={props.onClearHistory}
            onSelectHistory={props.onSelectHistory}
          />
        ) : props.submitting ? (
          <SubmittingPreview mode={props.mode} />
        ) : props.result ? (
          <ResultPreview
            result={props.result}
            now={props.now}
            refreshingTask={props.refreshingTask}
            onRefreshTask={props.onRefreshTask}
          />
        ) : (
          <EmptyPreview mode={props.mode} model={props.model} />
        )}
      </div>
    </section>
  )
}

function SubmittingPreview(props: { mode: CreationMode }) {
  const { t } = useTranslation()
  const title =
    props.mode === 'video'
      ? t('Submitting async media task')
      : t('Submitting creation task')

  return (
    <div className='max-w-md text-center'>
      <div className='bg-primary/10 text-primary mx-auto flex size-14 items-center justify-center rounded-lg'>
        <RefreshCw className='size-6 animate-spin' />
      </div>
      <h3 className='mt-4 text-sm font-semibold'>{title}</h3>
      <p className='text-muted-foreground mt-2 text-xs leading-5'>
        {t('Please keep this page open while the request is being submitted.')}
      </p>
    </div>
  )
}

function ResultPreview(props: {
  result: CreationResult
  now: number
  refreshingTask: boolean
  onRefreshTask: () => void
}) {
  const { t } = useTranslation()
  const statusLabel = getStatusLabel(props.result.status, t)
  const countdownSeconds = getCreationCountdownSeconds(
    props.result.createdAt,
    props.result.estimateSeconds,
    props.now
  )
  const timedOut = getCreationTimedOut(
    props.result.createdAt,
    props.result.estimateSeconds,
    props.now
  )

  if (props.result.status === 'failed') {
    return (
      <div className='max-w-lg text-center'>
        <div className='bg-destructive/10 text-destructive mx-auto flex size-14 items-center justify-center rounded-lg'>
          <MessageSquare className='size-6' />
        </div>
        <h3 className='mt-4 text-sm font-semibold'>{t('Task failed')}</h3>
        <p className='text-muted-foreground mt-2 text-xs leading-5'>
          {props.result.error || t('The upstream provider returned an error.')}
        </p>
      </div>
    )
  }

  if (props.result.mode === 'chat') {
    return (
      <div className='w-full max-w-2xl'>
        <div className='text-muted-foreground mb-2 text-xs'>
          {props.result.model} · {statusLabel}
        </div>
        <div className='bg-muted/40 max-h-[30rem] overflow-auto rounded-lg p-4 text-sm leading-6 whitespace-pre-wrap'>
          {props.result.outputText || t('No text content was returned.')}
        </div>
      </div>
    )
  }

  if (props.result.mode === 'image') {
    return (
      <div className='w-full max-w-2xl text-center'>
        {props.result.imageUrl ? (
          <img
            src={props.result.imageUrl}
            alt={t('Generated image')}
            className='mx-auto max-h-[31rem] w-auto max-w-full rounded-lg object-contain'
          />
        ) : (
          <EmptyMediaResult title={t('Image task returned no image URL')} />
        )}
        {props.result.outputText && (
          <p className='text-muted-foreground mx-auto mt-3 max-w-xl text-xs leading-5'>
            {props.result.outputText}
          </p>
        )}
      </div>
    )
  }

  return (
    <div className='w-full max-w-2xl text-center'>
      {props.result.videoUrl ? (
        <video
          controls
          src={props.result.videoUrl}
          className='mx-auto max-h-[31rem] w-auto max-w-full rounded-lg'
        />
      ) : (
        <EmptyMediaResult title={t('Video task is waiting for a result')} />
      )}
      <div className='text-muted-foreground mt-4 space-y-1 text-xs leading-5'>
        <div>
          {props.result.model} · {statusLabel}
        </div>
        {!props.result.videoUrl && props.result.status !== 'completed' && (
          <div
            className={cn(
              'inline-flex items-center justify-center gap-1.5',
              timedOut ? 'text-amber-600 dark:text-amber-400' : 'text-primary'
            )}
          >
            <Timer className='size-3.5' />
            {countdownSeconds > 0
              ? `${t('Estimated remaining')} ${formatCreationCountdown(
                  countdownSeconds
                )}`
              : t(
                  'Generation is taking longer than expected. You can refresh later or check the task log.'
                )}
          </div>
        )}
        {(props.result.resolution || props.result.duration) && (
          <div>
            {[
              formatCreationResolution(props.result.resolution),
              props.result.duration && `${props.result.duration}s`,
            ]
              .filter(Boolean)
              .join(' · ')}
          </div>
        )}
        {props.result.taskId && <div>{props.result.taskId}</div>}
      </div>
      {props.result.taskId && (
        <Button
          variant='outline'
          size='sm'
          className='mt-4'
          onClick={props.onRefreshTask}
          disabled={props.refreshingTask}
        >
          <RefreshCw
            className={cn('size-4', props.refreshingTask && 'animate-spin')}
          />
          {t('Refresh status')}
        </Button>
      )}
    </div>
  )
}

function EmptyMediaResult(props: { title: string }) {
  return (
    <div className='bg-muted/40 text-muted-foreground flex min-h-56 items-center justify-center rounded-lg border border-dashed px-6 text-sm'>
      {props.title}
    </div>
  )
}

function getStatusLabel(
  status: CreationResult['status'],
  t: (key: string) => string
) {
  switch (status) {
    case 'queued':
      return t('Queued')
    case 'processing':
      return t('Processing')
    case 'completed':
      return t('Completed')
    case 'failed':
      return t('Failed')
    default:
      return t('Unknown status')
  }
}

function formatCreationResolution(resolution?: string) {
  switch (resolution) {
    case '1080p':
      return '1080'
    case '2k':
      return '2K'
    case '4k':
      return '4K'
    default:
      return resolution
  }
}

function formatCreationTime(value: number) {
  return new Date(value).toLocaleString()
}

function EmptyPreview(props: { mode: CreationMode; model?: CreationModel }) {
  const { t } = useTranslation()
  const Icon =
    props.mode === 'chat'
      ? MessageSquare
      : props.mode === 'image'
        ? Image
        : Video
  const title =
    props.mode === 'chat'
      ? t('No conversation yet')
      : props.mode === 'image'
        ? t('No image task yet')
        : t('No video task yet')

  return (
    <div className='max-w-md text-center'>
      <div className='bg-muted text-muted-foreground mx-auto flex size-14 items-center justify-center rounded-lg'>
        <Icon className='size-6' />
      </div>
      <h3 className='mt-4 text-sm font-semibold'>{title}</h3>
      <p className='text-muted-foreground mt-2 text-xs leading-5'>
        {props.model
          ? t(
              'Write a prompt below, add reference images for video if needed, and sign in before submitting a real task.'
            )
          : t('Select a configured model from the sidebar to begin.')}
      </p>
    </div>
  )
}

function AssetPreview(props: {
  assets: CreationAsset[]
  onRemoveAsset: (index: number) => void
}) {
  const { t } = useTranslation()

  if (!props.assets.length) {
    return (
      <div className='text-muted-foreground text-center text-xs'>
        {t('No assets have been added to this session.')}
      </div>
    )
  }

  return (
    <div className='grid w-full gap-2 sm:grid-cols-2'>
      {props.assets.map((asset, index) => (
        <div
          key={asset.id}
          className='bg-muted/40 flex min-w-0 items-center gap-3 rounded-lg border p-3'
        >
          <FileImage className='text-primary size-4 shrink-0' />
          <span className='min-w-0 flex-1 truncate text-xs'>{asset.name}</span>
          <Button
            type='button'
            variant='ghost'
            size='icon-xs'
            className='ml-auto shrink-0'
            aria-label={`${t('Remove asset')}: ${asset.name}`}
            onClick={() => props.onRemoveAsset(index)}
          >
            <Trash2 className='size-3.5' />
          </Button>
        </div>
      ))}
    </div>
  )
}

function HistoryPreview(props: {
  items: CreationHistoryItem[]
  onClearHistory: () => void
  onSelectHistory: (item: CreationHistoryItem) => void
}) {
  const { t } = useTranslation()

  if (props.items.length) {
    return (
      <div className='flex h-full w-full flex-col gap-3'>
        <div className='flex items-center justify-between gap-3'>
          <div>
            <h3 className='text-sm font-semibold'>{t('History')}</h3>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t('History is saved in this browser.')}
            </p>
          </div>
          <Button variant='ghost' size='sm' onClick={props.onClearHistory}>
            <Trash2 className='size-4' />
            {t('Clear history')}
          </Button>
        </div>
        <div className='min-h-0 flex-1 space-y-2 overflow-auto pr-1'>
          {props.items.map((item) => (
            <button
              key={item.id}
              type='button'
              onClick={() => props.onSelectHistory(item)}
              className='border-border bg-muted/30 hover:bg-muted flex w-full min-w-0 flex-col gap-2 rounded-lg border p-3 text-left transition-colors'
            >
              <span className='flex min-w-0 items-center justify-between gap-2'>
                <span className='min-w-0 truncate text-xs font-semibold'>
                  {item.model}
                </span>
                <Badge variant='secondary' className='shrink-0 text-[10px]'>
                  {getStatusLabel(item.result.status, t)}
                </Badge>
              </span>
              <span className='text-muted-foreground line-clamp-2 text-xs leading-5'>
                {item.prompt}
              </span>
              <span className='text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px]'>
                <span>{formatCreationTime(item.createdAt)}</span>
                {item.result.taskId && <span>{item.result.taskId}</span>}
              </span>
            </button>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className='max-w-md text-center'>
      <Clock3 className='text-muted-foreground mx-auto size-8' />
      <h3 className='mt-3 text-sm font-semibold'>
        {t('No session history yet')}
      </h3>
      <p className='text-muted-foreground mt-2 text-xs leading-5'>
        {t('Completed creation tasks will appear here after you submit tasks.')}
      </p>
    </div>
  )
}
