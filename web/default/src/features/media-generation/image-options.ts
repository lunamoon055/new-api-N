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
  type CreationVideoReferenceValue,
} from './video-options'

export type CreationImageReferenceValue = CreationVideoReferenceValue

export type CreationImageReferences = {
  imageUrls: CreationImageReferenceValue[]
}

export type CreationImageAspectRatio =
  '1:1' | '16:9' | '9:16' | '4:3' | '3:4' | '3:2' | '2:3'

export type CreationImageOptions = {
  aspectRatio: CreationImageAspectRatio
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

function normalizeModelId(modelId?: string) {
  return modelId?.trim().toLowerCase() ?? ''
}

export function supportsCreationImageReferences(modelId?: string) {
  return normalizeModelId(modelId) === 'gpt-image2'
}

export function normalizeCreationImageReferences(
  references?: Partial<CreationImageReferences>,
  modelId?: string
): CreationImageReferences {
  if (!supportsCreationImageReferences(modelId)) {
    return { ...EMPTY_CREATION_IMAGE_REFERENCES }
  }
  return {
    imageUrls: cleanReferenceValues(references?.imageUrls),
  }
}

export function normalizeCreationImageOptions(
  options?: Partial<CreationImageOptions>,
  modelId?: string
): CreationImageOptions {
  if (!supportsCreationImageReferences(modelId)) {
    return { ...DEFAULT_CREATION_IMAGE_OPTIONS }
  }
  const aspectRatio = CREATION_IMAGE_ASPECT_RATIO_OPTIONS.includes(
    options?.aspectRatio as CreationImageAspectRatio
  )
    ? (options?.aspectRatio as CreationImageAspectRatio)
    : DEFAULT_CREATION_IMAGE_OPTIONS.aspectRatio

  return { aspectRatio }
}

export function getCreationImageReferenceError(
  modelId: string | undefined,
  references: CreationImageReferences
) {
  if (!supportsCreationImageReferences(modelId)) return undefined

  const normalized = normalizeCreationImageReferences(references, modelId)
  if (normalized.imageUrls.length > CREATION_IMAGE_REFERENCE_MAX_COUNT) {
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
  modelId?: string,
  references: CreationImageReferences = EMPTY_CREATION_IMAGE_REFERENCES,
  imageOptions: Partial<CreationImageOptions> = DEFAULT_CREATION_IMAGE_OPTIONS
): CreationImageRequestOptions {
  if (!supportsCreationImageReferences(modelId)) return {}

  const normalizedOptions = normalizeCreationImageOptions(imageOptions, modelId)
  const normalized = normalizeCreationImageReferences(references, modelId)
  const imageUrls = normalized.imageUrls
    .map(getCreationReferenceURL)
    .filter(Boolean)
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
