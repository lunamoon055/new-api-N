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
import { useState, type ReactNode } from 'react'
import { FileAudio, FileImage, FileVideo, Trash2, Upload } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldGroup } from '@/components/ui/field'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT,
  CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT,
  getCreationReferencePreviewURL,
  getCreationReferenceURL,
  type CreationVideoReferenceMode,
  type CreationVideoReferences,
} from './session'

const IMAGE_REFERENCE_ACCEPT =
  'image/avif,image/gif,image/jpeg,image/png,image/webp,.avif,.gif,.jpeg,.jpg,.png,.webp'
const VIDEO_REFERENCE_ACCEPT = 'video/mp4,.mp4'
const AUDIO_REFERENCE_ACCEPT =
  'audio/mpeg,audio/mp3,audio/wav,audio/x-wav,.mp3,.wav'

type ReferencePreview = {
  kind: 'image' | 'video' | 'audio'
  url: string
  title: string
}

type VideoReferenceFieldsProps = {
  value: CreationVideoReferences
  onFilesSelected: (files: File[]) => void
  onRemoveImage: (index: number) => void
  onRemoveVideo: (index: number) => void
  onRemoveAudio: () => void
}

export function VideoReferenceFields(props: VideoReferenceFieldsProps) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<ReferencePreview | null>(null)
  const referenceMode = props.value.referenceMode
  const showImages = referenceMode === 'image' || referenceMode === 'multimodal'
  const showVideos = referenceMode === 'video' || referenceMode === 'multimodal'
  const showAudio = referenceMode === 'multimodal'
  const imageReferences = showImages
    ? props.value.imageUrls.filter((reference) =>
        getCreationReferenceURL(reference)
      )
    : []
  const videoReferences = showVideos
    ? props.value.videoUrls.filter((reference) =>
        getCreationReferenceURL(reference)
      )
    : []
  const audioReference =
    showAudio && getCreationReferenceURL(props.value.audioUrl)
      ? props.value.audioUrl
      : ''
  const referenceCount =
    imageReferences.length + videoReferences.length + (audioReference ? 1 : 0)
  const uploadDisabled = getUploadDisabled({
    mode: referenceMode,
    imageCount: imageReferences.length,
    videoCount: videoReferences.length,
    hasAudio: !!audioReference,
  })

  return (
    <TooltipProvider>
      <FieldGroup className='mt-3 gap-2'>
        <p className='text-muted-foreground text-[11px] leading-4'>
          {t(getReferenceUploadTip(referenceMode))}
        </p>
        <Field>
          <div className='flex flex-wrap items-center gap-2'>
            <label
              className='border-input hover:bg-muted inline-flex h-9 cursor-pointer items-center justify-center gap-1.5 rounded-lg border bg-transparent px-3 text-sm font-medium whitespace-nowrap transition-colors data-[disabled=true]:pointer-events-none data-[disabled=true]:opacity-50'
              data-disabled={uploadDisabled ? 'true' : undefined}
            >
              <Upload data-icon='inline-start' />
              {t('Reference assets')}
              <input
                type='file'
                accept={getReferenceAccept(referenceMode)}
                multiple
                disabled={uploadDisabled}
                className='sr-only'
                onChange={(event) => {
                  props.onFilesSelected(
                    event.currentTarget.files
                      ? Array.from(event.currentTarget.files)
                      : []
                  )
                  event.currentTarget.value = ''
                }}
              />
            </label>
            <span className='text-muted-foreground text-xs'>
              {referenceCount
                ? t('{{count}} reference asset(s)', {
                    count: referenceCount,
                  })
                : t('No reference assets')}
            </span>
          </div>
        </Field>
        {!!referenceCount && (
          <div className='flex max-w-full flex-wrap gap-1.5'>
            {imageReferences.map((reference, index) => (
              <ReferenceChip
                key={`image-${getCreationReferenceURL(reference)}-${index}`}
                icon={<FileImage className='size-3 shrink-0' />}
                label={`${t('Reference image')} ${index + 1}`}
                removeLabel={`${t('Remove reference image')} ${index + 1}`}
                onOpen={() =>
                  setPreview({
                    kind: 'image',
                    url: getCreationReferencePreviewURL(reference),
                    title: `${t('Reference image')} ${index + 1}`,
                  })
                }
                onRemove={() => props.onRemoveImage(index)}
              />
            ))}
            {videoReferences.map((reference, index) => (
              <ReferenceChip
                key={`video-${getCreationReferenceURL(reference)}-${index}`}
                icon={<FileVideo className='size-3 shrink-0' />}
                label={`${t('Reference video')} ${index + 1}`}
                removeLabel={`${t('Remove reference video')} ${index + 1}`}
                onOpen={() =>
                  setPreview({
                    kind: 'video',
                    url: getCreationReferencePreviewURL(reference),
                    title: `${t('Reference video')} ${index + 1}`,
                  })
                }
                onRemove={() => props.onRemoveVideo(index)}
              />
            ))}
            {audioReference && (
              <ReferenceChip
                icon={<FileAudio className='size-3 shrink-0' />}
                label={t('Reference audio')}
                removeLabel={t('Remove reference audio')}
                onOpen={() =>
                  setPreview({
                    kind: 'audio',
                    url: getCreationReferencePreviewURL(audioReference),
                    title: t('Reference audio'),
                  })
                }
                onRemove={props.onRemoveAudio}
              />
            )}
          </div>
        )}
      </FieldGroup>
      <Dialog
        open={!!preview}
        onOpenChange={(open) => {
          if (!open) setPreview(null)
        }}
      >
        <DialogContent className='sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>
              {preview?.title ?? t('Reference preview')}
            </DialogTitle>
          </DialogHeader>
          {preview && <ReferencePreviewMedia preview={preview} />}
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  )
}

