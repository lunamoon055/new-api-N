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
  getCreationVideoReferenceLimits,
  getCreationReferencePreviewURL,
  getCreationReferenceURL,
  type CreationVideoCapability,
  type CreationVideoReferenceMode,
  type CreationVideoReferences,
} from './session'

const IMAGE_REFERENCE_ACCEPT =
  'image/avif,image/gif,image/jpeg,image/png,image/webp,.avif,.gif,.jpeg,.jpg,.png,.webp'
const SEEDANCE_25_IMAGE_REFERENCE_ACCEPT =
  'image/jpeg,image/png,image/webp,.jpeg,.jpg,.png,.webp'
const VIDEO_REFERENCE_ACCEPT = 'video/mp4,.mp4'
const AUDIO_REFERENCE_ACCEPT =
  'audio/mpeg,audio/mp3,audio/wav,audio/x-wav,.mp3,.wav'
const MINIMAX_H3_AUDIO_REFERENCE_ACCEPT =
  'audio/mpeg,audio/mp3,audio/wav,audio/x-wav,audio/mp4,audio/x-m4a,audio/aac,audio/x-aac,audio/ogg,application/ogg,audio/webm,.mp3,.wav,.m4a,.aac,.ogg,.webm'

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
  onRemoveAudio: (index: number) => void
  capability?: CreationVideoCapability
}

