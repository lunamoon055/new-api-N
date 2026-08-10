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

export type CreationResolution = '480p' | '720p' | '1080p' | '2k' | '4k'
export type CreationAspectRatio = string
export type CreationDuration = string
export type CreationVideoReferenceMode =
  | 'text'
  | 'image'
  | 'video'
  | 'multimodal'
  | 'frames'
  | 'image-audio'
type Sora2AspectRatio = '9:16' | '16:9'

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
  audioUrls: CreationVideoReferenceValue[]
  audioUrl: CreationVideoReferenceValue
}

export type CreationVideoReferenceObject = {
  url: string
  previewUrl?: string
  sizeBytes?: number
  durationSeconds?: number
}

export type CreationVideoReferenceValue = string | CreationVideoReferenceObject

export type CreationModelMetadataLike = {
  provider?: string
  id?: string
  upstream_model_id?: string
  type?: string
  category?: string
  model_type?: string
  resolutions?: string[]
  ratios?: string[]
  aspect_ratios?: string[]
  durations?: Array<number | string>
  max_prompt_length?: number
  max_images?: number
  max_videos?: number
  max_audios?: number
  max_media_files?: number
  max_image_size_mb?: number
  max_video_size_mb?: number
  max_audio_size_mb?: number
  min_reference_video_duration_seconds?: number
  max_reference_video_duration_seconds?: number
  concurrency_options?: number[]
}

export type CreationModelLike = {
  id?: string
  metadata?: CreationModelMetadataLike
}

export type CreationModelInput = string | CreationModelLike | undefined

export type CreationVideoReferenceLimits = {
  maxImages: number
  maxVideos: number
  maxAudios: number
  maxMediaFiles?: number
  maxImageSizeBytes: number
  maxVideoSizeBytes: number
  maxAudioSizeBytes: number
  maxImageSizeMB: number
  maxVideoSizeMB: number
  maxAudioSizeMB: number
  minReferenceVideoDurationSeconds?: number
  maxReferenceVideoDurationSeconds?: number
  maxReferenceVideoTotalDurationSeconds?: number
  minReferenceAudioDurationSeconds?: number
  maxReferenceAudioDurationSeconds?: number
  maxReferenceAudioTotalDurationSeconds?: number
}

export type CreationVideoCapability = {
  kind: 'sora2' | 'video2' | 'videos' | 'sanbao' | 'minimax-h3'
  durations: string[]
  resolutions: CreationResolution[]
  aspectRatios: CreationAspectRatio[]
  referenceModes: CreationVideoReferenceMode[]
  showResolution: boolean
  includeResolutionInRequest: boolean
  durationControl: 'menu' | 'select'
  referenceLimits: CreationVideoReferenceLimits
  uploadTipProfile?: 'seedance-2.5' | 'video-2.5'
  maxPromptLength?: number
  concurrencyOptions?: number[]
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
  aspect_ratio?: Sora2AspectRatio
  input_reference?: string
  estimateSeconds: number
}

type Video2CreationVideoRequestOptions = {
  duration: number
  aspect_ratio: CreationAspectRatio
  resolution: '480p' | '720p'
  async: true
  estimateSeconds: number
  image_url?: string
  image_urls?: string[]
  video_url?: string
  video_reference?: Array<{ url: string }>
  start_image_url?: string
  end_image_url?: string
  audio_url?: string
  audio_reference?: Array<{ url: string }>
}

type MiniMaxH3CreationVideoRequestOptions = {
  duration: number
  aspect_ratio: CreationAspectRatio
  estimateSeconds: number
  image_url?: string
  image_urls?: string[]
  start_image_url?: string
  end_image_url?: string
  audio_url?: string
}

type SanbaoCreationVideoRequestOptions = {
  duration: number
  ratio: CreationAspectRatio
  resolution: CreationResolution
  concurrency?: number
  reference?: 'all'
  estimateSeconds: number
  images?: string[]
  videos?: string[]
  audios?: string[]
}

type VideosApiCreationVideoRequestOptions = {
  duration: number
  ratio: CreationAspectRatio
  resolution?: '480p' | '720p'
  estimateSeconds: number
  referenceImages?: string[]
  referenceVideos?: string[]
  referenceAudios?: string[]
  start_image_url?: string
  end_image_url?: string
}

export type CreationVideoRequestOptions =
  | LegacyCreationVideoRequestOptions
  | Video2CreationVideoRequestOptions
  | MiniMaxH3CreationVideoRequestOptions
  | SanbaoCreationVideoRequestOptions
  | VideosApiCreationVideoRequestOptions

export const CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT = 4
export const CREATION_VIDEO_IMAGE_REFERENCE_MAX_BYTES = 20 * 1024 * 1024
export const CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT = 3
export const CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES = 200 * 1024 * 1024
export const CREATION_VIDEO_AUDIO_REFERENCE_MAX_BYTES = 15 * 1024 * 1024

const VIDEO2_REFERENCE_LIMITS: CreationVideoReferenceLimits = {
  maxImages: CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT,
  maxVideos: CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT,
  maxAudios: 1,
  maxMediaFiles: undefined,
  maxImageSizeBytes: CREATION_VIDEO_IMAGE_REFERENCE_MAX_BYTES,
  maxVideoSizeBytes: CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES,
  maxAudioSizeBytes: CREATION_VIDEO_AUDIO_REFERENCE_MAX_BYTES,
  maxImageSizeMB: 20,
  maxVideoSizeMB: 200,
  maxAudioSizeMB: 15,
}

const VIDEO25_REFERENCE_LIMITS: CreationVideoReferenceLimits = {
  ...VIDEO2_REFERENCE_LIMITS,
  maxImages: 30,
  maxVideos: 10,
  maxAudios: 10,
  maxMediaFiles: 50,
  minReferenceVideoDurationSeconds: 3,
  maxReferenceVideoDurationSeconds: 10,
  maxReferenceVideoTotalDurationSeconds: 30,
  minReferenceAudioDurationSeconds: 3,
  maxReferenceAudioDurationSeconds: 30,
  maxReferenceAudioTotalDurationSeconds: 30,
}

const VIDEOS_API_REFERENCE_LIMITS: CreationVideoReferenceLimits = {
  maxImages: 9,
  maxVideos: 3,
  maxAudios: 3,
  maxMediaFiles: 15,
  maxImageSizeBytes: CREATION_VIDEO_IMAGE_REFERENCE_MAX_BYTES,
  maxVideoSizeBytes: CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES,
  maxAudioSizeBytes: 50 * 1024 * 1024,
  maxImageSizeMB: 20,
  maxVideoSizeMB: 200,
  maxAudioSizeMB: 50,
}

