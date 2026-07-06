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
import { useMemo, useRef, useState } from 'react'
import {
  FileAudio,
  FileImage,
  FileVideo,
  RefreshCw,
  Send,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Field, FieldLabel } from '@/components/ui/field'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Textarea } from '@/components/ui/textarea'
import { ImageReferenceFields } from '../image-reference-fields'
import {
  getPromptMentionTrigger,
  type PromptMentionTrigger,
} from '../lib/prompt-mentions'
import {
  CREATION_IMAGE_ASPECT_RATIO_OPTIONS,
  type CreationImageAspectRatio,
  type CreationImageOptions,
  type CreationImageReferences,
  getCreationReferenceURL,
  normalizeCreationVideoReferences,
  type CreationAspectRatio,
  type CreationDuration,
  type CreationResolution,
  type CreationVideoCapability,
  type CreationVideoOptions,
  type CreationVideoReferenceMode,
  type CreationVideoReferences,
  type DurationOption,
  type ResolutionOption,
} from '../session'
import type { CreationAsset, CreationMode, CreationModel } from '../types'
import { VideoReferenceFields } from '../video-reference-fields'

type ComposerProps = {
  prompt: string
  assets: CreationAsset[]
  authenticated: boolean
  mode: CreationMode
  model?: CreationModel
  imageOptions: CreationImageOptions
  imageReferences: CreationImageReferences
  imageReferencesSupported: boolean
  videoOptions: CreationVideoOptions
  videoReferences: CreationVideoReferences
  videoCapabilities?: CreationVideoCapability
  resolutionOptions: ResolutionOption[]
  durationOptions: DurationOption[]
  submitting: boolean
  sessionNumber: number
  onPromptChange: (value: string) => void
  onImageOptionsChange: (options: CreationImageOptions) => void
  onImageReferenceFilesSelected: (files: File[]) => void
  onRemoveImageReferenceImage: (index: number) => void
  onVideoOptionsChange: (options: CreationVideoOptions) => void
  onVideoReferencesChange: (references: CreationVideoReferences) => void
  onVideoReferenceFilesSelected: (files: File[]) => void
  onRemoveVideoReferenceImage: (index: number) => void
  onRemoveVideoReferenceVideo: (index: number) => void
  onRemoveVideoReferenceAudio: () => void
  onRemoveAsset: (index: number) => void
  onSubmit: () => void
}

