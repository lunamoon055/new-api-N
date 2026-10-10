// The current site's model square is the only directory source.
// Local documentation enriches matching IDs; it never adds directory entries.
export function normalizeCatalog(payload) {
  if (!payload?.success || !Array.isArray(payload.data)) {
    throw new Error('模型目录返回格式不完整')
  }
  const seen = new Set()
  const models = payload.data.filter((model) => {
    const name = model?.model_name
    if (typeof name !== 'string' || !name.trim() || seen.has(name)) return false
    seen.add(name)
    return true
  })
  return { models, endpoints: payload.supported_endpoint || {} }
}

export function resolveSelectedModel(models, current) {
  if (models.some((model) => model.model_name === current)) return current
  if (models.some((model) => model.model_name === 'gpt-image-2')) return 'gpt-image-2'
  return models[0]?.model_name || ''
}
