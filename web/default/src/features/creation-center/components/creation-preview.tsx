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
import { Image, MessageSquare, RefreshCw, Timer, Video } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  formatCreationCountdown,
  getCreationCountdownSeconds,
  getCreationTimedOut,
} from '../session'
import type { CreationMode, CreationModel, CreationResult } from '../types'

type CreationPreviewProps = {
  className?: string
  mode: CreationMode
  model?: CreationModel
  result?: CreationResult
  now: number
  submitting: boolean
  refreshingTask: boolean
  onRefreshTask: () => void
}

export function CreationPreview(props: CreationPreviewProps) {
  const { t } = useTranslation()

  return (
    <section
      className={cn(
        'min-h-[28rem] min-w-0 overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm dark:border-white/10 dark:bg-[#101820]',
        props.className
      )}
    >
      <div className='flex min-h-12 flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-2 dark:border-white/10'>
        <div className='min-w-0'>
          <h2 className='text-sm font-semibold'>{t('Preview')}</h2>
          {props.model && (
            <p className='text-muted-foreground mt-0.5 truncate text-xs'>
              {props.model.id}
            </p>
          )}
        </div>
        {props.result && (
          <span className='text-muted-foreground rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-[11px] font-medium dark:border-white/10 dark:bg-white/[0.035]'>
            {getStatusLabel(props.result.status, t)}
          </span>
        )}
      </div>

      <div className='flex min-h-[calc(28rem-3rem)] items-center justify-center bg-slate-50/70 p-4 sm:p-6 dark:bg-[#0b1118]'>
        {props.submitting ? (
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
            className='mx-auto max-h-[48rem] w-auto max-w-full rounded-md object-contain'
          />
        ) : (
          <EmptyMediaResult
            title={
              props.result.status === 'completed'
                ? t('Image task returned no image URL')
                : t('Image task is waiting for a result')
            }
          />
        )}
        <div className='text-muted-foreground mt-4 space-y-1 text-xs leading-5'>
          <div>
            {props.result.model} · {statusLabel}
          </div>
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
          className='mx-auto max-h-[48rem] w-auto max-w-full rounded-md bg-black'
        />
      ) : props.result.outputText ? (
        <div className='bg-muted/40 max-h-[30rem] overflow-auto rounded-lg p-4 text-left text-sm leading-6 whitespace-pre-wrap'>
          {props.result.outputText}
        </div>
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
    <Empty className='bg-background/60 min-h-56 rounded-lg border'>
      <EmptyHeader>
        <EmptyMedia variant='icon'>
          <Image />
        </EmptyMedia>
        <EmptyTitle>{props.title}</EmptyTitle>
      </EmptyHeader>
    </Empty>
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
    <Empty className='max-w-md'>
      <EmptyHeader>
        <EmptyMedia variant='icon' className='size-12'>
          <Icon className='size-5' />
        </EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>
          {props.model
            ? t(
                'Write a prompt below, add reference images for video if needed, and sign in before submitting a real task.'
              )
            : t('Select a configured model from the sidebar to begin.')}
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  )
}