const SEEDANCE_25_REFERENCE_LIMITS: CreationVideoReferenceLimits = {
  maxImages: 30,
  maxVideos: 10,
  maxAudios: 10,
  maxMediaFiles: 50,
  maxImageSizeBytes: 5 * 1024 * 1024,
  maxVideoSizeBytes: CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES,
  maxAudioSizeBytes: CREATION_VIDEO_AUDIO_REFERENCE_MAX_BYTES,
  maxImageSizeMB: 5,
  maxVideoSizeMB: 200,
  maxAudioSizeMB: 15,
  minReferenceVideoDurationSeconds: 2,
  maxReferenceVideoDurationSeconds: 30,
  maxReferenceVideoTotalDurationSeconds: 30,
  minReferenceAudioDurationSeconds: 2,
  maxReferenceAudioDurationSeconds: 30,
  maxReferenceAudioTotalDurationSeconds: 30,
}

const MINIMAX_H3_REFERENCE_LIMITS: CreationVideoReferenceLimits = {
  ...VIDEO2_REFERENCE_LIMITS,
  maxImages: 5,
  maxVideos: 0,
  maxAudios: 1,
  minReferenceAudioDurationSeconds: 2,
  maxReferenceAudioDurationSeconds: 15,
}

const SORA2_REFERENCE_LIMITS: CreationVideoReferenceLimits = {
  ...VIDEO2_REFERENCE_LIMITS,
  maxImages: 1,
  maxVideos: 0,
  maxAudios: 0,
}

export const CREATION_RESOLUTION_OPTIONS: ResolutionOption[] = [
  { value: '1080p', label: '1080', size: '1920x1080', estimateMultiplier: 1 },
  { value: '2k', label: '2K', size: '2560x1440', estimateMultiplier: 1.35 },
  { value: '4k', label: '4K', size: '3840x2160', estimateMultiplier: 1.75 },
]

const VIDEO2_720P_CREATION_RESOLUTION_OPTIONS: ResolutionOption[] = [
  {
    value: '720p',
    label: '720p',
    size: '720x1280',
    estimateMultiplier: 1,
  },
]

const VIDEO2_480P_CREATION_RESOLUTION_OPTIONS: ResolutionOption[] = [
  {
    value: '480p',
    label: '480p',
    size: '496x864',
    estimateMultiplier: 1,
  },
]