function ReferenceChip(props: {
  icon: ReactNode
  label: string
  removeLabel: string
  onOpen: () => void
  onRemove: () => void
}) {
  const { t } = useTranslation()

  return (
    <span className='bg-muted text-muted-foreground inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-1 text-[11px]'>
      <button
        type='button'
        className='hover:text-foreground inline-flex min-w-0 items-center gap-1.5 transition-colors'
        aria-label={`${t('Open reference preview')}: ${props.label}`}
        onClick={props.onOpen}
      >
        {props.icon}
        <span className='max-w-28 truncate'>{props.label}</span>
      </button>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type='button'
              size='icon-xs'
              variant='ghost'
              aria-label={props.removeLabel}
              onClick={props.onRemove}
            />
          }
        >
          <Trash2 data-icon='inline-start' />
        </TooltipTrigger>
        <TooltipContent>{props.removeLabel}</TooltipContent>
      </Tooltip>
    </span>
  )
}

function ReferencePreviewMedia(props: { preview: ReferencePreview }) {
  const { preview } = props

  if (preview.kind === 'image') {
    return (
      <img
        src={preview.url}
        alt={preview.title}
        className='max-h-[70vh] w-full rounded-md object-contain'
      />
    )
  }
  if (preview.kind === 'video') {
    return (
      <video
        src={preview.url}
        controls
        className='max-h-[70vh] w-full rounded-md bg-black'
      />
    )
  }
  return <audio src={preview.url} controls className='w-full' />
}

function getReferenceAccept(mode: CreationVideoReferenceMode) {
  if (mode === 'image') return IMAGE_REFERENCE_ACCEPT
  if (mode === 'video') return VIDEO_REFERENCE_ACCEPT
  return [
    IMAGE_REFERENCE_ACCEPT,
    VIDEO_REFERENCE_ACCEPT,
    AUDIO_REFERENCE_ACCEPT,
  ].join(',')
}

function getReferenceUploadTip(mode: CreationVideoReferenceMode) {
  if (mode === 'video') return 'Video reference upload tip'
  if (mode === 'multimodal') return 'Multimodal reference upload tip'
  return 'Image reference upload tip'
}

function getUploadDisabled(props: {
  mode: CreationVideoReferenceMode
  imageCount: number
  videoCount: number
  hasAudio: boolean
}) {
  if (props.mode === 'image') {
    return props.imageCount >= CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT
  }
  if (props.mode === 'video') {
    return props.videoCount >= CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT
  }
  return (
    props.imageCount >= CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT &&
    props.videoCount >= CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT &&
    props.hasAudio
  )
}
