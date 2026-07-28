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
import { useState } from 'react'
import {
  Clock3,
  FileAudio,
  FileImage,
  FileVideo,
  ListChecks,
  RefreshCw,
  Trash2,
  Upload,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  formatCreationCountdown,
  getCreationCountdownSeconds,
  getCreationReferencePreviewURL,
  getCreationReferenceURL,
  type CreationHistoryItem,
  type CreationImageReferences,
  type CreationVideoReferences,
} from '../session'
import type { CreationAsset, CreationMode, CreationResult } from '../types'

export type CreationInspectorView = 'assets' | 'history' | 'task'

type CreationInspectorProps = {
  className?: string
  mode: CreationMode
  value: CreationInspectorView
  assets: CreationAsset[]
  imageReferences: CreationImageReferences
  videoReferences: CreationVideoReferences
  historyItems: CreationHistoryItem[]
  result?: CreationResult
  now: number
  refreshingTask: boolean
  canUploadReferences: boolean
  onValueChange: (value: CreationInspectorView) => void
  onRequestUpload: () => void
  onSelectHistory: (item: CreationHistoryItem) => void
  onClearHistory: () => void
  onRefreshTask: () => void
  onRemoveAsset: (index: number) => void
  onRemoveImageReference: (index: number) => void
  onRemoveVideoReferenceImage: (index: number) => void
  onRemoveVideoReferenceVideo: (index: number) => void
  onRemoveVideoReferenceAudio: (index: number) => void
}

type InspectorAsset = {
  id: string
  kind: 'image' | 'video' | 'audio' | 'file'
  label: string
  previewUrl?: string
  onRemove: () => void
}

type PreviewMedia = Pick<InspectorAsset, 'kind' | 'label' | 'previewUrl'>

export function CreationInspector(props: CreationInspectorProps) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<PreviewMedia>()
  const inspectorAssets = getInspectorAssets(props, t)

  return (
    <>
      <Tabs
        value={props.value}
        onValueChange={(value) => {
          if (value === 'assets' || value === 'history' || value === 'task') {
            props.onValueChange(value)
          }
        }}
        className={cn(
          'min-h-[26rem] min-w-0 gap-0 overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm dark:border-white/10 dark:bg-[#101820]',
          props.className
        )}
      >
        <div className='border-b border-slate-200 px-3 dark:border-white/10'>
          <TabsList
            variant='line'
            aria-label={t('Creation workspace')}
            className='grid h-12 w-full grid-cols-3 p-0'
          >
            <TabsTrigger
              value='assets'
              className='h-full rounded-none text-xs data-active:text-cyan-700 dark:data-active:text-cyan-300'
            >
              <FileImage data-icon='inline-start' />
              {t('Assets')}
              {inspectorAssets.length > 0 && (
                <span className='rounded bg-slate-100 px-1 text-[10px] tabular-nums dark:bg-white/10'>
                  {inspectorAssets.length}
                </span>
              )}
            </TabsTrigger>
            <TabsTrigger
              value='history'
              className='h-full rounded-none text-xs data-active:text-cyan-700 dark:data-active:text-cyan-300'
            >
              <Clock3 data-icon='inline-start' />
              {t('History')}
            </TabsTrigger>
            <TabsTrigger
              value='task'
              className='h-full rounded-none text-xs data-active:text-cyan-700 dark:data-active:text-cyan-300'
            >
              <ListChecks data-icon='inline-start' />
              {t('Task')}
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value='assets' className='h-full min-h-0 p-3'>
          <AssetsPanel
            assets={inspectorAssets}
            canUpload={props.canUploadReferences}
            onRequestUpload={props.onRequestUpload}
            onPreview={setPreview}
          />
        </TabsContent>

        <TabsContent value='history' className='h-full min-h-0 p-3'>
          <HistoryPanel
            items={props.historyItems}
            onClearHistory={props.onClearHistory}
            onSelectHistory={props.onSelectHistory}
          />
        </TabsContent>

        <TabsContent value='task' className='h-full min-h-0 p-3'>
          <TaskPanel
            mode={props.mode}
            result={props.result}
            now={props.now}
            refreshingTask={props.refreshingTask}
            onRefreshTask={props.onRefreshTask}
          />
        </TabsContent>
      </Tabs>

      <Dialog
        open={!!preview}
        onOpenChange={(open) => {
          if (!open) setPreview(undefined)
        }}
      >
        <DialogContent className='sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>
              {preview?.label ?? t('Reference preview')}
            </DialogTitle>
          </DialogHeader>
          {preview && <InspectorMediaPreview preview={preview} />}
        </DialogContent>
      </Dialog>
    </>
  )
}

