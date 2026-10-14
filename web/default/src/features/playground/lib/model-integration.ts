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
import { t } from 'i18next'
import type { IntegrationParameters } from '@/features/pricing/types'
import type { ModelOption, PlaygroundModelInfo } from '../types'

export function getPlaygroundGroupModels(models: ModelOption[], group: string) {
  if (group === 'auto') return models
  return models.filter((model) => {
    const groups = model.modelInfo?.model.enable_groups
    return !groups || groups.includes(group)
  })
}

export function getPlaygroundIntegration(info?: PlaygroundModelInfo) {
  const mapped = info?.model.mapped_endpoints
  // Mapped routes already account for upstream protocol and retain public IDs.
  if (mapped?.length) {
    return (
      mapped.find(
        (endpoint) => endpoint.kind === 'image' || endpoint.kind === 'video'
      ) ?? mapped[0]
    )
  }
  for (const type of info?.model.supported_endpoint_types ?? []) {
    const endpoint = info?.endpoints[type]
    if (!endpoint?.path) continue
    const kind = endpoint.path.includes('/images/')
      ? 'image'
      : endpoint.path.includes('/video')
        ? 'video'
        : 'text'
    return { ...endpoint, type, kind, parameters: undefined }
  }
  return undefined
}

export type PlaygroundReferenceSelection = {
  aspectRatio?: string
  resolution?: string
  images?: string
  videos?: string
  audios?: string
}

export function parsePlaygroundReferenceURLs(value = '') {
  const urls = value.split(/\s+/).filter(Boolean)
  for (const value of urls) {
    let url: URL
    try {
      url = new URL(value)
    } catch {
      throw new Error(t('Reference URLs must use HTTP or HTTPS'))
    }
    if (
      !['http:', 'https:'].includes(url.protocol) ||
      url.username ||
      url.password
    ) {
      throw new Error(t('Reference URLs must use HTTP or HTTPS'))
    }
  }
  return urls
}

export function buildPlaygroundReferences(
  parameters: IntegrationParameters,
  selection: PlaygroundReferenceSelection
) {
  const images = parsePlaygroundReferenceURLs(selection.images)
  const videos = parsePlaygroundReferenceURLs(selection.videos)
  const audios = parsePlaygroundReferenceURLs(selection.audios)
  if (parameters.audio_requires_image && audios.length && !images.length) {
    throw new Error(t('Audio reference requires at least one image reference.'))
  }
  if (images.length < (parameters.min_images ?? 0)) {
    throw new Error(t('This model requires a reference image'))
  }
  if (
    parameters.audio_needs_image &&
    audios.length &&
    !images.length &&
    !videos.length
  ) {
    throw new Error(t('Audio references require an image or video reference'))
  }
  const payload: Record<string, unknown> = {}
  const references = [
    {
      urls: images,
      field: parameters.image_field,
      max: parameters.max_images,
      kind: t('Reference images'),
    },
    {
      urls: videos,
      field: parameters.video_field,
      max: parameters.max_videos,
      kind: t('Reference videos'),
    },
    {
      urls: audios,
      field: parameters.audio_field,
      max: parameters.max_audios,
      kind: t('Reference audio'),
    },
  ]
  for (const reference of references) {
    if (!reference.urls.length) continue
    if (!reference.field || reference.urls.length > reference.max) {
      throw new Error(
        t('Too many references for this model: {{kind}} (maximum {{count}})', {
          kind: reference.kind,
          count: reference.max,
        })
      )
    }
    payload[reference.field] = reference.field.endsWith('_url')
      ? reference.urls[0]
      : reference.urls
  }
  return payload
}

export function resolveIntegrationChoice(
  value: string | undefined,
  choices?: string[]
) {
  return (
    choices?.find((choice) => choice.toLowerCase() === value?.toLowerCase()) ??
    choices?.[0]
  )
}
