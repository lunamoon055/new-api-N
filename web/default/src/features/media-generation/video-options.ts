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

export type CreationResolution = '720p' | '1080p' | '2k' | '4k'
export type CreationAspectRatio = '9:16' | '16:9' | '1:1'
export type CreationDuration = string
export type CreationVideoReferenceMode = 'image' | 'video' | 'multimodal'

export type CreationVideoOptions = {
  resolution: CreationResolution
  duration: CreationDuration
  aspectRatio?: CreationAspectRatio
}

export type CreationVideoReferences = {
  referenceMode: CreationVideoReferenceMode
  imageUrls: CreationVideoReferenceValue[]
  startImageUrl: string
  endImageUrl: string
  videoUrls: CreationVideoReferenceValue[]
  audioUrl: CreationVideoReferenceValue
}

export type CreationVideoReferenceObject = {
  url: string
  previewUrl?: string
}

export type CreationVideoReferenceValue = string | CreationVideoReferenceObject

export type CreationVideoCapability = {
  kind: 'sora2' | 'video2'
  durations: string[]
  resolutions: CreationResolution[]
  aspectRatios: CreationAspectRatio[]
  referenceModes: CreationVideoReferenceMode[]
  showResolution: boolean
  durationControl: 'menu' | 'select'
}

export type ResolutionOption = {
  value: CreationResolution
  label: string
  size: string
  estimateMultiplier: number
}

export type DurationOption = {
  value: CreationDuration
  label: string
  seconds: string
  estimateSeconds: number
}

type LegacyCreationVideoRequestOptions = {
  seconds: string
  size: string
  aspect_ratio?: Extract<CreationAspectRatio, '9:16' | '16:9'>
  input_reference?: string
  estimateSeconds: number
}

type Video2CreationVideoRequestOptions = {
  duration: number
  aspect_ratio: CreationAspectRatio
  resolution: '720p'
  async: true
  estimateSeconds: number
  image_url?: string
  image_urls?: string[]
  video_url?: string
  video_reference?: Array<{ url: string }>
  start_image_url?: string
  end_image_url?: string
  audio_url?: string
}

export type CreationVideoRequestOptions =
  | LegacyCreationVideoRequestOptions
  | Video2CreationVideoRequestOptions

export const CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT = 4
export const CREATION_VIDEO_IMAGE_REFERENCE_MAX_BYTES = 20 * 1024 * 1024
export const CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT = 3
export const CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES = 200 * 1024 * 1024
export const CREATION_VIDEO_AUDIO_REFERENCE_MAX_BYTES = 15 * 1024 * 1024

export const CREATION_RESOLUTION_OPTIONS: ResolutionOption[] = [
  { value: '1080p', label: '1080', size: '1920x1080', estimateMultiplier: 1 },
  { value: '2k', label: '2K', size: '2560x1440', estimateMultiplier: 1.35 },
  { value: '4k', label: '4K', size: '3840x2160', estimateMultiplier: 1.75 },
]

const VIDEO2_CREATION_RESOLUTION_OPTIONS: ResolutionOption[] = [
  {
    value: '720p',
    label: '720p',
    size: '720x1280',
    estimateMultiplier: 1,
  },
]

export const CREATION_DURATION_OPTIONS: DurationOption[] = [
  { value: '5', label: '5s', seconds: '5', estimateSeconds: 90 },
  { value: '10', label: '10s', seconds: '10', estimateSeconds: 150 },
  { value: '15', label: '15s', seconds: '15', estimateSeconds: 210 },
]

export const SORA2_CREATION_DURATION_OPTIONS: DurationOption[] = [
  { value: '4', label: '4s', seconds: '4', estimateSeconds: 75 },
  { value: '8', label: '8s', seconds: '8', estimateSeconds: 135 },
  { value: '12', label: '12s', seconds: '12', estimateSeconds: 195 },
]

const SORA2_CREATION_RESOLUTION_OPTIONS: ResolutionOption[] = [
  {
    value: '720p',
    label: '720p',
    size: '720x1280',
    estimateMultiplier: 1,
  },
]

const VIDEO2_DURATIONS = Array.from({ length: 12 }, (_, index) =>
  String(index + 4)
)

const VIDEO2_DURATION_OPTIONS: DurationOption[] = VIDEO2_DURATIONS.map(
  (duration) => ({
    value: duration,
    label: `${duration}s`,
    seconds: duration,
    estimateSeconds: 60 + Number(duration) * 15,
  })
)

const SORA2_VIDEO_CAPABILITY: CreationVideoCapability = {
  kind: 'sora2',
  durations: ['4', '8', '12'],
  resolutions: ['720p'],
  aspectRatios: ['9:16', '16:9'],
  referenceModes: ['image'],
  showResolution: false,
  durationControl: 'select',
}