function AssetsPanel(props: {
  assets: InspectorAsset[]
  canUpload: boolean
  onRequestUpload: () => void
  onPreview: (asset: PreviewMedia) => void
}) {
  const { t } = useTranslation()

  return (
    <div className='flex h-full min-h-0 flex-col'>
      <div className='flex items-center justify-between gap-2'>
        <div>
          <h3 className='text-xs font-semibold'>{t('Reference assets')}</h3>
          <p className='text-muted-foreground mt-0.5 text-[11px]'>
            {props.assets.length
              ? t('{{count}} reference asset(s)', {
                  count: props.assets.length,
                })
              : t('No reference assets')}
          </p>
        </div>
        {props.canUpload && (
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={props.onRequestUpload}
          >
            <Upload data-icon='inline-start' />
            {t('Reference assets')}
          </Button>
        )}
      </div>

      {props.assets.length ? (
        <div className='mt-3 grid min-h-0 grid-cols-2 gap-2 overflow-y-auto pr-1'>
          {props.assets.map((asset) => (
            <AssetTile
              key={asset.id}
              asset={asset}
              onPreview={props.onPreview}
            />
          ))}
        </div>
      ) : (
        <Empty className='min-h-64 flex-1'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <FileImage />
            </EmptyMedia>
            <EmptyTitle>{t('Assets')}</EmptyTitle>
            <EmptyDescription>
              {t('No assets have been added to this session.')}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
    </div>
  )
}

function AssetTile(props: {
  asset: InspectorAsset
  onPreview: (asset: PreviewMedia) => void
}) {
  const { t } = useTranslation()
  const canPreview = !!props.asset.previewUrl

  return (
    <div className='group relative min-w-0 overflow-hidden rounded-md border border-slate-200 bg-slate-50 dark:border-white/10 dark:bg-white/[0.035]'>
      <button
        type='button'
        className='flex aspect-[4/3] w-full items-center justify-center overflow-hidden bg-slate-100 disabled:cursor-default dark:bg-black/20'
        disabled={!canPreview}
        aria-label={`${t('Open reference preview')}: ${props.asset.label}`}
        onClick={() => props.onPreview(props.asset)}
      >
        {props.asset.kind === 'image' && props.asset.previewUrl ? (
          <img
            src={props.asset.previewUrl}
            alt={props.asset.label}
            className='size-full object-cover transition-transform duration-200 group-hover:scale-[1.02]'
          />
        ) : (
          <AssetIcon kind={props.asset.kind} className='size-5' />
        )}
      </button>
      <div className='flex min-w-0 items-center gap-1.5 border-t border-slate-200 px-2 py-1.5 dark:border-white/10'>
        <AssetIcon
          kind={props.asset.kind}
          className='text-muted-foreground size-3 shrink-0'
        />
        <span className='min-w-0 flex-1 truncate text-[11px]'>
          {props.asset.label}
        </span>
        <Button
          type='button'
          variant='ghost'
          size='icon-xs'
          className='shrink-0'
          aria-label={`${t('Remove asset')}: ${props.asset.label}`}
          onClick={props.asset.onRemove}
        >
          <Trash2 />
        </Button>
      </div>
    </div>
  )
}

function HistoryPanel(props: {
  items: CreationHistoryItem[]
  onClearHistory: () => void
  onSelectHistory: (item: CreationHistoryItem) => void
}) {
  const { t } = useTranslation()

  if (!props.items.length) {
    return (
      <Empty className='min-h-64'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Clock3 />
          </EmptyMedia>
          <EmptyTitle>{t('No session history yet')}</EmptyTitle>
          <EmptyDescription>
            {t(
              'Completed creation tasks will appear here after you submit tasks.'
            )}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <div className='flex h-full min-h-0 flex-col'>
      <div className='flex items-center justify-between gap-2'>
        <div>
          <h3 className='text-xs font-semibold'>{t('History')}</h3>
          <p className='text-muted-foreground mt-0.5 text-[11px]'>
            {t('History is saved in this browser.')}
          </p>
        </div>
        <Button
          type='button'
          variant='ghost'
          size='icon-sm'
          aria-label={t('Clear history')}
          onClick={props.onClearHistory}
        >
          <Trash2 />
        </Button>
      </div>
      <div className='mt-3 min-h-0 flex-1 space-y-2 overflow-y-auto pr-1'>
        {props.items.map((item) => (
          <button
            key={item.id}
            type='button'
            onClick={() => props.onSelectHistory(item)}
            className='flex w-full min-w-0 gap-2.5 rounded-md border border-slate-200 bg-slate-50 p-2 text-left transition-colors hover:border-cyan-500/35 hover:bg-cyan-500/[0.04] dark:border-white/10 dark:bg-white/[0.035]'
          >
            <HistoryThumbnail item={item} />
            <span className='min-w-0 flex-1'>
              <span className='flex min-w-0 items-center justify-between gap-2'>
                <span className='min-w-0 truncate text-xs font-semibold'>
                  {item.model}
                </span>
                <StatusBadge status={item.result.status} />
              </span>
              <span className='text-muted-foreground mt-1 line-clamp-2 block text-[11px] leading-4'>
                {item.prompt}
              </span>
              <span className='text-muted-foreground mt-1.5 block text-[10px] tabular-nums'>
                {formatCreationTime(item.createdAt)}
              </span>
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}

function HistoryThumbnail(props: { item: CreationHistoryItem }) {
  const { t } = useTranslation()
  const imageUrl = props.item.result.imageUrl

  if (imageUrl) {
    return (
      <img
        src={imageUrl}
        alt={t('Generated image')}
        className='size-14 shrink-0 rounded-md border object-cover'
      />
    )
  }

  const Icon =
    props.item.mode === 'video'
      ? FileVideo
      : props.item.mode === 'image'
        ? FileImage
        : ListChecks
  return (
    <span className='bg-background/70 text-muted-foreground flex size-14 shrink-0 items-center justify-center rounded-md border'>
      <Icon className='size-4' />
    </span>
  )
}

function TaskPanel(props: {
  mode: CreationMode
  result?: CreationResult
  now: number
  refreshingTask: boolean
  onRefreshTask: () => void
}) {
  const { t } = useTranslation()

  if (!props.result) {
    return (
      <Empty className='min-h-64'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <ListChecks />
          </EmptyMedia>
          <EmptyTitle>{t('Task')}</EmptyTitle>
          <EmptyDescription>
            {props.mode === 'video'
              ? t('No video task yet')
              : props.mode === 'image'
                ? t('No image task yet')
                : t('No conversation yet')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  const progress = getResultProgress(props.result)
  const countdown = getCreationCountdownSeconds(
    props.result.createdAt,
    props.result.estimateSeconds,
    props.now
  )

  return (
    <div className='flex h-full min-h-0 flex-col'>
      <div className='rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-white/10 dark:bg-white/[0.035]'>
        <div className='flex min-w-0 items-start justify-between gap-2'>
          <div className='min-w-0'>
            <p className='text-muted-foreground text-[11px]'>
              {t('Current model')}
            </p>
            <h3 className='mt-0.5 truncate text-xs font-semibold'>
              {props.result.model}
            </h3>
          </div>
          <StatusBadge status={props.result.status} />
        </div>

        {typeof progress === 'number' && (
          <div className='mt-3'>
            <div className='mb-1.5 flex items-center justify-between text-[11px]'>
              <span className='text-muted-foreground'>{t('Processing')}</span>
              <span className='font-medium tabular-nums'>{progress}%</span>
            </div>
            <Progress value={progress} />
          </div>
        )}

        <dl className='mt-3 space-y-2 text-[11px]'>
          {props.result.taskId && (
            <TaskDetail label={t('Task')} value={props.result.taskId} />
          )}
          {(props.result.resolution || props.result.duration) && (
            <TaskDetail
              label={t('Creation workspace')}
              value={[
                props.result.resolution,
                props.result.duration && `${props.result.duration}s`,
              ]
                .filter(Boolean)
                .join(' · ')}
            />
          )}
          {countdown > 0 &&
            props.result.status !== 'completed' &&
            props.result.status !== 'failed' && (
              <TaskDetail
                label={t('Estimated remaining')}
                value={formatCreationCountdown(countdown)}
              />
            )}
        </dl>

        {props.result.error && (
          <p className='text-destructive border-destructive/25 bg-destructive/5 mt-3 rounded-md border p-2 text-[11px] leading-4'>
            {props.result.error}
          </p>
        )}

        {props.result.taskId &&
          props.result.status !== 'completed' &&
          props.result.status !== 'failed' && (
            <Button
              type='button'
              variant='outline'
              size='sm'
              className='mt-3 w-full'
              onClick={props.onRefreshTask}
              disabled={props.refreshingTask}
            >
              <RefreshCw
                data-icon='inline-start'
                className={cn(props.refreshingTask && 'animate-spin')}
              />
              {t('Refresh status')}
            </Button>
          )}
      </div>
    </div>
  )
}

function TaskDetail(props: { label: string; value: string }) {
  return (
    <div className='grid grid-cols-[5.5rem_minmax(0,1fr)] gap-2'>
      <dt className='text-muted-foreground'>{props.label}</dt>
      <dd
        className='truncate text-right font-medium tabular-nums'
        title={props.value}
      >
        {props.value}
      </dd>
    </div>
  )
}

function StatusBadge(props: { status: CreationResult['status'] }) {
  const { t } = useTranslation()
  const label = getStatusLabel(props.status, t)

  return (
    <Badge
      variant='outline'
      className={cn(
        'shrink-0 text-[10px]',
        props.status === 'completed' &&
          'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
        (props.status === 'queued' || props.status === 'processing') &&
          'border-cyan-500/30 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300',
        props.status === 'failed' &&
          'border-destructive/30 bg-destructive/10 text-destructive'
      )}
    >
      {label}
    </Badge>
  )
}

function InspectorMediaPreview(props: { preview: PreviewMedia }) {
  const { preview } = props

  if (!preview.previewUrl) return null
  if (preview.kind === 'image') {
    return (
      <img
        src={preview.previewUrl}
        alt={preview.label}
        className='max-h-[70vh] w-full rounded-md object-contain'
      />
    )
  }
  if (preview.kind === 'video') {
    return (
      <video
        src={preview.previewUrl}
        controls
        className='max-h-[70vh] w-full rounded-md bg-black'
      />
    )
  }
  if (preview.kind === 'audio') {
    return <audio src={preview.previewUrl} controls className='w-full' />
  }
  return null
}

function AssetIcon(props: {
  kind: InspectorAsset['kind']
  className?: string
}) {
  const Icon =
    props.kind === 'video'
      ? FileVideo
      : props.kind === 'audio'
        ? FileAudio
        : FileImage
  return <Icon className={props.className} />
}

function getInspectorAssets(
  props: Pick<
    CreationInspectorProps,
    | 'assets'
    | 'imageReferences'
    | 'mode'
    | 'onRemoveAsset'
    | 'onRemoveImageReference'
    | 'onRemoveVideoReferenceAudio'
    | 'onRemoveVideoReferenceImage'
    | 'onRemoveVideoReferenceVideo'
    | 'videoReferences'
  >,
  t: (key: string) => string
): InspectorAsset[] {
  const genericAssets = props.assets.map((asset, index) => ({
    id: `asset-${asset.id}`,
    kind: asset.type.startsWith('image/')
      ? ('image' as const)
      : ('file' as const),
    label: asset.name,
    previewUrl: asset.dataUrl,
    onRemove: () => props.onRemoveAsset(index),
  }))

  if (props.mode === 'image') {
    return [
      ...genericAssets,
      ...props.imageReferences.imageUrls.flatMap((reference, index) => {
        const url = getCreationReferenceURL(reference)
        if (!url) return []
        return [
          {
            id: `image-reference-${index}-${url}`,
            kind: 'image' as const,
            label: `${t('Reference image')} ${index + 1}`,
            previewUrl: getCreationReferencePreviewURL(reference),
            onRemove: () => props.onRemoveImageReference(index),
          },
        ]
      }),
    ]
  }

  if (props.mode !== 'video') return genericAssets

  const imageAssets = props.videoReferences.imageUrls.flatMap(
    (reference, index) => {
      const url = getCreationReferenceURL(reference)
      if (!url) return []
      return [
        {
          id: `video-image-reference-${index}-${url}`,
          kind: 'image' as const,
          label: `${t('Reference image')} ${index + 1}`,
          previewUrl: getCreationReferencePreviewURL(reference),
          onRemove: () => props.onRemoveVideoReferenceImage(index),
        },
      ]
    }
  )
  const videoAssets = props.videoReferences.videoUrls.flatMap(
    (reference, index) => {
      const url = getCreationReferenceURL(reference)
      if (!url) return []
      return [
        {
          id: `video-reference-${index}-${url}`,
          kind: 'video' as const,
          label: `${t('Reference video')} ${index + 1}`,
          previewUrl: getCreationReferencePreviewURL(reference),
          onRemove: () => props.onRemoveVideoReferenceVideo(index),
        },
      ]
    }
  )
  const audioAssets = props.videoReferences.audioUrls.flatMap(
    (reference, index) => {
      const url = getCreationReferenceURL(reference)
      if (!url) return []
      return [
        {
          id: `audio-reference-${index}-${url}`,
          kind: 'audio' as const,
          label: `${t('Reference audio')} ${index + 1}`,
          previewUrl: getCreationReferencePreviewURL(reference),
          onRemove: () => props.onRemoveVideoReferenceAudio(index),
        },
      ]
    }
  )

  return [...genericAssets, ...imageAssets, ...videoAssets, ...audioAssets]
}

function getResultProgress(result: CreationResult) {
  if (result.status === 'completed') return 100
  const raw = asRecord(result.raw)
  const data = asRecord(raw?.data)
  const task = asRecord(raw?.task)
  const progress = [raw?.progress, data?.progress, task?.progress]
    .map(toFiniteNumber)
    .find((value) => value !== undefined)
  return progress === undefined
    ? undefined
    : Math.max(0, Math.min(100, Math.round(progress)))
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object'
    ? (value as Record<string, unknown>)
    : undefined
}

function toFiniteNumber(value: unknown) {
  const number = typeof value === 'string' ? Number(value) : value
  return typeof number === 'number' && Number.isFinite(number)
    ? number
    : undefined
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

function formatCreationTime(value: number) {
  return new Date(value).toLocaleString()
}
