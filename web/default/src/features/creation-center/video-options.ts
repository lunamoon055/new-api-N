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

export type CreationVideoOptions = {
  resolution: CreationResolution
  duration: CreationDuration
  aspectRatio?: CreationAspectRatio
}

export type CreationVideoReferences = {
  imageUrls: string[]
  startImageUrl: string
  endImageUrl: string
  videoUrls: string[]
  audioUrl: string
}

export type CreationVideoCapability = {
  kind: 'video2'
  durations: string[]
  resolutions: CreationResolution[]
  aspectRatios: CreationAspectRatio[]
  referenceMode: 'images' | 'all'
}

type ResolutionOption = {
  value: CreationResolution
  label: string
  size: string
  estimateMultiplier: number
}

type DurationOption = {
  value: CreationDuration
  label: string
  seconds: string
  estimateSeconds: number
}

type LegacyCreationVideoRequestOptions = {
  seconds: string
  size: string
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

const VIDEO_CAPABILITIES: Record<string, CreationVideoCapability> = {
  'video-2.0': {
    kind: 'video2',
    durations: VIDEO2_DURATIONS,
    resolutions: ['720p'],
    aspectRatios: ['9:16', '16:9', '1:1'],
    referenceMode: 'images',
  },
  'video-2.0-fast': {
    kind: 'video2',
    durations: VIDEO2_DURATIONS,
    resolutions: ['720p'],
    aspectRatios: ['9:16', '16:9', '1:1'],
    referenceMode: 'all',
  },
}

export const DEFAULT_CREATION_VIDEO_OPTIONS: CreationVideoOptions = {
  resolution: '1080p',
  duration: '5',
}

export const EMPTY_CREATION_VIDEO_REFERENCES: CreationVideoReferences = {
  imageUrls: [],
  startImageUrl: '',
  endImageUrl: '',
  videoUrls: [],
  audioUrl: '',
}

const SORA2_VIDEO_SIZE = '720x1280'

function normalizeModelId(modelId?: string) {
  return modelId?.trim().toLowerCase() ?? ''
}

function isSora2Model(modelId?: string) {
  const normalized = normalizeModelId(modelId)
  return normalized === 'sora2' || normalized === 'sora-2'
}

export function isVideo2Model(modelId?: string) {
  return !!getCreationVideoCapabilities(modelId)
}

export function getCreationVideoCapabilities(modelId?: string) {
  return VIDEO_CAPABILITIES[normalizeModelId(modelId)]
}

export function getCreationResolutionOptions(modelId?: string) {
  return getCreationVideoCapabilities(modelId)
    ? VIDEO2_CREATION_RESOLUTION_OPTIONS
    : CREATION_RESOLUTION_OPTIONS
}

export function getCreationDurationOptions(modelId?: string) {
  if (getCreationVideoCapabilities(modelId)) {
    return VIDEO2_DURATION_OPTIONS
  }
  return isSora2Model(modelId)
    ? SORA2_CREATION_DURATION_OPTIONS
    : CREATION_DURATION_OPTIONS
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
  const duration =
    capability && !matchedResolution
      ? durationOptions[0]
      : (matchedDuration ?? durationOptions[0])

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
  if (!Number.isInteger(duration) || duration < 4 || duration > 15) {
    return 'Duration must be between 4 and 15 seconds.'
  }
  return undefined
}

function emptyCreationVideoReferences(): CreationVideoReferences {
  return {
    imageUrls: [],
    startImageUrl: '',
    endImageUrl: '',
    videoUrls: [],
    audioUrl: '',
  }
}

function cleanURLs(values: string[]) {
  return values.map((value) => value.trim()).filter(Boolean)
}

export function normalizeCreationVideoReferences(
  references: CreationVideoReferences,
  modelId?: string
): CreationVideoReferences {
  const capability = getCreationVideoCapabilities(modelId)
  if (!capability) return emptyCreationVideoReferences()

  const imageUrls = cleanURLs(references.imageUrls)
  if (capability.referenceMode === 'images') {
    return {
      ...emptyCreationVideoReferences(),
      imageUrls,
    }
  }

  return {
    imageUrls,
    startImageUrl: references.startImageUrl.trim(),
    endImageUrl: references.endImageUrl.trim(),
    videoUrls: cleanURLs(references.videoUrls),
    audioUrl: references.audioUrl.trim(),
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
  if (imageCount > 4) {
    return 'Video2 accepts at most 4 image references.'
  }
  if (normalized.videoUrls.length > 3) {
    return 'Video2 accepts at most 3 video references.'
  }

  const urls = [
    ...normalized.imageUrls,
    normalized.startImageUrl,
    normalized.endImageUrl,
    ...normalized.videoUrls,
    normalized.audioUrl,
  ].filter(Boolean)
  if (urls.some((url) => !isHTTPURL(url))) {
    return 'Reference URL must use HTTP or HTTPS.'
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
    durationOptions.find(
      (item) => item.value === normalizedOptions.duration
    ) ?? durationOptions[0]
  const capability = getCreationVideoCapabilities(modelId)

  if (!capability) {
    const resolutionOptions = getCreationResolutionOptions(modelId)
    const resolution =
      resolutionOptions.find(
        (item) => item.value === normalizedOptions.resolution
      ) ?? resolutionOptions[0]
    return {
      seconds: duration.seconds,
      size: isSora2Model(modelId) ? SORA2_VIDEO_SIZE : resolution.size,
      estimateSeconds: Math.ceil(
        duration.estimateSeconds *
          (isSora2Model(modelId) ? 1 : resolution.estimateMultiplier)
      ),
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

  if (normalizedReferences.imageUrls.length === 1) {
    request.image_url = normalizedReferences.imageUrls[0]
  } else if (normalizedReferences.imageUrls.length > 1) {
    request.image_urls = normalizedReferences.imageUrls
  }
  if (normalizedReferences.videoUrls.length === 1) {
    request.video_url = normalizedReferences.videoUrls[0]
  } else if (normalizedReferences.videoUrls.length > 1) {
    request.video_reference = normalizedReferences.videoUrls.map((url) => ({
      url,
    }))
  }
  if (normalizedReferences.startImageUrl) {
    request.start_image_url = normalizedReferences.startImageUrl
  }
  if (normalizedReferences.endImageUrl) {
    request.end_image_url = normalizedReferences.endImageUrl
  }
  if (normalizedReferences.audioUrl) {
    request.audio_url = normalizedReferences.audioUrl
  }

  return request
}
