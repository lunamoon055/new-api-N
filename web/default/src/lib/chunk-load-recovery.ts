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

const CHUNK_LOAD_RETRY_KEY = 'chunk-load-recovery'
const CHUNK_LOAD_RETRY_WINDOW_MS = 60_000

const CHUNK_LOAD_ERROR_PATTERNS = [
  /\bChunkLoadError\b/i,
  /\bLoading (?:CSS )?chunk\b.*\bfailed\b/i,
  /Failed to fetch dynamically imported module/i,
  /Failed to load dynamically imported module/i,
  /error loading dynamically imported module/i,
  /Importing a module script failed/i,
  /Failed to load module script/i,
]

type RecoveryStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

type ChunkLoadRecoveryEnvironment = {
  location: Pick<Location, 'href' | 'reload'>
  sessionStorage: RecoveryStorage
}

type ChunkLoadRetry = {
  timestamp: number
  url: string
}

function collectErrorText(error: unknown, depth = 0): string {
  if (depth > 2 || error === null || error === undefined) return ''
  if (typeof error === 'string') return error
  if (error instanceof Error) {
    return [error.name, error.message, collectErrorText(error.cause, depth + 1)]
      .filter(Boolean)
      .join('\n')
  }
  if (typeof error !== 'object') return String(error)

  const record = error as Record<string, unknown>
  return [
    collectErrorText(record.name, depth + 1),
    collectErrorText(record.message, depth + 1),
    collectErrorText(record.cause, depth + 1),
  ]
    .filter(Boolean)
    .join('\n')
}

function getBrowserEnvironment(): ChunkLoadRecoveryEnvironment | undefined {
  if (typeof window === 'undefined') return undefined
  return {
    location: window.location,
    sessionStorage: window.sessionStorage,
  }
}

function parseRetry(value: string | null): ChunkLoadRetry | undefined {
  if (!value) return undefined
  try {
    const parsed = JSON.parse(value) as Partial<ChunkLoadRetry>
    if (
      typeof parsed.timestamp !== 'number' ||
      typeof parsed.url !== 'string'
    ) {
      return undefined
    }
    return { timestamp: parsed.timestamp, url: parsed.url }
  } catch {
    return undefined
  }
}

export function isChunkLoadError(error: unknown): boolean {
  const errorText = collectErrorText(error)
  return CHUNK_LOAD_ERROR_PATTERNS.some((pattern) => pattern.test(errorText))
}

export function recoverFromChunkLoadError(
  error: unknown,
  environment = getBrowserEnvironment(),
  now = Date.now()
): boolean {
  if (!environment || !isChunkLoadError(error)) return false

  try {
    const currentUrl = environment.location.href
    const previousRetry = parseRetry(
      environment.sessionStorage.getItem(CHUNK_LOAD_RETRY_KEY)
    )
    const alreadyRetriedRecently =
      previousRetry?.url === currentUrl &&
      now - previousRetry.timestamp < CHUNK_LOAD_RETRY_WINDOW_MS

    if (alreadyRetriedRecently) return false

    environment.sessionStorage.setItem(
      CHUNK_LOAD_RETRY_KEY,
      JSON.stringify({
        timestamp: now,
        url: currentUrl,
      } satisfies ChunkLoadRetry)
    )
    environment.location.reload()
    return true
  } catch {
    // Do not reload when the retry guard cannot be persisted; this avoids loops
    // in browsers that block sessionStorage.
    return false
  }
}

export function clearChunkLoadRecovery(
  storage: RecoveryStorage | undefined = typeof window === 'undefined'
    ? undefined
    : window.sessionStorage
): void {
  try {
    storage?.removeItem(CHUNK_LOAD_RETRY_KEY)
  } catch {
    // sessionStorage can be unavailable in restricted browser contexts.
  }
}
