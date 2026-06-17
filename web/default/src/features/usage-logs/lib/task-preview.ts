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
import { TASK_ACTIONS, TASK_STATUS } from '../constants'
import type { TaskLog } from '../types'

const VIDEO_ACTIONS = new Set<string>([
  TASK_ACTIONS.GENERATE,
  TASK_ACTIONS.TEXT_GENERATE,
  TASK_ACTIONS.FIRST_TAIL_GENERATE,
  TASK_ACTIONS.REFERENCE_GENERATE,
  TASK_ACTIONS.REMIX_GENERATE,
])

export function isTaskLogVideoTask(log: Pick<TaskLog, 'action'>) {
  return VIDEO_ACTIONS.has(log.action)
}

export function getTaskLogVideoPreviewUrl(
  log: Pick<TaskLog, 'action' | 'status' | 'task_id' | 'result_url'>
) {
  if (log.status !== TASK_STATUS.SUCCESS || !isTaskLogVideoTask(log)) {
    return null
  }

  if (log.task_id) {
    return `/v1/videos/${encodeURIComponent(log.task_id)}/content`
  }

  return normalizePreviewUrl(log.result_url)
}

function normalizePreviewUrl(url: string | undefined) {
  const trimmed = url?.trim()
  if (!trimmed) return null
  if (trimmed.startsWith('http') || trimmed.startsWith('data:')) {
    return trimmed
  }
  return null
}
