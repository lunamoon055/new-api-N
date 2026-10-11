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
import type { PlaygroundMediaResult } from './media-routing'

type PollOptions = {
  signal?: AbortSignal
  onProgress?: (result: PlaygroundMediaResult) => void
  fetch: (taskId: string) => Promise<PlaygroundMediaResult>
  wait?: (milliseconds: number, signal?: AbortSignal) => Promise<void>
  timeoutMs?: number
  now?: () => number
}

export async function pollPlaygroundTask(
  initial: PlaygroundMediaResult,
  options: PollOptions
) {
  if (!initial.taskId || initial.mediaUrl) return initial
  let latest = initial
  let failures = 0
  const now = options.now ?? Date.now
  const started = now()
  const wait = options.wait ?? waitForMediaTask
  while (now() - started < (options.timeoutMs ?? 30 * 60_000)) {
    await wait(Math.min(30_000, 5000 * 2 ** failures), options.signal)
    options.signal?.throwIfAborted()
    try {
      latest = {
        ...(await options.fetch(initial.taskId)),
        taskId: initial.taskId,
      }
      failures = 0
      options.onProgress?.(latest)
      if (
        latest.mediaUrl ||
        latest.status === 'completed' ||
        latest.status === 'failed'
      )
        return latest
    } catch (error) {
      options.signal?.throwIfAborted()
      const status = (error as { response?: { status?: number } }).response
        ?.status
      if (
        !status ||
        status === 408 ||
        status === 409 ||
        status === 425 ||
        status === 429 ||
        status >= 500
      ) {
        // Only retry transport/HTTP errors; parsing a failed task is terminal.
        if (!status && !(error as { isAxiosError?: boolean }).isAxiosError)
          throw error
        failures = Math.min(failures + 1, 3)
        continue
      }
      throw error
    }
  }
  return latest
}

export function waitForMediaTask(milliseconds: number, signal?: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    signal?.throwIfAborted()
    const abort = () => {
      clearTimeout(timer)
      reject(signal?.reason ?? new DOMException('Aborted', 'AbortError'))
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', abort)
      resolve()
    }, milliseconds)
    signal?.addEventListener('abort', abort, { once: true })
  })
}