const VIDEOS_API_CREATION_RESOLUTION_OPTIONS: ResolutionOption[] = [
  {
    value: '720p',
    label: '720p',
    size: '1280x720',
    estimateMultiplier: 1,
  },
  {
    value: '480p',
    label: '480p',
    size: '864x496',
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

const VIDEO25_DURATIONS = Array.from({ length: 27 }, (_, index) =>
  String(index + 4)
)

const SEEDANCE_25_DURATIONS = Array.from({ length: 26 }, (_, index) =>
  String(index + 4)
)

const MINIMAX_H3_DURATIONS = Array.from({ length: 11 }, (_, index) =>
  String(index + 5)
)

const SORA2_VIDEO_CAPABILITY: CreationVideoCapability = {
  kind: 'sora2',
  durations: ['4', '8', '12'],
  resolutions: ['720p'],
  aspectRatios: ['9:16', '16:9'],
  referenceModes: ['image'],
  showResolution: false,
  includeResolutionInRequest: false,
  durationControl: 'select',
  referenceLimits: SORA2_REFERENCE_LIMITS,
}

const VIDEO2_720P_CAPABILITY: CreationVideoCapability = {
  kind: 'video2',
  durations: VIDEO2_DURATIONS,
  resolutions: ['720p'],
  aspectRatios: ['9:16', '16:9', '1:1'],
  referenceModes: ['image', 'video', 'multimodal'],
  showResolution: true,
  includeResolutionInRequest: true,
  durationControl: 'menu',
  referenceLimits: VIDEO2_REFERENCE_LIMITS,
}

const VIDEO2_480P_CAPABILITY: CreationVideoCapability = {
  ...VIDEO2_720P_CAPABILITY,
  resolutions: ['480p'],
}

const VIDEO25_720P_CAPABILITY: CreationVideoCapability = {
  ...VIDEO2_720P_CAPABILITY,
  durations: VIDEO25_DURATIONS,
  referenceModes: ['text', 'image', 'frames', 'video', 'multimodal'],
  referenceLimits: VIDEO25_REFERENCE_LIMITS,
  uploadTipProfile: 'video-2.5',
  maxPromptLength: 5000,
}

const VIDEO25_480P_CAPABILITY: CreationVideoCapability = {
  ...VIDEO25_720P_CAPABILITY,
  resolutions: ['480p'],
  showResolution: false,
}

const VIDEOS_API_CAPABILITY: CreationVideoCapability = {
  kind: 'videos',
  durations: VIDEO2_DURATIONS,
  resolutions: ['720p', '480p'],
  aspectRatios: ['16:9', '9:16', '1:1'],
  referenceModes: ['image', 'video', 'multimodal'],
  showResolution: true,
  includeResolutionInRequest: true,
  durationControl: 'menu',
  referenceLimits: VIDEOS_API_REFERENCE_LIMITS,
}

const VIDEOS_4_API_CAPABILITY: CreationVideoCapability = {
  ...VIDEOS_API_CAPABILITY,
  referenceLimits: VIDEO2_REFERENCE_LIMITS,
}

const SD2_1080P_API_CAPABILITY: CreationVideoCapability = {
  ...VIDEOS_API_CAPABILITY,
  resolutions: ['1080p'],
  showResolution: false,
  includeResolutionInRequest: false,
}

const SD2_4K_API_CAPABILITY: CreationVideoCapability = {
  ...VIDEOS_API_CAPABILITY,
  resolutions: ['4k'],
  showResolution: false,
  includeResolutionInRequest: false,
}

const SD2_720P_API_CAPABILITY: CreationVideoCapability = {
  ...VIDEOS_API_CAPABILITY,
  resolutions: ['720p', '480p'],
}

// OpenAI-compatible Seedance 2.x channels use the same internal task endpoint
// as the Videos API. The gateway converts these fields to the upstream
// /v1/videos request contract.
const SEEDANCE_2_API_CAPABILITY: CreationVideoCapability = {
  ...VIDEOS_API_CAPABILITY,
  resolutions: ['720p'],
  aspectRatios: ['16:9', '9:16', '1:1', '4:3', '3:4'],
  referenceModes: ['text', 'image', 'frames', 'multimodal'],
  showResolution: false,
  includeResolutionInRequest: true,
}

const SEEDANCE_25_API_CAPABILITY: CreationVideoCapability = {
  ...SEEDANCE_2_API_CAPABILITY,
  durations: SEEDANCE_25_DURATIONS,
  resolutions: ['720p', '480p'],
  aspectRatios: ['16:9', '9:16', '1:1'],
  referenceModes: ['text', 'image', 'video', 'multimodal'],
  referenceLimits: SEEDANCE_25_REFERENCE_LIMITS,
  uploadTipProfile: 'seedance-2.5',
  showResolution: true,
}

const MINIMAX_H3_CAPABILITY: CreationVideoCapability = {
  kind: 'minimax-h3',
  durations: MINIMAX_H3_DURATIONS,
  resolutions: ['2k'],
  aspectRatios: ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9'],
  referenceModes: ['text', 'image', 'frames', 'image-audio'],
  showResolution: false,
  includeResolutionInRequest: false,
  durationControl: 'menu',
  referenceLimits: MINIMAX_H3_REFERENCE_LIMITS,
  maxPromptLength: 2000,
}

const VIDEO_MODEL_ID_ALIASES: Record<string, string> = {
  'sd2-mini': 'videos-mini',
  'sd2-fast': 'videos-fast',
  sd2满血: 'videos-standard',
}

const VIDEO_CAPABILITIES: Record<string, CreationVideoCapability> = {
  sora2: SORA2_VIDEO_CAPABILITY,
  'sora-2': SORA2_VIDEO_CAPABILITY,
  'sora-2-pro': SORA2_VIDEO_CAPABILITY,
  'video-2.0': VIDEO2_720P_CAPABILITY,
  'video-2.0-fast': VIDEO2_720P_CAPABILITY,
  'video-2.0-mini': VIDEO2_720P_CAPABILITY,
  'video-2.0-480p': VIDEO2_480P_CAPABILITY,
  'video-2.0-fast-480p': VIDEO2_480P_CAPABILITY,
  'video-2.0-mini-480p': VIDEO2_480P_CAPABILITY,
  'video-2.5': VIDEO25_720P_CAPABILITY,
  'video-2.5-480p': VIDEO25_480P_CAPABILITY,
  'videos-standard': VIDEOS_API_CAPABILITY,
  'videos-fast': VIDEOS_API_CAPABILITY,
  'videos-mini': VIDEOS_API_CAPABILITY,
  'videos-4': VIDEOS_4_API_CAPABILITY,
  'videos-4-fast': VIDEOS_4_API_CAPABILITY,
  'videos-4-mini': VIDEOS_4_API_CAPABILITY,
  'sd2-mini': VIDEOS_API_CAPABILITY,
  'sd2-fast': VIDEOS_API_CAPABILITY,
  sd2满血: VIDEOS_API_CAPABILITY,
  'sd2-1080p': SD2_1080P_API_CAPABILITY,
  'sd2-4k': SD2_4K_API_CAPABILITY,
  'sd2-720p': SD2_720P_API_CAPABILITY,
  'sd-2.0-933': SEEDANCE_2_API_CAPABILITY,
  'sd-2-c8': SEEDANCE_2_API_CAPABILITY,
  'seedance-2.0': SEEDANCE_2_API_CAPABILITY,
  'seedance-2.5': SEEDANCE_25_API_CAPABILITY,
  'minimax-h3': MINIMAX_H3_CAPABILITY,
}

export const DEFAULT_CREATION_VIDEO_OPTIONS: CreationVideoOptions = {
  resolution: '720p',
  duration: '5',
}

export const EMPTY_CREATION_VIDEO_REFERENCES: CreationVideoReferences = {
  referenceMode: 'text',
  imageUrls: [],
  startImageUrl: '',
  endImageUrl: '',
  videoUrls: [],
  audioUrls: [],
  audioUrl: '',
}

const SORA2_VIDEO_SIZES: Record<Sora2AspectRatio, string> = {
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
const MINIMAX_H3_REFERENCE_AUDIO_EXTENSIONS = [
  'mp3',
  'wav',
  'm4a',
  'aac',
  'ogg',
  'webm',
]
const VIDEO_REFERENCE_IMAGE_MIME_TYPES = [
  'image/avif',
  'image/gif',
  'image/jpeg',
  'image/png',
  'image/webp',
]
const SEEDANCE_25_REFERENCE_IMAGE_EXTENSIONS = ['jpeg', 'jpg', 'png', 'webp']
const SEEDANCE_25_REFERENCE_IMAGE_MIME_TYPES = [
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
const MINIMAX_H3_REFERENCE_AUDIO_MIME_TYPES = [
  ...VIDEO_REFERENCE_AUDIO_MIME_TYPES,
  'audio/mp4',
  'audio/x-m4a',
  'audio/aac',
  'audio/x-aac',
  'audio/ogg',
  'application/ogg',
  'audio/webm',
]

function getModelId(model?: CreationModelInput) {
  if (typeof model === 'string') return model
  return model?.id ?? ''
}

function getModelMetadata(model?: CreationModelInput) {
  return typeof model === 'string' ? undefined : model?.metadata
}

function normalizeModelId(model?: CreationModelInput) {
  return getModelId(model).trim().toLowerCase()
}

function getNormalizedModelIdVariants(value?: string) {
  const normalized = value?.trim().toLowerCase() ?? ''
  if (!normalized) return []

  const variants = new Set<string>([normalized])
  // Catalog labels may prepend a channel marker (for example
  // "(线路3)sd-2.0-933") and append capability text. Keep both the original
  // display value and the progressively cleaned values for matching.
  for (let pass = 0; pass < 2; pass += 1) {
    for (const candidate of [...variants]) {
      const withoutPrefix = candidate
        .replace(/^\s*[(（][^()（）]*[)）]\s*/u, '')
        .trim()
      const withoutSuffix = candidate
        .replace(/\s*[(（][^()（）]*[)）]\s*$/u, '')
        .trim()
      if (withoutPrefix) variants.add(withoutPrefix)
      if (withoutSuffix) variants.add(withoutSuffix)
    }
  }
  return [...variants].filter(Boolean)
}

function getCreationModelIdCandidates(model?: CreationModelInput) {
  const candidates = getNormalizedModelIdVariants(normalizeModelId(model))
  const metadata = getModelMetadata(model)
  for (const value of [metadata?.upstream_model_id, metadata?.id]) {
    candidates.push(...getNormalizedModelIdVariants(value))
  }
  for (const candidate of [...candidates]) {
    const mapped = VIDEO_MODEL_ID_ALIASES[candidate]
    if (mapped) candidates.push(mapped)
  }
  return [...new Set(candidates.filter(Boolean))]
}

function hasStaticVideoCapabilityKind(
  model: CreationModelInput | undefined,
  kind: CreationVideoCapability['kind']
) {
  return getCreationModelIdCandidates(model).some((candidate) => {
    const capability = VIDEO_CAPABILITIES[candidate]
    return capability?.kind === kind
  })
}

function isVideosApiModel(model?: CreationModelInput) {
  return hasStaticVideoCapabilityKind(model, 'videos')
}

function isMiniMaxH3Model(model?: CreationModelInput) {
  return hasStaticVideoCapabilityKind(model, 'minimax-h3')
}

function isSeedance2Model(model?: CreationModelInput) {
  return getCreationModelIdCandidates(model).some((candidate) =>
    ['sd-2.0-933', 'sd-2-c8', 'seedance-2.0', 'seedance-2.5'].includes(
      candidate
    )
  )
}

function isSeedance25Model(model?: CreationModelInput) {
  return getCreationModelIdCandidates(model).includes('seedance-2.5')
}

function isVideo25Model(model?: CreationModelInput) {
  return getCreationModelIdCandidates(model).some((candidate) =>
    ['video-2.5', 'video-2.5-480p'].includes(candidate)
  )
}

function isSanbaoMetadata(
  metadata?: CreationModelMetadataLike
): metadata is CreationModelMetadataLike {
  return metadata?.provider?.trim().toLowerCase() === 'sanbao'
}

function getSanbaoMediaType(metadata?: CreationModelMetadataLike) {
  return [metadata?.type, metadata?.category, metadata?.model_type]
    .map((value) => value?.trim().toLowerCase() ?? '')
    .find(Boolean)
}

function cleanStringOptions(values?: Array<string | number>) {
  const seen = new Set<string>()
  return (values ?? []).flatMap((value) => {
    const normalized = String(value).trim()
    if (!normalized || seen.has(normalized)) return []
    seen.add(normalized)
    return [normalized]
  })
}

function cleanPositiveIntegers(values?: Array<string | number>) {
  const seen = new Set<string>()
  return (values ?? []).flatMap((value) => {
    const parsed = Number(value)
    if (!Number.isInteger(parsed) || parsed <= 0) return []
    const normalized = String(parsed)
    if (seen.has(normalized)) return []
    seen.add(normalized)
    return [normalized]
  })
}

function buildSanbaoReferenceLimits(
  metadata: CreationModelMetadataLike
): CreationVideoReferenceLimits {
  const maxImageSizeMB = metadata.max_image_size_mb ?? 20
  const maxVideoSizeMB = metadata.max_video_size_mb ?? 200
  const maxAudioSizeMB = metadata.max_audio_size_mb ?? 15
  return {
    maxImages: metadata.max_images ?? 0,
    maxVideos: metadata.max_videos ?? 0,
    maxAudios: metadata.max_audios ?? 0,
    maxMediaFiles: metadata.max_media_files,
    maxImageSizeMB,
    maxVideoSizeMB,
    maxAudioSizeMB,
    maxImageSizeBytes: maxImageSizeMB * 1024 * 1024,
    maxVideoSizeBytes: maxVideoSizeMB * 1024 * 1024,
    maxAudioSizeBytes: maxAudioSizeMB * 1024 * 1024,
    minReferenceVideoDurationSeconds:
      metadata.min_reference_video_duration_seconds,
    maxReferenceVideoDurationSeconds:
      metadata.max_reference_video_duration_seconds,
  }
}

function getSanbaoVideoCapability(
  model?: CreationModelInput
): CreationVideoCapability | undefined {
  if (isVideosApiModel(model) || isMiniMaxH3Model(model)) return undefined
  const metadata = getModelMetadata(model)
  if (!isSanbaoMetadata(metadata)) return undefined
  const mediaType = getSanbaoMediaType(metadata)
  if (!mediaType?.includes('video')) return undefined

  const referenceLimits = buildSanbaoReferenceLimits(metadata)
  const referenceModes: CreationVideoReferenceMode[] = []
  if (referenceLimits.maxImages > 0) referenceModes.push('image')
  if (referenceLimits.maxVideos > 0) referenceModes.push('video')
  if (
    referenceLimits.maxImages > 0 &&
    (referenceLimits.maxVideos > 0 || referenceLimits.maxAudios > 0)
  ) {
    referenceModes.push('multimodal')
  }

  return {
    kind: 'sanbao',
    durations: cleanPositiveIntegers(metadata.durations).length
      ? cleanPositiveIntegers(metadata.durations)
      : VIDEO2_DURATIONS,
    resolutions: cleanStringOptions(metadata.resolutions).length
      ? (cleanStringOptions(metadata.resolutions) as CreationResolution[])
      : ['720p'],
    aspectRatios: cleanStringOptions(
      metadata.ratios?.length ? metadata.ratios : metadata.aspect_ratios
    ).length
      ? (cleanStringOptions(
          metadata.ratios?.length ? metadata.ratios : metadata.aspect_ratios
        ) as CreationAspectRatio[])
      : ['9:16', '16:9', '1:1'],
    referenceModes: referenceModes.length ? referenceModes : ['image'],
    showResolution: true,
    includeResolutionInRequest: true,
    durationControl: 'select',
    referenceLimits,
    concurrencyOptions: metadata.concurrency_options,
  }
}

function getResolutionOption(value: CreationResolution): ResolutionOption {
  const known = [
    ...CREATION_RESOLUTION_OPTIONS,
    ...VIDEO2_720P_CREATION_RESOLUTION_OPTIONS,
    ...VIDEO2_480P_CREATION_RESOLUTION_OPTIONS,
  ].find((item) => item.value === value)
  if (known) return known
  return {
    value,
    label: value,
    size: value,
    estimateMultiplier: 1,
  }
}

function getDurationOption(value: CreationDuration): DurationOption {
  const seconds = Number(value)
  return {
    value,
    label: `${value}s`,
    seconds: value,
    estimateSeconds: Number.isFinite(seconds) ? 60 + seconds * 15 : 120,
  }
}

export function isVideo2Model(model?: CreationModelInput) {
  return getCreationVideoCapabilities(model)?.kind === 'video2'
}

export function getCreationVideoCapabilities(model?: CreationModelInput) {
  const sanbaoCapability = getSanbaoVideoCapability(model)
  if (sanbaoCapability) return sanbaoCapability

  for (const candidate of getCreationModelIdCandidates(model)) {
    const capability = VIDEO_CAPABILITIES[candidate]
    if (capability) return capability
  }
  return undefined
}

export function getCreationPromptMaxLength(model?: CreationModelInput) {
  const capabilityLimit = getCreationVideoCapabilities(model)?.maxPromptLength
  if (capabilityLimit) return capabilityLimit

  const metadataLimit = getModelMetadata(model)?.max_prompt_length
  if (Number.isInteger(metadataLimit) && Number(metadataLimit) > 0) {
    return Number(metadataLimit)
  }
  return 5000
}

export function getCreationResolutionOptions(model?: CreationModelInput) {
  const capability = getCreationVideoCapabilities(model)
  if (capability?.kind === 'sanbao' && capability.resolutions.length) {
    return capability.resolutions.map(getResolutionOption)
  }
  if (capability?.kind === 'videos') {
    return capability.resolutions.map(
      (value) =>
        VIDEOS_API_CREATION_RESOLUTION_OPTIONS.find(
          (item) => item.value === value
        ) ?? getResolutionOption(value)
    )
  }
  if (capability?.kind === 'video2') {
    return capability.resolutions.includes('480p')
      ? VIDEO2_480P_CREATION_RESOLUTION_OPTIONS
      : VIDEO2_720P_CREATION_RESOLUTION_OPTIONS
  }
  if (capability?.kind === 'minimax-h3') {
    return capability.resolutions.map(getResolutionOption)
  }
  if (capability?.kind === 'sora2') return SORA2_CREATION_RESOLUTION_OPTIONS
  return CREATION_RESOLUTION_OPTIONS
}

export function getCreationDurationOptions(model?: CreationModelInput) {
  const capability = getCreationVideoCapabilities(model)
  if (capability?.kind === 'sanbao' && capability.durations.length) {
    return capability.durations.map(getDurationOption)
  }
  if (capability?.kind === 'videos') {
    return capability.durations.map(getDurationOption)
  }
  if (capability?.kind === 'video2') {
    return capability.durations.map(getDurationOption)
  }
  if (capability?.kind === 'minimax-h3') {
    return capability.durations.map(getDurationOption)
  }
  if (capability?.kind === 'sora2') return SORA2_CREATION_DURATION_OPTIONS
  return CREATION_DURATION_OPTIONS
}

export function normalizeCreationVideoOptions(
  options: CreationVideoOptions,
  model?: CreationModelInput
): CreationVideoOptions {
  const resolutionOptions = getCreationResolutionOptions(model)
  const matchedResolution = resolutionOptions.find(
    (item) => item.value === options.resolution
  )
  const resolution = matchedResolution ?? resolutionOptions[0]
  const durationOptions = getCreationDurationOptions(model)
  const capability = getCreationVideoCapabilities(model)
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
  model?: CreationModelInput
) {
  if (!getCreationVideoCapabilities(model)) return undefined
  const duration = Number(options.duration)
  const capability = getCreationVideoCapabilities(model)
  if (
    !Number.isInteger(duration) ||
    !capability?.durations.includes(String(duration))
  ) {
    if (capability?.kind === 'sanbao' || capability?.kind === 'minimax-h3') {
      return 'This model does not support the selected duration.'
    }
    if (isSeedance25Model(model)) {
      return 'Seedance 2.5 duration must be between 4 and 29 seconds.'
    }
    if (isVideo25Model(model)) {
      return 'Video 2.5 duration must be between 4 and 30 seconds.'
    }
    return 'Duration must be between 4 and 15 seconds.'
  }
  return undefined
}

export function getCreationVideoReferenceLimits(
  model?: CreationModelInput
): CreationVideoReferenceLimits {
  return (
    getCreationVideoCapabilities(model)?.referenceLimits ??
    VIDEO2_REFERENCE_LIMITS
  )
}

function emptyCreationVideoReferences(): CreationVideoReferences {
  return {
    referenceMode: 'text',
    imageUrls: [],
    startImageUrl: '',
    endImageUrl: '',
    videoUrls: [],
    audioUrls: [],
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

export function getCreationReferenceSizeBytes(
  value?: CreationVideoReferenceValue | null
) {
  if (!value || typeof value === 'string') return 0
  return Number.isFinite(value.sizeBytes) && Number(value.sizeBytes) > 0
    ? Number(value.sizeBytes)
    : 0
}

export function getCreationReferenceDurationSeconds(
  value?: CreationVideoReferenceValue | null
) {
  if (!value || typeof value === 'string') return 0
  return Number.isFinite(value.durationSeconds) &&
    Number(value.durationSeconds) > 0
    ? Number(value.durationSeconds)
    : 0
}

function cleanReferenceValues(
  values: CreationVideoReferenceValue[] | undefined
) {
  return (values ?? []).filter((value) => getCreationReferenceURL(value))
}

function mergeReferenceValues(
  values: CreationVideoReferenceValue[] | undefined,
  fallback?: CreationVideoReferenceValue
) {
  const items = cleanReferenceValues(values)
  const fallbackURL = getCreationReferenceURL(fallback)
  if (
    fallbackURL &&
    !items.some((value) => getCreationReferenceURL(value) === fallbackURL)
  ) {
    items.push(fallback as CreationVideoReferenceValue)
  }
  return items
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
    : (capability.referenceModes[0] ?? 'text')
}

export function normalizeCreationVideoReferences(
  references?: Partial<CreationVideoReferences>,
  model?: CreationModelInput
): CreationVideoReferences {
  const capability = getCreationVideoCapabilities(model)
  if (!capability) return emptyCreationVideoReferences()

  const referenceMode = normalizeReferenceMode(
    references?.referenceMode,
    capability
  )
  const supportsMultiMedia =
    referenceMode === 'multimodal' && capability.referenceLimits.maxAudios > 0
  const supportsImageAudio =
    referenceMode === 'image-audio' && capability.referenceLimits.maxAudios > 0
  const imageUrls =
    referenceMode === 'image' || supportsMultiMedia || supportsImageAudio
      ? cleanReferenceValues(references?.imageUrls ?? [])
      : []
  const videoUrls =
    referenceMode === 'video' || supportsMultiMedia
      ? cleanReferenceValues(references?.videoUrls ?? [])
      : []
  const audioUrls =
    (supportsMultiMedia || supportsImageAudio) &&
    capability.referenceLimits.maxAudios > 0
      ? mergeReferenceValues(references?.audioUrls, references?.audioUrl)
      : []

  if (referenceMode === 'frames') {
    return {
      ...emptyCreationVideoReferences(),
      referenceMode,
      startImageUrl: normalizeReferenceString(references?.startImageUrl),
      endImageUrl: normalizeReferenceString(references?.endImageUrl),
    }
  }

  if (referenceMode === 'image') {
    return {
      ...emptyCreationVideoReferences(),
      referenceMode,
      imageUrls,
    }
  }

  if (capability.kind === 'videos') {
    const limitedAudioUrls = audioUrls.slice(
      0,
      capability.referenceLimits.maxAudios
    )
    const limitedImageUrls = imageUrls.slice(
      0,
      capability.referenceLimits.maxImages
    )
    const limitedVideoUrls = videoUrls.slice(
      0,
      capability.referenceLimits.maxVideos
    )
    return {
      referenceMode,
      imageUrls: limitedImageUrls,
      startImageUrl: '',
      endImageUrl: '',
      videoUrls: limitedVideoUrls,
      audioUrls: limitedAudioUrls,
      audioUrl: getCreationReferenceURL(limitedAudioUrls[0])
        ? limitedAudioUrls[0]
        : '',
    }
  }

  if (capability.kind === 'minimax-h3') {
    const limitedAudioUrls = audioUrls.slice(
      0,
      capability.referenceLimits.maxAudios
    )
    return {
      referenceMode,
      imageUrls,
      startImageUrl: '',
      endImageUrl: '',
      videoUrls: [],
      audioUrls: limitedAudioUrls,
      audioUrl: getCreationReferenceURL(limitedAudioUrls[0])
        ? limitedAudioUrls[0]
        : '',
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
    audioUrls: audioUrls.slice(0, capability.referenceLimits.maxAudios || 1),
    audioUrl: audioUrls[0] ?? '',
  }
}

export function filterCreationVideoReferencesByPromptMentions(
  prompt: string,
  references: CreationVideoReferences,
  model?: CreationModelInput
): CreationVideoReferences {
  const normalized = normalizeCreationVideoReferences(references, model)
  const mentionedImageUrls = normalized.imageUrls.filter((_, index) =>
    hasPromptMention(prompt, getReferenceMentionAliases('image', index + 1))
  )
  const mentionedVideoUrls = normalized.videoUrls.filter((_, index) =>
    hasPromptMention(prompt, getReferenceMentionAliases('video', index + 1))
  )
  const genericAudioMention = hasPromptMention(
    prompt,
    getReferenceMentionAliases('audio')
  )
  const mentionedAudioUrls = genericAudioMention
    ? normalized.audioUrls
    : normalized.audioUrls.filter((_, index) =>
        hasPromptMention(prompt, getReferenceMentionAliases('audio', index + 1))
      )
  const hasMentions =
    mentionedImageUrls.length > 0 ||
    mentionedVideoUrls.length > 0 ||
    mentionedAudioUrls.length > 0

  if (!hasMentions) return normalized

  return normalizeCreationVideoReferences(
    {
      ...normalized,
      imageUrls: mentionedImageUrls,
      startImageUrl: '',
      endImageUrl: '',
      videoUrls: mentionedVideoUrls,
      audioUrls: mentionedAudioUrls,
      audioUrl: mentionedAudioUrls[0] ?? '',
    },
    model
  )
}

function hasPromptMention(prompt: string, aliases: string[]) {
  const normalizedPrompt = prompt.toLocaleLowerCase()
  return aliases.some((alias) => {
    const mention = `@${alias.toLocaleLowerCase()}`
    let index = normalizedPrompt.indexOf(mention)
    while (index >= 0) {
      const next = normalizedPrompt[index + mention.length]
      if (!next || isReferenceMentionBoundary(next)) return true
      index = normalizedPrompt.indexOf(mention, index + mention.length)
    }
    return false
  })
}

function isReferenceMentionBoundary(value: string) {
  return !/[a-z0-9_-]/i.test(value)
}

function getReferenceMentionAliases(
  kind: 'image' | 'video' | 'audio',
  index?: number
) {
  if (kind === 'audio') {
    if (index === undefined) {
      return ['参考音频', '音频', 'reference audio', 'audio']
    }
    const number = index
    return [
      `参考音频${number}`,
      `音频${number}`,
      `reference audio ${number}`,
      `reference audio${number}`,
      `audio${number}`,
      `audio-${number}`,
    ]
  }
  const number = index ?? 1
  if (kind === 'video') {
    return [
      `参考视频${number}`,
      `视频${number}`,
      `reference video ${number}`,
      `reference video${number}`,
      `video${number}`,
      `video-${number}`,
    ]
  }
  return [
    `参考图片${number}`,
    `参考图${number}`,
    `图片${number}`,
    `reference image ${number}`,
    `reference image${number}`,
    `image${number}`,
    `image-${number}`,
  ]
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
  model: CreationModelInput,
  references: CreationVideoReferences
) {
  const capability = getCreationVideoCapabilities(model)
  if (!capability) return undefined

  const normalized = normalizeCreationVideoReferences(references, model)
  const imageCount =
    normalized.imageUrls.length +
    (normalized.startImageUrl ? 1 : 0) +
    (normalized.endImageUrl ? 1 : 0)
  const audioCount = normalized.audioUrls.length
  const referenceLimits = capability.referenceLimits
  if (
    capability.kind === 'minimax-h3' &&
    normalized.referenceMode === 'image' &&
    normalized.imageUrls.length === 0
  ) {
    return 'Image reference mode requires at least one image reference.'
  }
  if (
    capability.kind === 'minimax-h3' &&
    normalized.referenceMode === 'frames' &&
    (!normalized.startImageUrl || !normalized.endImageUrl)
  ) {
    return 'Start/end frame mode requires both a start frame and an end frame.'
  }
  if (
    capability.kind === 'minimax-h3' &&
    normalized.referenceMode === 'image-audio' &&
    audioCount > 0 &&
    normalized.imageUrls.length === 0
  ) {
    return 'Audio reference requires at least one image reference.'
  }
  if (
    capability.kind === 'minimax-h3' &&
    normalized.referenceMode === 'image-audio' &&
    (normalized.imageUrls.length === 0 || audioCount === 0)
  ) {
    return 'Image and audio mode requires at least one image and one audio reference.'
  }
  if (capability.kind === 'sora2' && imageCount > 1) {
    return 'Sora2 accepts at most 1 reference image.'
  }
  if (imageCount > referenceLimits.maxImages) {
    if (capability.kind === 'sanbao') {
      return 'Sanbao accepts too many reference images.'
    }
    if (capability.kind === 'videos') {
      if (isSeedance25Model(model)) {
        return 'Seedance 2.5 accepts at most 30 image references.'
      }
      if (isSeedance2Model(model)) {
        return 'Seedance accepts at most 9 image references.'
      }
      return 'Videos API accepts at most 9 image references.'
    }
    if (capability.kind === 'minimax-h3') {
      return 'MiniMax H3 accepts at most 5 image references.'
    }
    if (isVideo25Model(model)) {
      return 'Video 2.5 accepts at most 30 image references.'
    }
    return 'Video2 accepts at most 4 image references.'
  }
  if (normalized.videoUrls.length > referenceLimits.maxVideos) {
    if (capability.kind === 'sanbao') {
      return 'Sanbao accepts too many reference videos.'
    }
    if (capability.kind === 'videos') {
      if (isSeedance25Model(model)) {
        return 'Seedance 2.5 accepts at most 10 reference videos.'
      }
      if (isSeedance2Model(model)) {
        return 'Seedance accepts at most 3 reference videos.'
      }
      return 'Videos API accepts at most 3 reference videos.'
    }
    if (isVideo25Model(model)) {
      return 'Video 2.5 accepts at most 10 reference videos.'
    }
    return 'Video2 accepts at most 3 video references.'
  }
  if (audioCount > referenceLimits.maxAudios) {
    if (capability.kind === 'videos') {
      if (isSeedance25Model(model)) {
        return 'Seedance 2.5 accepts at most 10 reference audios.'
      }
      if (isSeedance2Model(model)) {
        return 'Seedance accepts at most 3 reference audios.'
      }
      return 'Videos API accepts at most 3 reference audios.'
    }
    if (capability.kind === 'sanbao') {
      return 'Sanbao accepts too many reference audios.'
    }
    if (isVideo25Model(model)) {
      return 'Video 2.5 accepts at most 10 reference audios.'
    }
    return 'Video2 accepts at most 1 audio reference.'
  }
  const mediaCount = imageCount + normalized.videoUrls.length + audioCount
  if (
    referenceLimits.maxMediaFiles &&
    mediaCount > referenceLimits.maxMediaFiles
  ) {
    if (capability.kind === 'videos') {
      if (isSeedance25Model(model)) {
        return 'Seedance 2.5 accepts too many reference assets.'
      }
      if (isSeedance2Model(model)) {
        return 'Seedance accepts too many reference assets.'
      }
      return 'Videos API accepts too many reference assets.'
    }
    if (isVideo25Model(model)) {
      return 'Video 2.5 accepts too many reference assets.'
    }
    return 'Sanbao accepts too many reference assets.'
  }

  if (isSeedance25Model(model) || isVideo25Model(model)) {
    const minVideoDuration =
      referenceLimits.minReferenceVideoDurationSeconds ?? 0
    const maxVideoDuration =
      referenceLimits.maxReferenceVideoDurationSeconds ?? Infinity
    const invalidVideoDuration = normalized.videoUrls.some((value) => {
      const duration = getCreationReferenceDurationSeconds(value)
      return (
        duration > 0 &&
        (duration < minVideoDuration || duration > maxVideoDuration)
      )
    })
    if (invalidVideoDuration) {
      if (isVideo25Model(model)) {
        return 'Video 2.5 reference videos must be between 3 and 10 seconds each.'
      }
      return 'Seedance 2.5 reference videos must be between 2 and 30 seconds each.'
    }

    const minAudioDuration =
      referenceLimits.minReferenceAudioDurationSeconds ?? 0
    const maxAudioDuration =
      referenceLimits.maxReferenceAudioDurationSeconds ?? Infinity
    const invalidAudioDuration = normalized.audioUrls.some((value) => {
      const duration = getCreationReferenceDurationSeconds(value)
      return (
        duration > 0 &&
        (duration < minAudioDuration || duration > maxAudioDuration)
      )
    })
    if (invalidAudioDuration) {
      if (isVideo25Model(model)) {
        return 'Video 2.5 reference audios must be between 3 and 30 seconds each.'
      }
      return 'Seedance 2.5 reference audios must be between 2 and 30 seconds each.'
    }
  }

  const videoTotalDurationSeconds = normalized.videoUrls.reduce(
    (total, value) => total + getCreationReferenceDurationSeconds(value),
    0
  )
  if (
    referenceLimits.maxReferenceVideoTotalDurationSeconds &&
    videoTotalDurationSeconds >
      referenceLimits.maxReferenceVideoTotalDurationSeconds
  ) {
    if (isVideo25Model(model)) {
      return 'Video 2.5 reference videos must not exceed 30 seconds in total.'
    }
    return 'Seedance 2.5 reference videos must not exceed 30 seconds in total.'
  }
  const audioTotalDurationSeconds = normalized.audioUrls.reduce(
    (total, value) => total + getCreationReferenceDurationSeconds(value),
    0
  )
  if (
    referenceLimits.maxReferenceAudioTotalDurationSeconds &&
    audioTotalDurationSeconds >
      referenceLimits.maxReferenceAudioTotalDurationSeconds
  ) {
    if (isVideo25Model(model)) {
      return 'Video 2.5 reference audios must not exceed 30 seconds in total.'
    }
    return 'Seedance 2.5 reference audios must not exceed 30 seconds in total.'
  }

  const images = [
    ...normalized.imageUrls.map(getCreationReferenceURL),
    normalized.startImageUrl,
    normalized.endImageUrl,
  ].filter(Boolean)
  if (images.some((url) => !isReferenceImage(url))) {
    return 'Reference images must be images or HTTP URLs.'
  }
  if (isSeedance2Model(model) && images.some((url) => !isHTTPURL(url))) {
    return 'Reference URL must use HTTP or HTTPS.'
  }
  if (
    isSeedance25Model(model) &&
    images.some(
      (url) =>
        !hasAllowedReferenceFormat(
          url,
          SEEDANCE_25_REFERENCE_IMAGE_EXTENSIONS,
          SEEDANCE_25_REFERENCE_IMAGE_MIME_TYPES
        )
    )
  ) {
    return 'Seedance 2.5 reference image format must be JPG, PNG, or WebP.'
  }
  if (
    (capability.kind === 'video2' || capability.kind === 'sanbao') &&
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

  const videoUrls = normalized.videoUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
  const audioUrls = normalized.audioUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
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
          capability.kind === 'minimax-h3'
            ? MINIMAX_H3_REFERENCE_AUDIO_EXTENSIONS
            : VIDEO_REFERENCE_AUDIO_EXTENSIONS,
          capability.kind === 'minimax-h3'
            ? MINIMAX_H3_REFERENCE_AUDIO_MIME_TYPES
            : VIDEO_REFERENCE_AUDIO_MIME_TYPES
        )
    )
  ) {
    if (capability.kind === 'minimax-h3') {
      return 'Reference audio format must be MP3, WAV, M4A, AAC, OGG, or WebM.'
    }
    return 'Reference audio format must be MP3 or WAV.'
  }
  return undefined
}

export function getCreationVideoRequestOptions(
  options: CreationVideoOptions,
  model?: CreationModelInput,
  references: CreationVideoReferences = EMPTY_CREATION_VIDEO_REFERENCES
): CreationVideoRequestOptions {
  const normalizedOptions = normalizeCreationVideoOptions(options, model)
  const durationOptions = getCreationDurationOptions(model)
  const duration =
    durationOptions.find((item) => item.value === normalizedOptions.duration) ??
    durationOptions[0]
  const capability = getCreationVideoCapabilities(model)

  if (!capability) {
    const resolutionOptions = getCreationResolutionOptions(model)
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
      model
    )
    const request: LegacyCreationVideoRequestOptions = {
      seconds: duration.seconds,
      size: SORA2_VIDEO_SIZES[aspectRatio],
      aspect_ratio: aspectRatio,
      estimateSeconds: duration.estimateSeconds,
    }
    const imageReference = getCreationReferenceURL(
      normalizedReferences.imageUrls[0]
    )
    if (imageReference) request.input_reference = imageReference
    return {
      ...request,
    }
  }

  const normalizedReferences = normalizeCreationVideoReferences(
    references,
    model
  )
  if (capability.kind === 'sanbao') {
    const imageUrls = normalizedReferences.imageUrls
      .map(getCreationReferenceURL)
      .filter(Boolean)
    const videoUrls = normalizedReferences.videoUrls
      .map(getCreationReferenceURL)
      .filter(Boolean)
    const audioUrls = normalizedReferences.audioUrls
      .map(getCreationReferenceURL)
      .filter(Boolean)
    const audioUrl = audioUrls[0]
    const request: SanbaoCreationVideoRequestOptions = {
      duration: Number(normalizedOptions.duration),
      ratio: normalizedOptions.aspectRatio ?? capability.aspectRatios[0],
      resolution: normalizedOptions.resolution,
      concurrency: capability.concurrencyOptions?.[0] ?? 1,
      estimateSeconds: duration.estimateSeconds,
    }
    if (imageUrls.length) request.images = imageUrls
    if (videoUrls.length) request.videos = videoUrls
    if (audioUrl) request.audios = [audioUrl]
    if (imageUrls.length || videoUrls.length || audioUrl) {
      request.reference = 'all'
    }
    return request
  }
  const imageUrls = normalizedReferences.imageUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
  const videoUrls = normalizedReferences.videoUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
  const audioUrls = normalizedReferences.audioUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)

  if (capability.kind === 'videos') {
    const request: VideosApiCreationVideoRequestOptions = {
      duration: Number(normalizedOptions.duration),
      ratio: normalizedOptions.aspectRatio ?? capability.aspectRatios[0],
      estimateSeconds: duration.estimateSeconds,
    }
    if (capability.includeResolutionInRequest) {
      request.resolution =
        normalizedOptions.resolution === '480p' ? '480p' : '720p'
    }
    if (isSeedance25Model(model)) {
      if (imageUrls.length) request.referenceImages = imageUrls
      if (videoUrls.length) request.referenceVideos = videoUrls
      if (audioUrls.length) request.referenceAudios = audioUrls
    } else if (isSeedance2Model(model)) {
      if (normalizedReferences.referenceMode === 'image') {
        const [firstImage, ...referenceImages] = imageUrls
        if (firstImage) request.start_image_url = firstImage
        if (referenceImages.length) request.referenceImages = referenceImages
      } else if (normalizedReferences.referenceMode === 'frames') {
        if (normalizedReferences.startImageUrl) {
          request.start_image_url = normalizedReferences.startImageUrl
        }
        if (normalizedReferences.endImageUrl) {
          request.end_image_url = normalizedReferences.endImageUrl
        }
      } else {
        if (imageUrls.length) request.referenceImages = imageUrls
        if (videoUrls.length) request.referenceVideos = videoUrls
        if (audioUrls.length) request.referenceAudios = audioUrls
      }
    } else {
      if (imageUrls.length) request.referenceImages = imageUrls
      if (videoUrls.length) request.referenceVideos = videoUrls
      if (audioUrls.length) request.referenceAudios = audioUrls
    }
    return request
  }

  if (capability.kind === 'minimax-h3') {
    const request: MiniMaxH3CreationVideoRequestOptions = {
      duration: Number(normalizedOptions.duration),
      aspect_ratio: normalizedOptions.aspectRatio ?? capability.aspectRatios[0],
      estimateSeconds: duration.estimateSeconds,
    }
    if (normalizedReferences.referenceMode === 'frames') {
      if (normalizedReferences.startImageUrl) {
        request.start_image_url = normalizedReferences.startImageUrl
      }
      if (normalizedReferences.endImageUrl) {
        request.end_image_url = normalizedReferences.endImageUrl
      }
      return request
    }
    if (imageUrls.length === 1) {
      request.image_url = imageUrls[0]
    } else if (imageUrls.length > 1) {
      request.image_urls = imageUrls
    }
    if (
      normalizedReferences.referenceMode === 'image-audio' &&
      audioUrls.length
    ) {
      request.audio_url = audioUrls[0]
    }
    return request
  }

  const request: Video2CreationVideoRequestOptions = {
    duration: Number(normalizedOptions.duration),
    aspect_ratio: normalizedOptions.aspectRatio ?? capability.aspectRatios[0],
    resolution: normalizedOptions.resolution === '480p' ? '480p' : '720p',
    async: true,
    estimateSeconds: duration.estimateSeconds,
  }

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
  if (audioUrls.length === 1) {
    request.audio_url = audioUrls[0]
  } else if (audioUrls.length > 1) {
    request.audio_reference = audioUrls.map((url) => ({ url }))
  }

  return request
}