export function Composer(props: ComposerProps) {
  const { t } = useTranslation()
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [mentionTrigger, setMentionTrigger] =
    useState<PromptMentionTrigger | null>(null)
  const canSubmit = !!props.prompt.trim() && !!props.model && !props.submitting
  const referenceMentionItems = useMemo(
    () =>
      props.mode === 'video' && props.videoCapabilities?.kind === 'video2'
        ? getVideoReferenceMentionItems(props.videoReferences, t)
        : [],
    [props.mode, props.videoCapabilities?.kind, props.videoReferences, t]
  )
  const mentionSuggestions = mentionTrigger
    ? referenceMentionItems.filter((item) =>
        matchesReferenceMentionQuery(item, mentionTrigger.query)
      )
    : []
  const showReferenceMentions =
    !!mentionTrigger && mentionSuggestions.length > 0

  const updateMentionTrigger = (
    value: string,
    caretPosition: number | null | undefined
  ) => {
    setMentionTrigger(
      props.mode === 'video' && props.videoCapabilities?.kind === 'video2'
        ? getPromptMentionTrigger(value, caretPosition)
        : null
    )
  }

  const insertReferenceMention = (item: ReferenceMentionItem) => {
    const trigger =
      mentionTrigger ??
      getPromptMentionTrigger(
        props.prompt,
        textareaRef.current?.selectionStart ?? props.prompt.length
      )
    const token = `@${item.token} `
    const start = trigger?.start ?? props.prompt.length
    const end = trigger?.end ?? props.prompt.length
    const nextPrompt = `${props.prompt.slice(0, start)}${token}${props.prompt.slice(end)}`
    const nextCaret = start + token.length

    props.onPromptChange(nextPrompt)
    setMentionTrigger(null)
    requestAnimationFrame(() => {
      textareaRef.current?.focus()
      textareaRef.current?.setSelectionRange(nextCaret, nextCaret)
    })
  }

  return (
    <section className='bg-card rounded-lg border p-3'>
      <div className='flex min-w-0 items-start gap-3'>
        <div className='min-w-0 flex-1'>
          <div className='relative'>
            <Textarea
              ref={textareaRef}
              aria-label={t('Prompt')}
              value={props.prompt}
              maxLength={5000}
              onChange={(event) => {
                props.onPromptChange(event.target.value)
                updateMentionTrigger(
                  event.target.value,
                  event.target.selectionStart
                )
              }}
              onClick={(event) =>
                updateMentionTrigger(
                  event.currentTarget.value,
                  event.currentTarget.selectionStart
                )
              }
              onKeyUp={(event) =>
                updateMentionTrigger(
                  event.currentTarget.value,
                  event.currentTarget.selectionStart
                )
              }
              onKeyDown={(event) => {
                if (event.key === 'Escape' && mentionTrigger) {
                  event.preventDefault()
                  setMentionTrigger(null)
                  return
                }
                if (
                  event.key === 'Enter' &&
                  !event.shiftKey &&
                  !event.nativeEvent.isComposing &&
                  showReferenceMentions
                ) {
                  event.preventDefault()
                  insertReferenceMention(mentionSuggestions[0])
                  return
                }
                if (
                  event.key !== 'Enter' ||
                  event.shiftKey ||
                  event.nativeEvent.isComposing
                ) {
                  return
                }
                event.preventDefault()
                if (canSubmit) props.onSubmit()
              }}
              placeholder={t(
                'Describe the task you want the selected model to complete...'
              )}
              className='min-h-20 resize-none border-0 px-0 py-1 shadow-none focus-visible:ring-0'
            />
            {showReferenceMentions && (
              <div className='bg-popover text-popover-foreground absolute top-full right-0 left-0 z-20 mt-1 max-h-48 overflow-auto rounded-lg border p-1 shadow-md'>
                {mentionSuggestions.map((item) => (
                  <button
                    key={item.id}
                    type='button'
                    className='hover:bg-accent hover:text-accent-foreground flex w-full min-w-0 items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors'
                    aria-label={item.label}
                    onMouseDown={(event) => {
                      event.preventDefault()
                      insertReferenceMention(item)
                    }}
                  >
                    {getReferenceMentionIcon(item.kind)}
                    <span className='min-w-0 flex-1 truncate'>
                      {item.label}
                    </span>
                    <span className='text-muted-foreground shrink-0 text-xs'>
                      @{item.token}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
          {!!props.assets.length && (
            <div className='mt-2 flex flex-wrap gap-1.5'>
              {props.assets.map((asset, index) => (
                <button
                  key={asset.id}
                  type='button'
                  onClick={() => props.onRemoveAsset(index)}
                  className='bg-muted text-muted-foreground hover:bg-muted/80 inline-flex max-w-full items-center gap-1.5 rounded-md border px-2 py-1 text-[11px] transition-colors'
                  aria-label={`${t('Remove asset')}: ${asset.name}`}
                >
                  <FileImage className='size-3 shrink-0' />
                  <span className='truncate'>{asset.name}</span>
                  <Trash2 className='size-3 shrink-0' />
                </button>
              ))}
            </div>
          )}
          <div className='text-muted-foreground text-right text-[11px]'>
            {props.prompt.length}/5000
          </div>
          {props.mode === 'video' && props.videoCapabilities && (
            <VideoReferenceFields
              value={props.videoReferences}
              onFilesSelected={props.onVideoReferenceFilesSelected}
              onRemoveImage={props.onRemoveVideoReferenceImage}
              onRemoveVideo={props.onRemoveVideoReferenceVideo}
              onRemoveAudio={props.onRemoveVideoReferenceAudio}
            />
          )}
          {props.mode === 'image' && props.imageReferencesSupported && (
            <ImageReferenceFields
              value={props.imageReferences}
              onFilesSelected={props.onImageReferenceFilesSelected}
              onRemoveImage={props.onRemoveImageReferenceImage}
            />
          )}
        </div>
        <Button
          size='icon-lg'
          aria-label={t('Submit')}
          onClick={props.onSubmit}
          disabled={!canSubmit}
        >
          {props.submitting ? (
            <RefreshCw data-icon='inline-start' className='animate-spin' />
          ) : (
            <Send data-icon='inline-start' />
          )}
        </Button>
      </div>
      {props.mode === 'image' && props.imageReferencesSupported && (
        <>
          <Separator className='my-3' />
          <div className='grid gap-3 sm:grid-cols-3'>
            <ComposerSelectGroup
              label={t('Aspect ratio')}
              value={props.imageOptions.aspectRatio}
              options={CREATION_IMAGE_ASPECT_RATIO_OPTIONS.map((value) => ({
                value,
                label: value,
              }))}
              onChange={(value) =>
                props.onImageOptionsChange({
                  ...props.imageOptions,
                  aspectRatio: value as CreationImageAspectRatio,
                })
              }
            />
          </div>
        </>
      )}
      {props.mode === 'video' && (
        <>
          <Separator className='my-3' />
          <div
            className={cn(
              'grid gap-3',
              props.videoCapabilities?.showResolution === false
                ? 'sm:grid-cols-3'
                : props.videoCapabilities
                  ? 'sm:grid-cols-4'
                  : 'sm:grid-cols-2'
            )}
          >
            {!!props.videoCapabilities?.referenceModes.length && (
              <ComposerSelectGroup
                label={t('Reference mode')}
                value={props.videoReferences.referenceMode}
                options={props.videoCapabilities.referenceModes.map(
                  (value) => ({
                    value,
                    label: getReferenceModeLabel(value, t),
                  })
                )}
                onChange={(value) =>
                  props.onVideoReferencesChange(
                    normalizeCreationVideoReferences(
                      {
                        ...props.videoReferences,
                        referenceMode: value as CreationVideoReferenceMode,
                      },
                      props.model?.id
                    )
                  )
                }
              />
            )}
            {!!props.videoCapabilities?.aspectRatios.length && (
              <ComposerSelectGroup
                label={t('Aspect ratio')}
                value={props.videoOptions.aspectRatio ?? '9:16'}
                options={props.videoCapabilities.aspectRatios.map((value) => ({
                  value,
                  label: value,
                }))}
                onChange={(value) =>
                  props.onVideoOptionsChange({
                    ...props.videoOptions,
                    aspectRatio: value as CreationAspectRatio,
                  })
                }
              />
            )}
            {props.videoCapabilities?.showResolution !== false && (
              <ComposerSelectGroup
                label={t('Resolution')}
                value={props.videoOptions.resolution}
                options={props.resolutionOptions}
                onChange={(value) =>
                  props.onVideoOptionsChange({
                    ...props.videoOptions,
                    resolution: value as CreationResolution,
                  })
                }
              />
            )}
            <ComposerSelectGroup
              label={t('Video duration')}
              value={props.videoOptions.duration}
              options={props.durationOptions}
              onChange={(value) =>
                props.onVideoOptionsChange({
                  ...props.videoOptions,
                  duration: value as CreationDuration,
                })
              }
            />
          </div>
        </>
      )}
      <div className='text-muted-foreground mt-3 flex flex-wrap items-center justify-end gap-2 rounded-lg border px-3 py-2 text-xs'>
        {!props.authenticated && (
          <span className='mr-auto'>
            {t('Sign in before submitting a real creation task.')}
          </span>
        )}
        <span className='text-muted-foreground'>
          {t('Session')} #{props.sessionNumber} · {props.assets.length}{' '}
          {t('assets')}
        </span>
        <span>{t('Press Enter to send, Shift+Enter for newline.')}</span>
      </div>
    </section>
  )
}

function ComposerSelectGroup(props: {
  label: string
  value: string
  options: Array<{ value: string; label: string }>
  onChange: (value: string) => void
}) {
  return (
    <Field>
      <FieldLabel>{props.label}</FieldLabel>
      <Select
        items={props.options}
        value={props.value}
        onValueChange={(value) => {
          if (typeof value === 'string') props.onChange(value)
        }}
      >
        <SelectTrigger className='w-full'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {props.options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  )
}

function getReferenceModeLabel(
  value: CreationVideoReferenceMode,
  t: (key: string) => string
) {
  if (value === 'video') return t('Video reference')
  if (value === 'multimodal') return t('Multimodal reference')
  return t('Image reference')
}

type ReferenceMentionItem = {
  id: string
  kind: 'image' | 'video' | 'audio'
  label: string
  token: string
}

function getVideoReferenceMentionItems(
  references: CreationVideoReferences,
  t: (key: string) => string
): ReferenceMentionItem[] {
  const imageItems = references.imageUrls
    .filter((reference) => getCreationReferenceURL(reference))
    .map((_, index) => ({
      id: `image-${index}`,
      kind: 'image' as const,
      label: `${t('Reference image')} ${index + 1}`,
      token: `${t('Reference image')}${index + 1}`,
    }))
  const videoItems = references.videoUrls
    .filter((reference) => getCreationReferenceURL(reference))
    .map((_, index) => ({
      id: `video-${index}`,
      kind: 'video' as const,
      label: `${t('Reference video')} ${index + 1}`,
      token: `${t('Reference video')}${index + 1}`,
    }))
  const audioItem = getCreationReferenceURL(references.audioUrl)
    ? [
        {
          id: 'audio',
          kind: 'audio' as const,
          label: t('Reference audio'),
          token: t('Reference audio'),
        },
      ]
    : []

  return [...imageItems, ...videoItems, ...audioItem]
}

function matchesReferenceMentionQuery(
  item: ReferenceMentionItem,
  query: string
) {
  const normalizedQuery = query.toLocaleLowerCase()
  return (
    !normalizedQuery ||
    item.label.toLocaleLowerCase().includes(normalizedQuery) ||
    item.token.toLocaleLowerCase().includes(normalizedQuery)
  )
}

function getReferenceMentionIcon(kind: ReferenceMentionItem['kind']) {
  if (kind === 'video') return <FileVideo className='size-4 shrink-0' />
  if (kind === 'audio') return <FileAudio className='size-4 shrink-0' />
  return <FileImage className='size-4 shrink-0' />
}
