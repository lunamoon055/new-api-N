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
  getCreationReferenceURL,
  type CreationModelInput,
  type CreationModelMetadataLike,
  type CreationVideoReferenceValue,
} from './video-options'

export type CreationImageReferenceValue = CreationVideoReferenceValue

export type CreationImageReferences = {
  imageUrls: CreationImageReferenceValue[]
}

export type CreationImageAspectRatio = string

export type CreationImageOptions = {
  aspectRatio: CreationImageAspectRatio
}

export type CreationImageReferenceLimits = {
  maxImages: number
  maxImageSizeBytes: number
  maxImageSizeMB: number
}

type CreationImageMessageContent =
  | { type: 'text'; text: string }
  | { type: 'image_url'; image_url: { url: string } }

export type CreationImageRequestOptions =
  | Record<string, never>
  | {
      output_resolution: '1K'
      aspect_ratio: CreationImageAspectRatio
      messages?: Array<{
        role: 'user'
        content: CreationImageMessageContent[]
      }>
    }
  | {
      aspect_ratio: CreationImageAspectRatio
      images?: string[]
      quality?: 'high' | 'medium' | 'low'
      concurrency?: number
    }

export const CREATION_IMAGE_REFERENCE_MAX_COUNT = 6
export const CREATION_IMAGE_REFERENCE_MAX_BYTES = 20 * 1024 * 1024
export const CREATION_IMAGE_ASPECT_RATIO_OPTIONS: CreationImageAspectRatio[] = [
  '1:1',
  '16:9',
  '9:16',
  '4:3',
  '3:4',
  '3:2',
  '2:3',
]

export const EMPTY_CREATION_IMAGE_REFERENCES: CreationImageReferences = {
  imageUrls: [],
}

export const DEFAULT_CREATION_IMAGE_OPTIONS: CreationImageOptions = {
  aspectRatio: '1:1',
}

