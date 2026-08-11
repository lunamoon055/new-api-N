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
import type { TaskLog } from '../types'

export interface TaskFailureDetails {
  downstreamMessage: string
  upstreamRawError?: string
}

export function getTaskFailureDetails(
  log: Pick<TaskLog, 'fail_reason' | 'raw_fail_reason'>,
  canViewPrivateTaskDetails: boolean
): TaskFailureDetails {
  const downstreamMessage = log.fail_reason?.trim() ?? ''
  const upstreamRawError = canViewPrivateTaskDetails
    ? log.raw_fail_reason?.trim() || undefined
    : undefined

  return { downstreamMessage, upstreamRawError }
}
