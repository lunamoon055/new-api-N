import type { TaskLog } from '../types'
import {
  getTaskLogModelName,
  getTaskLogPrompt,
  getVisibleTaskLogInputMaterials,
} from './task-preview'

export function getTaskLogRequestBody(log: TaskLog, canViewPrivate: boolean) {
  const snapshot = canViewPrivate ? log.request_body_snapshot : undefined
  if (snapshot) {
    const text =
      snapshot.body_json ??
      (snapshot.body === undefined
        ? ''
        : JSON.stringify(snapshot.body, null, 2))
    return {
      text,
      preview: text?.replace(/\n\s*/g, ' '),
      snapshot,
      notice: undefined,
    }
  }

  // Historical tasks never stored the original parameters. Show only known
  // fields, and explicitly identify this as an incomplete record in the dialog.
  const body: Record<string, unknown> = {}
  const model = getTaskLogModelName(log)
  const prompt = getTaskLogPrompt(log)
  if (model) body.model = model
  if (prompt) body.prompt = prompt
  const materials = getVisibleTaskLogInputMaterials(log, canViewPrivate)
  for (const [kind, field] of [
    ['image', 'referenceImages'],
    ['video', 'referenceVideos'],
    ['audio', 'referenceAudios'],
  ] as const) {
    const urls = materials
      .filter((material) => material.kind === kind)
      .map((material) => material.url)
    if (urls.length) body[field] = urls
  }
  const hasFields = Object.keys(body).length > 0
  return {
    text: hasFields ? JSON.stringify(body, null, 2) : '',
    preview: hasFields ? JSON.stringify(body) : '',
    snapshot: undefined,
    notice: canViewPrivate ? ('historical' as const) : ('restricted' as const),
  }
}
