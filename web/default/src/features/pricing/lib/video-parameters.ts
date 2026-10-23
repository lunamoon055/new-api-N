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
import type { IntegrationParameters, PricingModel } from '../types'
import type { SupportedParameter } from './mock-stats'
import { getModelEndpointPreset } from './model-endpoint-presets'

type VideoParameters = {
  parameters: SupportedParameter[]
  hasDetails: boolean
}

export function isVideoModel(model: PricingModel): boolean {
  if (model.mapped_endpoints?.length) {
    return model.mapped_endpoints.some((endpoint) => endpoint.kind === 'video')
  }
  return (
    model.supported_endpoint_types?.includes('openai-video') === true ||
    model.output_modalities?.includes('video') === true ||
    getModelEndpointPreset(model.model_name)?.kind === 'video'
  )
}

function durationParameter(
  p: IntegrationParameters
): SupportedParameter | null {
  if (!p.duration_field || !p.durations?.length) return null
  const values = [...new Set(p.durations)].sort((a, b) => a - b)
  const contiguous = values.every((value, index) => value === values[0] + index)
  return {
    name: p.duration_field,
    type: p.duration_field === 'seconds' ? 'string' : 'integer',
    ...(contiguous && values.length > 2
      ? { range: `${values[0]} ~ ${values[values.length - 1]}` }
      : { enumValues: values.map(String) }),
    descriptionKey: 'Video length in seconds',
  }
}

/** Use the public, mapped relay contract; never guess limits from a model name. */
export function buildVideoSupportedParameters(
  model: PricingModel
): VideoParameters | undefined {
  if (!isVideoModel(model)) return undefined

  const parameters: SupportedParameter[] = [
    {
      name: 'model',
      type: 'string',
      enumValues: [model.model_name],
      required: true,
      descriptionKey: 'Video model ID used in the request',
    },
    {
      name: 'prompt',
      type: 'string',
      required: true,
      descriptionKey: 'Text description of the desired video',
    },
  ]
  const p = model.mapped_endpoints?.find(
    (endpoint) => endpoint.kind === 'video' && endpoint.parameters
  )?.parameters
  if (!p) return { parameters, hasDetails: false }

  const duration = durationParameter(p)
  if (duration) parameters.push(duration)
  if (p.ratio_field && p.ratios?.length) {
    parameters.push({
      name: p.ratio_field,
      type: 'string',
      enumValues: p.ratios,
      descriptionKey: 'Output aspect ratio',
    })
  }
  const resolutions = p.resolutions?.length
    ? p.resolutions
    : [p.request_resolution].filter((value): value is string => Boolean(value))
  if (p.resolution_field && resolutions.length) {
    parameters.push({
      name: p.resolution_field,
      type: 'string',
      enumValues: resolutions,
      descriptionKey: 'Output video resolution',
    })
  }

  const media = [
    {
      field: p.image_field,
      max: p.max_images,
      min: p.min_images ?? 0,
      size: p.max_image_size_mb,
      descriptionKey: 'Public HTTP(S) URLs for reference images.',
    },
    {
      field: p.video_field,
      max: p.max_videos,
      min: 0,
      size: p.max_video_size_mb,
      descriptionKey: 'Public HTTP(S) URLs for reference videos.',
    },
    {
      field: p.audio_field,
      max: p.max_audios,
      min: 0,
      size: p.max_audio_size_mb,
      descriptionKey: 'Public HTTP(S) URLs for reference audio.',
    },
  ]
  for (const item of media) {
    if (!item.field || item.max <= 0) continue
    const notes: NonNullable<SupportedParameter['notes']> = [
      { key: 'At most {{count}} references.', values: { count: item.max } },
    ]
    if (item.size && item.size > 0) {
      notes.push({
        key: 'Maximum file size: {{size}} MB.',
        values: { size: item.size },
      })
    }
    if (item.min > 0) {
      notes.push({
        key: 'At least {{count}} image references are required.',
        values: { count: item.min },
      })
    }
    if (item.field === p.audio_field) {
      if (p.audio_requires_image) {
        notes.push({ key: 'Audio references require an image reference.' })
      } else if (p.audio_needs_image) {
        notes.push({
          key: 'Audio references require an image or video reference',
        })
      }
    }
    parameters.push({
      name: item.field,
      type: ['image_url', 'video_url', 'audio_url'].includes(item.field)
        ? 'string'
        : 'array',
      required: item.min > 0 && !p.allow_frames,
      range: `${item.min} ~ ${item.max}`,
      descriptionKey: item.descriptionKey,
      notes,
    })
  }
  if (p.allow_frames && p.max_images > 0) {
    const notes = [
      {
        key: 'Image references and start/end frames share the same limit of {{count}} images.',
        values: { count: p.max_images },
      },
    ]
    parameters.push(
      {
        name: 'start_image_url',
        type: 'string',
        notes,
        descriptionKey: 'Starting frame image as a public HTTP(S) URL.',
      },
      {
        name: 'end_image_url',
        type: 'string',
        notes,
        descriptionKey: 'Ending frame image as a public HTTP(S) URL.',
      }
    )
  }
  return { parameters, hasDetails: parameters.length > 2 }
}
