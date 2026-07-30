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
import type { CreationMode, CreationResult } from '../types'

export type VideoGenerationWaitingPhase = 'submitting' | 'queued' | 'processing'

export function getVideoGenerationWaitingPhase({
  mode,
  submitting,
  result,
}: {
  mode: CreationMode
  submitting: boolean
  result?: CreationResult
}): VideoGenerationWaitingPhase | null {
  if (mode !== 'video') return null
  if (submitting) return 'submitting'
  if (result?.mode !== 'video' || result.videoUrl) return null

  if (result.status === 'queued' || result.status === 'processing') {
    return result.status
  }

  if (result.status === 'unknown' && result.taskId) {
    return 'queued'
  }

  return null
}