const VIDEO_CAPABILITIES: Record<string, CreationVideoCapability> = {
  sora2: SORA2_VIDEO_CAPABILITY,
  'sora-2': SORA2_VIDEO_CAPABILITY,
  'sora-2-pro': SORA2_VIDEO_CAPABILITY,
  'video-2.0': {
    kind: 'video2',
    durations: VIDEO2_DURATIONS,
    resolutions: ['720p'],
    aspectRatios: ['9:16', '16:9', '1:1'],
    referenceModes: ['image', 'video', 'multimodal'],
    showResolution: true,
    durationControl: 'menu',
  },
  'video-2.0-fast': {
    kind: 'video2',
    durations: VIDEO2_DURATIONS,
    resolutions: ['720p'],
    aspectRatios: ['9:16', '16:9', '1:1'],
    referenceModes: ['image', 'video', 'multimodal'],
    showResolution: true,
    durationControl: 'menu',
  },
}

export const DEFAULT_CREATION_VIDEO_OPTIONS: CreationVideoOptions = {
  resolution: '1080p',
  duration: '5',
}

export const EMPTY_CREATION_VIDEO_REFERENCES: CreationVideoReferences = {
  referenceMode: 'image',
  imageUrls: [],
  startImageUrl: '',
  endImageUrl: '',
  videoUrls: [],
  audioUrl: '',
}

const SORA2_VIDEO_SIZES: Record<
  Extract<CreationAspectRatio, '9:16' | '16:9'>,
  string
> = {
  '9:16': '720x1280',
  '16:9': '1280x720',
}

const VIDEO_REFERENCE_IMAGE_EXTENSIONS = [
  'avif',
  'gif',
  'jpeg',
  'jpg',
  'png',
  'webp',
]

const VIDEO_REFERENCE_VIDEO_EXTENSIONS = ['mp4']
const VIDEO_REFERENCE_AUDIO_EXTENSIONS = ['mp3', 'wav']
const VIDEO_REFERENCE_IMAGE_MIME_TYPES = [
  'image/avif',
  'image/gif',
  'image/jpeg',
  'image/png',
  'image/webp',
]
const VIDEO_REFERENCE_VIDEO_MIME_TYPES = ['video/mp4']
const VIDEO_REFERENCE_AUDIO_MIME_TYPES = [
  'audio/mpeg',
  'audio/mp3',
  'audio/wav',
  'audio/wave',
  'audio/x-wav',
]

function normalizeModelId(modelId?: string) {
  return modelId?.trim().toLowerCase() ?? ''
}

export function isVideo2Model(modelId?: string) {
  return getCreationVideoCapabilities(modelId)?.kind === 'video2'
}

export function getCreationVideoCapabilities(modelId?: string) {
  return VIDEO_CAPABILITIES[normalizeModelId(modelId)]
}

export function getCreationResolutionOptions(modelId?: string) {
  const capability = getCreationVideoCapabilities(modelId)
  if (capability?.kind === 'video2') return VIDEO2_CREATION_RESOLUTION_OPTIONS
  if (capability?.kind === 'sora2') return SORA2_CREATION_RESOLUTION_OPTIONS
  return CREATION_RESOLUTION_OPTIONS
}

export function getCreationDurationOptions(modelId?: string) {
  const capability = getCreationVideoCapabilities(modelId)
  if (capability?.kind === 'video2') {
    return VIDEO2_DURATION_OPTIONS
  }
  if (capability?.kind === 'sora2') return SORA2_CREATION_DURATION_OPTIONS
  return CREATION_DURATION_OPTIONS
}

export function normalizeCreationVideoOptions(
  options: CreationVideoOptions,
  modelId?: string
): CreationVideoOptions {
  const resolutionOptions = getCreationResolutionOptions(modelId)
  const matchedResolution = resolutionOptions.find(
    (item) => item.value === options.resolution
  )
  const resolution = matchedResolution ?? resolutionOptions[0]
  const durationOptions = getCreationDurationOptions(modelId)
  const capability = getCreationVideoCapabilities(modelId)
  const matchedDuration = durationOptions.find(
    (item) => item.value === options.duration
  )
  const duration = matchedDuration ?? durationOptions[0]

  if (!capability) {
    return {
      resolution: resolution.value,
      duration: duration.value,
    }
  }

  const aspectRatio = capability.aspectRatios.includes(
    options.aspectRatio as CreationAspectRatio
  )
    ? (options.aspectRatio as CreationAspectRatio)
    : capability.aspectRatios[0]

  return {
    resolution: resolution.value,
    duration: duration.value,
    aspectRatio,
  }
}

export function getCreationVideoOptionsError(
  options: CreationVideoOptions,
  modelId?: string
) {
  if (!getCreationVideoCapabilities(modelId)) return undefined
  const duration = Number(options.duration)
  const capability = getCreationVideoCapabilities(modelId)
  if (
    !Number.isInteger(duration) ||
    !capability?.durations.includes(String(duration))
  ) {
    return 'Duration must be between 4 and 15 seconds.'
  }
  return undefined
}