export function VideoReferenceFields(props: VideoReferenceFieldsProps) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<ReferencePreview | null>(null)
  const referenceMode = props.value.referenceMode
  const limits =
    props.capability?.referenceLimits ?? getCreationVideoReferenceLimits()
  const showImages =
    (referenceMode === 'image' ||
      referenceMode === 'multimodal' ||
      referenceMode === 'frames' ||
      referenceMode === 'image-audio') &&
    limits.maxImages > 0
  const showVideos =
    (referenceMode === 'video' || referenceMode === 'multimodal') &&
    limits.maxVideos > 0
  const showAudio =
    (referenceMode === 'multimodal' || referenceMode === 'image-audio') &&
    limits.maxAudios > 0
  const imageReferences = showImages
    ? referenceMode === 'frames'
      ? [
          {
            reference: props.value.startImageUrl,
            label: t('Start frame'),
            removeIndex: 0,
          },
          {
            reference: props.value.endImageUrl,
            label: t('End frame'),
            removeIndex: 1,
          },
        ].filter((item) => getCreationReferenceURL(item.reference))
      : props.value.imageUrls
          .filter((reference) => getCreationReferenceURL(reference))
          .map((reference, index) => ({
            reference,
            label: `${t('Reference image')} ${index + 1}`,
            removeIndex: index,
          }))
    : []
  const videoReferences = showVideos
    ? props.value.videoUrls.filter((reference) =>
        getCreationReferenceURL(reference)
      )
    : []
  const audioReferences = showAudio
    ? props.value.audioUrls.filter((reference) =>
        getCreationReferenceURL(reference)
      )
    : []
  const referenceCount =
    imageReferences.length + videoReferences.length + audioReferences.length
  const uploadDisabled = getUploadDisabled({
    mode: referenceMode,
    imageCount: imageReferences.length,
    videoCount: videoReferences.length,
    audioCount: audioReferences.length,
    limits,
  })

  return (
    <TooltipProvider>
      <FieldGroup className='mt-3 gap-2'>
        <p className='text-muted-foreground text-[11px] leading-4'>
          {getReferenceUploadTip(
            referenceMode,
            limits,
            t,
            props.capability?.uploadTipProfile
          )}
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
                id='creation-reference-upload'
                type='file'
                accept={getReferenceAccept(
                  referenceMode,
                  limits,
                  props.capability?.uploadTipProfile
                )}
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
            {imageReferences.map((item, index) => (
              <ReferenceChip
                key={`image-${getCreationReferenceURL(item.reference)}-${index}`}
                icon={<FileImage className='size-3 shrink-0' />}
                label={item.label}
                removeLabel={`${t('Remove reference image')}: ${item.label}`}
                onOpen={() =>
                  setPreview({
                    kind: 'image',
                    url: getCreationReferencePreviewURL(item.reference),
                    title: item.label,
                  })
                }
                onRemove={() => props.onRemoveImage(item.removeIndex)}
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
            {audioReferences.map((reference, index) => (
              <ReferenceChip
                key={`audio-${getCreationReferenceURL(reference)}-${index}`}
                icon={<FileAudio className='size-3 shrink-0' />}
                label={`${t('Reference audio')} ${index + 1}`}
                removeLabel={`${t('Remove reference audio')} ${index + 1}`}
                onOpen={() =>
                  setPreview({
                    kind: 'audio',
                    url: getCreationReferencePreviewURL(reference),
                    title: `${t('Reference audio')} ${index + 1}`,
                  })
                }
                onRemove={() => props.onRemoveAudio(index)}
              />
            ))}
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

function getReferenceAccept(
  mode: CreationVideoReferenceMode,
  limits: CreationVideoCapability['referenceLimits'],
  profile?: CreationVideoCapability['uploadTipProfile']
) {
  const imageAccept =
    profile === 'seedance-2.5'
      ? SEEDANCE_25_IMAGE_REFERENCE_ACCEPT
      : IMAGE_REFERENCE_ACCEPT
  if (mode === 'image' || mode === 'frames') return imageAccept
  if (mode === 'video') return VIDEO_REFERENCE_ACCEPT
  if (mode === 'image-audio') {
    const audioAccept =
      profile === 'seedance-2.5'
        ? AUDIO_REFERENCE_ACCEPT
        : MINIMAX_H3_AUDIO_REFERENCE_ACCEPT
    return [imageAccept, audioAccept].join(',')
  }
  return [
    limits.maxImages > 0 ? imageAccept : '',
    limits.maxVideos > 0 ? VIDEO_REFERENCE_ACCEPT : '',
    limits.maxAudios > 0 ? AUDIO_REFERENCE_ACCEPT : '',
  ]
    .filter(Boolean)
    .join(',')
}

function getReferenceUploadTip(
  mode: CreationVideoReferenceMode,
  limits: CreationVideoCapability['referenceLimits'],
  t: (key: string, options?: Record<string, unknown>) => string,
  profile?: CreationVideoCapability['uploadTipProfile']
) {
  if (profile === 'video-2.5') {
    if (mode === 'video') {
      return t(
        'Video 2.5 tip: Upload up to {{videoCount}} MP4 videos. Each video must be 3-10 seconds and no more than {{videoSize}} MB; all reference videos together must not exceed 30 seconds.',
        {
          videoCount: limits.maxVideos,
          videoSize: limits.maxVideoSizeMB,
        }
      )
    }
    if (mode === 'multimodal') {
      return t(
        'Video 2.5 tip: Images support PNG, JPEG, WebP, GIF, or AVIF, up to {{imageCount}}, {{imageSize}} MB each. Videos support MP4, up to {{videoCount}}, 3-10 seconds and {{videoSize}} MB each, 30 seconds total. Audio supports MP3 or WAV, up to {{audioCount}}, 3-30 seconds and {{audioSize}} MB each, 30 seconds total.',
        {
          imageCount: limits.maxImages,
          videoCount: limits.maxVideos,
          audioCount: limits.maxAudios,
          imageSize: limits.maxImageSizeMB,
          videoSize: limits.maxVideoSizeMB,
          audioSize: limits.maxAudioSizeMB,
        }
      )
    }
  }
  if (profile === 'seedance-2.5') {
    if (mode === 'frames') {
      return t(
        'Seedance 2.5 tip: Upload one JPG, PNG, or WebP start frame and one end frame. Each image must not exceed {{imageSize}} MB. Recommended resolution: 1080p to 4K.',
        { imageSize: limits.maxImageSizeMB }
      )
    }
    if (mode === 'image-audio') {
      return t(
        'Seedance 2.5 tip: Images support JPG, PNG, or WebP, up to {{imageCount}}, {{imageSize}} MB each, recommended 1080p to 4K. Audio supports MP3 or WAV, up to {{audioCount}}, 2-30 seconds and {{audioSize}} MB each, 30 seconds total.',
        {
          imageCount: limits.maxImages,
          audioCount: limits.maxAudios,
          imageSize: limits.maxImageSizeMB,
          audioSize: limits.maxAudioSizeMB,
        }
      )
    }
    return t(
      'Seedance 2.5 tip: Upload up to {{imageCount}} JPG, PNG, or WebP reference images. Each image must not exceed {{imageSize}} MB. Recommended resolution: 1080p to 4K.',
      {
        imageCount: limits.maxImages,
        imageSize: limits.maxImageSizeMB,
      }
    )
  }
  if (mode === 'frames') {
    return t(
      'Tip: Upload one start frame and one end frame. Each image must not exceed {{size}} MB.',
      { size: limits.maxImageSizeMB }
    )
  }
  if (mode === 'image-audio') {
    return t(
      'Tip: Upload up to {{imageCount}} images and one audio file. Audio requires an image, supports MP3, WAV, M4A, AAC, OGG, or WebM, lasts {{min}}-{{max}} seconds, and must not exceed {{audioSize}} MB.',
      {
        imageCount: limits.maxImages,
        min: limits.minReferenceAudioDurationSeconds ?? 2,
        max: limits.maxReferenceAudioDurationSeconds ?? 15,
        audioSize: limits.maxAudioSizeMB,
      }
    )
  }
  if (mode === 'video') {
    return t(
      'Tip: Reference videos support MP4. Up to {{count}} videos, {{size}} MB each.',
      {
        count: limits.maxVideos,
        size: limits.maxVideoSizeMB,
      }
    )
  }
  if (mode === 'multimodal') {
    return t(
      'Tip: Reference assets support images, videos, and audio. Images: up to {{imageCount}}, {{imageSize}} MB each. Videos: up to {{videoCount}}, {{videoSize}} MB each. Audio: up to {{audioCount}}, {{audioSize}} MB each.',
      {
        imageCount: limits.maxImages,
        imageSize: limits.maxImageSizeMB,
        videoCount: limits.maxVideos,
        videoSize: limits.maxVideoSizeMB,
        audioCount: limits.maxAudios,
        audioSize: limits.maxAudioSizeMB,
      }
    )
  }
  return t(
    'Tip: Reference images support PNG, JPEG, WebP, GIF, or AVIF. Up to {{count}} images, {{size}} MB each.',
    {
      count: limits.maxImages,
      size: limits.maxImageSizeMB,
    }
  )
}

function getUploadDisabled(props: {
  mode: CreationVideoReferenceMode
  imageCount: number
  videoCount: number
  audioCount: number
  limits: CreationVideoCapability['referenceLimits']
}) {
  if (props.mode === 'image') {
    return props.imageCount >= props.limits.maxImages
  }
  if (props.mode === 'frames') {
    return props.imageCount >= 2
  }
  if (props.mode === 'video') {
    return props.videoCount >= props.limits.maxVideos
  }
  if (props.mode === 'image-audio') {
    return (
      props.imageCount >= props.limits.maxImages &&
      props.audioCount >= props.limits.maxAudios
    )
  }
  return (
    props.imageCount >= props.limits.maxImages &&
    props.videoCount >= props.limits.maxVideos &&
    (props.limits.maxAudios <= 0 || props.audioCount >= props.limits.maxAudios)
  )
}
