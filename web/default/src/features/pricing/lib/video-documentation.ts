import type { IntegrationParameters, PricingModel } from '../types'

export function isVideoCapabilityUnknown(
  parameters: IntegrationParameters,
  field: string
): boolean {
  return Boolean(
    parameters.capabilities_unknown ||
    parameters.unknown_fields?.includes(field)
  )
}

export function getVideoDocumentationParameters(
  model: PricingModel
): IntegrationParameters | undefined {
  const endpoint = model.mapped_endpoints?.find(
    (entry) =>
      entry.kind === 'video' &&
      Boolean(entry.documentation_fields ?? entry.parameters)
  )
  return endpoint?.documentation_fields ?? endpoint?.parameters
}

export function buildVideoDocumentationBody(
  modelName: string,
  prompt: string,
  parameters?: IntegrationParameters,
  example?: {
    duration?: number
    resolution?: string
    imageRequired?: boolean
    includeReferenceMedia?: boolean
  }
): Record<string, unknown> {
  const body: Record<string, unknown> = { model: modelName, prompt }
  const p = parameters
  const duration = p?.durations?.includes(example?.duration ?? -1)
    ? example?.duration
    : p?.durations?.[0]
  if (duration && p?.duration_field) body[p.duration_field] = duration
  if (!p && example?.duration) body.duration = example.duration
  body[p?.ratio_field || 'ratio'] = p?.ratios?.[0] || '16:9'
  const resolution = p?.resolutions?.includes(example?.resolution || '')
    ? example?.resolution
    : p?.request_resolution || p?.resolutions?.[0]
  if (resolution && p?.resolution_field) body[p.resolution_field] = resolution
  if (!p && example?.resolution) body.resolution = example.resolution
  const includeReferences =
    example?.includeReferenceMedia && hasVideoReferenceMedia(p)
  if (
    p?.image_field &&
    p.max_images > 0 &&
    !isVideoCapabilityUnknown(p, 'max_images') &&
    ((p.min_images ?? 0) > 0 || example?.imageRequired || includeReferences)
  ) {
    const images = Array.from(
      { length: Math.max(1, p.min_images ?? 0) },
      (_, index) =>
        `https://cdn.example.com/reference${index === 0 ? '' : `-${index + 1}`}.jpg`
    )
    body[p.image_field] = p.image_field.endsWith('_url') ? images[0] : images
  }
  if (includeReferences && p) {
    for (const item of [
      {
        field: p.video_field,
        maximum: p.max_videos,
        capability: 'max_videos',
        url: 'https://cdn.example.com/reference.mp4',
      },
      {
        field: p.audio_field,
        maximum: p.max_audios,
        capability: 'max_audios',
        url: 'https://cdn.example.com/reference.mp3',
      },
    ]) {
      if (
        item.field &&
        item.maximum > 0 &&
        !isVideoCapabilityUnknown(p, item.capability)
      ) {
        body[item.field] = item.field.endsWith('_url') ? item.url : [item.url]
      }
    }
  }
  if (!p && example?.imageRequired) {
    body.referenceImages = ['https://cdn.example.com/reference.jpg']
  }
  return body
}

export function hasVideoReferenceMedia(
  parameters?: IntegrationParameters
): boolean {
  if (!parameters || parameters.capabilities_unknown) return false
  return Boolean(
    (parameters.image_field &&
      parameters.max_images > 0 &&
      !isVideoCapabilityUnknown(parameters, 'max_images')) ||
    (parameters.video_field &&
      parameters.max_videos > 0 &&
      !isVideoCapabilityUnknown(parameters, 'max_videos')) ||
    (parameters.audio_field &&
      parameters.max_audios > 0 &&
      !isVideoCapabilityUnknown(parameters, 'max_audios'))
  )
}