const IMAGE_REFERENCE_EXTENSIONS = ['avif', 'gif', 'jpeg', 'jpg', 'png', 'webp']
const IMAGE_REFERENCE_MIME_TYPES = [
  'image/avif',
  'image/gif',
  'image/jpeg',
  'image/png',
  'image/webp',
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

function isSanbaoImageModel(model?: CreationModelInput) {
  const metadata = getModelMetadata(model)
  if (metadata?.provider?.trim().toLowerCase() !== 'sanbao') return false
  return getSanbaoMediaType(metadata)?.includes('image') ?? false
}

function getSanbaoMediaType(metadata?: CreationModelMetadataLike) {
  return [metadata?.type, metadata?.category, metadata?.model_type]
    .map((value) => value?.trim().toLowerCase() ?? '')
    .find(Boolean)
}

function cleanAspectRatioOptions(values?: string[]) {
  const seen = new Set<string>()
  return (values ?? []).flatMap((value) => {
    const normalized = value.trim()
    if (!normalized || seen.has(normalized)) return []
    seen.add(normalized)
    return [normalized]
  })
}

export function supportsCreationImageReferences(model?: CreationModelInput) {
  return normalizeModelId(model) === 'gpt-image2' || isSanbaoImageModel(model)
}

export function getCreationImageReferenceLimits(
  model?: CreationModelInput
): CreationImageReferenceLimits {
  const metadata = getModelMetadata(model)
  if (isSanbaoImageModel(model)) {
    const maxImageSizeMB = metadata?.max_image_size_mb ?? 20
    return {
      maxImages: metadata?.max_images ?? CREATION_IMAGE_REFERENCE_MAX_COUNT,
      maxImageSizeMB,
      maxImageSizeBytes: maxImageSizeMB * 1024 * 1024,
    }
  }
  return {
    maxImages: CREATION_IMAGE_REFERENCE_MAX_COUNT,
    maxImageSizeMB: 20,
    maxImageSizeBytes: CREATION_IMAGE_REFERENCE_MAX_BYTES,
  }
}

export function getCreationImageAspectRatioOptions(
  model?: CreationModelInput
): CreationImageAspectRatio[] {
  const metadata = getModelMetadata(model)
  if (isSanbaoImageModel(model)) {
    const values = cleanAspectRatioOptions(
      metadata?.aspect_ratios?.length
        ? metadata.aspect_ratios
        : metadata?.ratios
    )
    if (values.length) return values
  }
  return CREATION_IMAGE_ASPECT_RATIO_OPTIONS
}

export function normalizeCreationImageReferences(
  references?: Partial<CreationImageReferences>,
  model?: CreationModelInput
): CreationImageReferences {
  if (!supportsCreationImageReferences(model)) {
    return { ...EMPTY_CREATION_IMAGE_REFERENCES }
  }
  return {
    imageUrls: cleanReferenceValues(references?.imageUrls),
  }
}

export function normalizeCreationImageOptions(
  options?: Partial<CreationImageOptions>,
  model?: CreationModelInput
): CreationImageOptions {
  if (!supportsCreationImageReferences(model)) {
    return { ...DEFAULT_CREATION_IMAGE_OPTIONS }
  }
  const aspectRatioOptions = getCreationImageAspectRatioOptions(model)
  const aspectRatio = aspectRatioOptions.includes(
    options?.aspectRatio as CreationImageAspectRatio
  )
    ? (options?.aspectRatio as CreationImageAspectRatio)
    : (aspectRatioOptions[0] ?? DEFAULT_CREATION_IMAGE_OPTIONS.aspectRatio)

  return { aspectRatio }
}

export function getCreationImageReferenceError(
  model: CreationModelInput,
  references: CreationImageReferences
) {
  if (!supportsCreationImageReferences(model)) return undefined

  const normalized = normalizeCreationImageReferences(references, model)
  const limits = getCreationImageReferenceLimits(model)
  if (normalized.imageUrls.length > limits.maxImages) {
    if (isSanbaoImageModel(model)) {
      return 'Sanbao accepts too many reference images.'
    }
    return 'Gpt-image2 accepts at most 6 reference images.'
  }

  const imageUrls = normalized.imageUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
  if (imageUrls.some((url) => !isReferenceImage(url))) {
    return 'Reference images must be images or HTTP URLs.'
  }
  if (
    imageUrls.some(
      (url) =>
        !hasAllowedReferenceFormat(
          url,
          IMAGE_REFERENCE_EXTENSIONS,
          IMAGE_REFERENCE_MIME_TYPES
        )
    )
  ) {
    return 'Reference image format must be PNG, JPEG, WebP, GIF, or AVIF.'
  }

  return undefined
}

export function getCreationImageRequestOptions(
  prompt: string,
  model?: CreationModelInput,
  references: CreationImageReferences = EMPTY_CREATION_IMAGE_REFERENCES,
  imageOptions: Partial<CreationImageOptions> = DEFAULT_CREATION_IMAGE_OPTIONS
): CreationImageRequestOptions {
  if (!supportsCreationImageReferences(model)) return {}

  const normalizedOptions = normalizeCreationImageOptions(imageOptions, model)
  const normalized = normalizeCreationImageReferences(references, model)
  const imageUrls = normalized.imageUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
  if (isSanbaoImageModel(model)) {
    return {
      aspect_ratio: normalizedOptions.aspectRatio,
      ...(imageUrls.length ? { images: imageUrls } : {}),
      quality: 'high',
      concurrency: 1,
    }
  }
  const options: Exclude<CreationImageRequestOptions, Record<string, never>> = {
    output_resolution: '1K',
    aspect_ratio: normalizedOptions.aspectRatio,
  }

  if (imageUrls.length) {
    options.messages = [
      {
        role: 'user',
        content: [
          { type: 'text', text: prompt },
          ...imageUrls.map((url) => ({
            type: 'image_url' as const,
            image_url: { url },
          })),
        ],
      },
    ]
  }

  return options
}

function cleanReferenceValues(
  values: CreationImageReferenceValue[] | undefined
) {
  return (values ?? []).filter((value) => getCreationReferenceURL(value))
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

function hasAllowedReferenceFormat(
  value: string,
  extensions: string[],
  mimeTypes: string[]
) {
  const mime = getDataURLMime(value)
  if (mime) return mimeTypes.includes(mime)
  const extension = getURLFileExtension(value)
  return !!extension && extensions.includes(extension)
}
