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
import { Badge } from '@/components/ui/badge'
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
import {
  getVideoGenerationWaitingPhase,
  type VideoGenerationWaitingPhase,
} from '../lib/preview-state'
import type { CreationMode, CreationModel, CreationResult } from '../types'
import './creation-video-waiting.css'

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
  const videoWaitingPhase = getVideoGenerationWaitingPhase({
    mode: props.mode,
    submitting: props.submitting,
    result: props.result,
  })

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

      <div
        className={cn(
          'min-h-[calc(28rem-3rem)] bg-slate-50/70 dark:bg-[#0b1118]',
          videoWaitingPhase
            ? 'relative'
            : 'flex items-center justify-center p-4 sm:p-6'
        )}
      >
        {videoWaitingPhase ? (
          <VideoGenerationWaiting
            phase={videoWaitingPhase}
            result={props.result}
            now={props.now}
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

function VideoGenerationWaiting(props: {
  phase: VideoGenerationWaitingPhase
  result?: CreationResult
  now: number
}) {
  const { t } = useTranslation()
  const requestAccepted = props.phase !== 'submitting'
  const countdownSeconds = props.result
    ? getCreationCountdownSeconds(
        props.result.createdAt,
        props.result.estimateSeconds,
        props.now
      )
    : 0
  const timedOut = props.result
    ? getCreationTimedOut(
        props.result.createdAt,
        props.result.estimateSeconds,
        props.now
      )
    : false
  const hasEstimate = Boolean(
    props.result?.createdAt && props.result.estimateSeconds
  )
  const currentStageLabel =
    props.phase === 'queued' ? t('Queued') : t('Processing')
  const stages = [
    {
      label: requestAccepted ? t('Submitted') : t('Submitting'),
      state: requestAccepted ? 'done' : 'active',
    },
    {
      label: currentStageLabel,
      state: requestAccepted ? 'active' : 'pending',
    },
    {
      label: t('Completed'),
      state: 'pending',
    },
  ] as const

  return (
    <div className='creation-video-waiting relative isolate flex min-h-[calc(28rem-3rem)] w-full items-center justify-center overflow-hidden px-5 py-10 sm:px-8'>
      <div
        aria-hidden='true'
        className='pointer-events-none absolute inset-0 overflow-hidden'
      >
        <div className='creation-video-waiting__orb top-[18%] left-[18%] bg-cyan-300 dark:bg-cyan-600' />
        <div className='creation-video-waiting__orb right-[12%] bottom-[12%] bg-violet-300 [animation-delay:-5s] dark:bg-violet-700' />
        <svg
          className='creation-video-waiting__wave creation-video-waiting__wave--back opacity-55 dark:opacity-45'
          viewBox='0 0 1200 420'
          preserveAspectRatio='none'
        >
          <defs>
            <linearGradient
              id='creation-aurora-back'
              x1='0'
              y1='0'
              x2='1'
              y2='0'
            >
              <stop offset='0%' stopColor='#67e8f9' stopOpacity='0' />
              <stop offset='24%' stopColor='#38bdf8' stopOpacity='0.55' />
              <stop offset='68%' stopColor='#a78bfa' stopOpacity='0.5' />
              <stop offset='100%' stopColor='#c4b5fd' stopOpacity='0' />
            </linearGradient>
            <filter
              id='creation-aurora-back-blur'
              x='-20%'
              y='-40%'
              width='140%'
              height='180%'
            >
              <feGaussianBlur stdDeviation='22' />
            </filter>
          </defs>
          <path
            d='M-100 270 C120 430 220 65 455 215 S760 85 970 255 S1210 345 1330 155'
            fill='none'
            stroke='url(#creation-aurora-back)'
            strokeWidth='72'
            filter='url(#creation-aurora-back-blur)'
          />
        </svg>
        <svg
          className='creation-video-waiting__wave creation-video-waiting__wave--front opacity-75 dark:opacity-60'
          viewBox='0 0 1200 420'
          preserveAspectRatio='none'
        >
          <defs>
            <linearGradient
              id='creation-aurora-front'
              x1='0'
              y1='0'
              x2='1'
              y2='0'
            >
              <stop offset='0%' stopColor='#bae6fd' stopOpacity='0' />
              <stop offset='22%' stopColor='#22d3ee' stopOpacity='0.62' />
              <stop offset='60%' stopColor='#e0e7ff' stopOpacity='0.48' />
              <stop offset='82%' stopColor='#8b5cf6' stopOpacity='0.48' />
              <stop offset='100%' stopColor='#ddd6fe' stopOpacity='0' />
            </linearGradient>
            <filter
              id='creation-aurora-front-blur'
              x='-20%'
              y='-30%'
              width='140%'
              height='160%'
            >
              <feGaussianBlur stdDeviation='8' />
            </filter>
          </defs>
          <path
            d='M-120 315 C115 410 240 115 465 245 S760 125 945 275 S1180 365 1320 195'
            fill='none'
            stroke='url(#creation-aurora-front)'
            strokeWidth='18'
            filter='url(#creation-aurora-front-blur)'
          />
          <path
            d='M-120 315 C115 410 240 115 465 245 S760 125 945 275 S1180 365 1320 195'
            fill='none'
            stroke='url(#creation-aurora-front)'
            strokeWidth='2'
          />
        </svg>
      </div>

      <div className='relative z-10 flex w-full max-w-md flex-col items-center gap-5 text-center'>
        <div className='creation-video-waiting__icon flex size-16 items-center justify-center rounded-2xl border border-white/75 bg-white/60 text-sky-500 backdrop-blur-md dark:border-white/15 dark:bg-white/10 dark:text-cyan-300'>
          <Video className='size-7' strokeWidth={1.8} />
        </div>

        <div aria-live='polite' aria-atomic='true' className='grid gap-2'>
          <h3 className='text-base font-semibold tracking-tight sm:text-lg'>
            {t('Video is being generated')}
          </h3>
          <p className='text-muted-foreground text-xs leading-5 sm:text-sm'>
            {props.phase === 'submitting'
              ? t(
                  'Please keep this page open while the request is being submitted.'
                )
              : t('The model is processing the scene. Please wait.')}
          </p>
        </div>

        {hasEstimate &&
          (countdownSeconds > 0 ? (
            <Badge
              variant='secondary'
              className='border-border/60 bg-background/65 h-7 gap-1.5 border px-3 font-normal shadow-sm backdrop-blur-md'
            >
              <Timer className='size-3.5!' />
              <span className='tabular-nums'>
                {t('Estimated remaining')}{' '}
                {formatCreationCountdown(countdownSeconds)}
              </span>
            </Badge>
          ) : timedOut ? (
            <p className='border-amber-500/20 bg-amber-500/10 text-amber-700 max-w-sm rounded-full border px-3 py-1.5 text-[11px] leading-4 dark:text-amber-300'>
              {t(
                'Generation is taking longer than expected. You can refresh later or check the task log.'
              )}
            </p>
          ) : null)}

        <div
          className='relative grid w-full max-w-sm grid-cols-3'
          aria-label={t('Processing')}
        >
          <div
            aria-hidden='true'
            className='bg-border/70 absolute top-[5px] right-[16.667%] left-[16.667%] h-px'
          >
            <div
              className={cn(
                'h-full bg-cyan-500 transition-[width] duration-500 motion-reduce:transition-none',
                requestAccepted ? 'w-1/2' : 'w-0'
              )}
            />
          </div>
          {stages.map((stage) => (
            <div
              key={stage.label}
              className='relative z-10 flex flex-col items-center gap-2'
            >
              <span
                aria-hidden='true'
                className={cn(
                  'size-2.5 rounded-full border-2 transition-colors motion-reduce:transition-none',
                  stage.state === 'active' &&
                    'border-cyan-500 bg-cyan-500 ring-4 ring-cyan-500/10',
                  stage.state === 'done' && 'border-cyan-500 bg-cyan-500',
                  stage.state === 'pending' &&
                    'border-muted-foreground/30 bg-background'
                )}
              />
              <span
                className={cn(
                  'text-[11px] font-medium',
                  stage.state === 'active'
                    ? 'text-cyan-700 dark:text-cyan-300'
                    : stage.state === 'done'
                      ? 'text-foreground/75'
                      : 'text-muted-foreground'
                )}
              >
                {stage.label}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
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