function emptyCreationVideoReferences(): CreationVideoReferences {
  return {
    referenceMode: 'image',
    imageUrls: [],
    startImageUrl: '',
    endImageUrl: '',
    videoUrls: [],
    audioUrl: '',
  }
}

export function getCreationReferenceURL(
  value?: CreationVideoReferenceValue | null
) {
  if (!value) return ''
  if (typeof value === 'string') return value.trim()
  return value.url.trim()
}

export function getCreationReferencePreviewURL(
  value?: CreationVideoReferenceValue | null
) {
  if (!value) return ''
  if (typeof value === 'string') return value.trim()
  return value.previewUrl?.trim() || value.url.trim()
}

function cleanReferenceValues(
  values: CreationVideoReferenceValue[] | undefined
) {
  return (values ?? []).filter((value) => getCreationReferenceURL(value))
}

function normalizeReferenceString(value: string | undefined) {
  return value?.trim() ?? ''
}

function normalizeReferenceMode(
  value: CreationVideoReferenceMode | undefined,
  capability: CreationVideoCapability
): CreationVideoReferenceMode {
  return value && capability.referenceModes.includes(value)
    ? value
    : (capability.referenceModes[0] ?? 'image')
}

export function normalizeCreationVideoReferences(
  references?: Partial<CreationVideoReferences>,
  modelId?: string
): CreationVideoReferences {
  const capability = getCreationVideoCapabilities(modelId)
  if (!capability) return emptyCreationVideoReferences()

  const referenceMode = normalizeReferenceMode(
    references?.referenceMode,
    capability
  )
  const imageUrls =
    referenceMode === 'image' || referenceMode === 'multimodal'
      ? cleanReferenceValues(references?.imageUrls ?? [])
      : []
  const videoUrls =
    referenceMode === 'video' || referenceMode === 'multimodal'
      ? cleanReferenceValues(references?.videoUrls ?? [])
      : []

  if (referenceMode === 'image') {
    return {
      ...emptyCreationVideoReferences(),
      referenceMode,
      imageUrls,
    }
  }

  return {
    referenceMode,
    imageUrls,
    startImageUrl:
      referenceMode === 'multimodal'
        ? normalizeReferenceString(references?.startImageUrl)
        : '',
    endImageUrl:
      referenceMode === 'multimodal'
        ? normalizeReferenceString(references?.endImageUrl)
        : '',
    videoUrls,
    audioUrl:
      referenceMode === 'multimodal'
        ? references?.audioUrl && getCreationReferenceURL(references.audioUrl)
          ? references.audioUrl
          : ''
        : '',
  }
}

