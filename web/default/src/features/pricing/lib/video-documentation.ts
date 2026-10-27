import type { IntegrationParameters, PricingModel } from '../types'

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
  example?: { duration?: number; resolution?: string; imageRequired?: boolean }
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
  if (p?.image_field && ((p.min_images ?? 0) > 0 || example?.imageRequired)) {
    const image = 'https://cdn.example.com/reference.jpg'
    body[p.image_field] = p.image_field.endsWith('_url') ? image : [image]
  }
  if (!p && example?.imageRequired) {
    body.referenceImages = ['https://cdn.example.com/reference.jpg']
  }
  return body
}