function isHTTPURL(value: string) {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function isReferenceImage(value: string) {
  return isHTTPURL(value) || getDataURLMime(value)?.startsWith('image/')
}

function isReferenceVideo(value: string) {
  return isHTTPURL(value)
}

function isReferenceAudio(value: string) {
  return isHTTPURL(value)
}

function getDataURLMime(value: string) {
  const match = value.match(/^data:([^;,]+)(?:;[^,]*)?,/i)
  return match?.[1]?.toLowerCase()
}

function getURLFileExtension(value: string) {
  try {
    const pathname = new URL(value).pathname
    const filename = pathname.split('/').pop() ?? ''
    const extension = filename.includes('.') ? filename.split('.').pop() : ''
    return extension?.toLowerCase() ?? ''
  } catch {
    return ''
  }
}

function hasAllowedURLFileExtension(value: string, extensions: string[]) {
  const extension = getURLFileExtension(value)
  return !!extension && extensions.includes(extension)
}

function hasAllowedReferenceFormat(
  value: string,
  extensions: string[],
  mimeTypes: string[]
) {
  const mime = getDataURLMime(value)
  if (mime) return mimeTypes.includes(mime)
  return hasAllowedURLFileExtension(value, extensions)
}

export function getCreationVideoReferenceError(
  modelId: string | undefined,
  references: CreationVideoReferences
) {
  const capability = getCreationVideoCapabilities(modelId)
  if (!capability) return undefined

  const normalized = normalizeCreationVideoReferences(references, modelId)
  const imageCount =
    normalized.imageUrls.length +
    (normalized.startImageUrl ? 1 : 0) +
    (normalized.endImageUrl ? 1 : 0)
  if (capability.kind === 'sora2' && imageCount > 1) {
    return 'Sora2 accepts at most 1 reference image.'
  }
  if (imageCount > CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT) {
    return 'Video2 accepts at most 4 image references.'
  }
  if (normalized.videoUrls.length > CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT) {
    return 'Video2 accepts at most 3 video references.'
  }

  const images = [
    ...normalized.imageUrls.map(getCreationReferenceURL),
    normalized.startImageUrl,
    normalized.endImageUrl,
  ].filter(Boolean)
  if (images.some((url) => !isReferenceImage(url))) {
    return 'Reference images must be images or HTTP URLs.'
  }
  if (
    capability.kind === 'video2' &&
    images.some(
      (url) =>
        !hasAllowedReferenceFormat(
          url,
          VIDEO_REFERENCE_IMAGE_EXTENSIONS,
          VIDEO_REFERENCE_IMAGE_MIME_TYPES
        )
    )
  ) {
    return 'Reference image format must be PNG, JPEG, WebP, GIF, or AVIF.'
  }

  const videoUrls = normalized.videoUrls.map(getCreationReferenceURL).filter(Boolean)
  const audioUrls = [getCreationReferenceURL(normalized.audioUrl)].filter(Boolean)
  if (
    videoUrls.some((url) => !isReferenceVideo(url)) ||
    audioUrls.some((url) => !isReferenceAudio(url))
  ) {
    return 'Reference URL must use HTTP or HTTPS.'
  }
  if (
    videoUrls.some(
      (url) =>
        !hasAllowedReferenceFormat(
          url,
          VIDEO_REFERENCE_VIDEO_EXTENSIONS,
          VIDEO_REFERENCE_VIDEO_MIME_TYPES
        )
    )
  ) {
    return 'Reference video format must be MP4.'
  }
  if (
    audioUrls.some(
      (url) =>
        !hasAllowedReferenceFormat(
          url,
          VIDEO_REFERENCE_AUDIO_EXTENSIONS,
          VIDEO_REFERENCE_AUDIO_MIME_TYPES
        )
    )
  ) {
    return 'Reference audio format must be MP3 or WAV.'
  }
  return undefined
}

export function getCreationVideoRequestOptions(
  options: CreationVideoOptions,
  modelId?: string,
  references: CreationVideoReferences = EMPTY_CREATION_VIDEO_REFERENCES
): CreationVideoRequestOptions {
  const normalizedOptions = normalizeCreationVideoOptions(options, modelId)
  const durationOptions = getCreationDurationOptions(modelId)
  const duration =
    durationOptions.find((item) => item.value === normalizedOptions.duration) ??
    durationOptions[0]
  const capability = getCreationVideoCapabilities(modelId)

  if (!capability) {
    const resolutionOptions = getCreationResolutionOptions(modelId)
    const resolution =
      resolutionOptions.find(
        (item) => item.value === normalizedOptions.resolution
      ) ?? resolutionOptions[0]
    return {
      seconds: duration.seconds,
      size: resolution.size,
      estimateSeconds: Math.ceil(
        duration.estimateSeconds * resolution.estimateMultiplier
      ),
    }
  }

  if (capability.kind === 'sora2') {
    const aspectRatio =
      normalizedOptions.aspectRatio === '16:9' ? '16:9' : '9:16'
    const normalizedReferences = normalizeCreationVideoReferences(
      references,
      modelId
    )
    const request: LegacyCreationVideoRequestOptions = {
      seconds: duration.seconds,
      size: SORA2_VIDEO_SIZES[aspectRatio],
      aspect_ratio: aspectRatio,
      estimateSeconds: duration.estimateSeconds,
    }
    const imageReference = getCreationReferenceURL(normalizedReferences.imageUrls[0])
    if (imageReference) request.input_reference = imageReference
    return {
      ...request,
    }
  }

  const normalizedReferences = normalizeCreationVideoReferences(
    references,
    modelId
  )
  const request: Video2CreationVideoRequestOptions = {
    duration: Number(normalizedOptions.duration),
    aspect_ratio: normalizedOptions.aspectRatio ?? capability.aspectRatios[0],
    resolution: '720p',
    async: true,
    estimateSeconds: duration.estimateSeconds,
  }

  const imageUrls = normalizedReferences.imageUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
  const videoUrls = normalizedReferences.videoUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
  const audioUrl = getCreationReferenceURL(normalizedReferences.audioUrl)

  if (imageUrls.length === 1) {
    request.image_url = imageUrls[0]
  } else if (imageUrls.length > 1) {
    request.image_urls = imageUrls
  }
  if (videoUrls.length === 1) {
    request.video_url = videoUrls[0]
  } else if (videoUrls.length > 1) {
    request.video_reference = videoUrls.map((url) => ({
      url,
    }))
  }
  if (normalizedReferences.startImageUrl) {
    request.start_image_url = normalizedReferences.startImageUrl
  }
  if (normalizedReferences.endImageUrl) {
    request.end_image_url = normalizedReferences.endImageUrl
  }
  if (audioUrl) {
    request.audio_url = audioUrl
  }

  return request
}
