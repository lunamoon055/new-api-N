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
import { describe, expect, it } from 'bun:test'
import {
  DEFAULT_CREATION_VIDEO_OPTIONS,
  formatCreationCountdown,
  getCreationDurationOptions,
  getCreationTimedOut,
  getCreationVideoRequestOptions,
  loadCreationHistory,
  saveCreationHistory,
  upsertCreationHistoryItem,
  type CreationHistoryStorage,
} from '../src/features/creation-center/session'

function createMemoryStorage(): CreationHistoryStorage {
  const values = new Map<string, string>()
  return {
    getItem: (key) => values.get(key) ?? null,
    removeItem: (key) => {
      values.delete(key)
    },
    setItem: (key, value) => {
      values.set(key, value)
    },
  }
}

describe('creation center session helpers', () => {
  it('maps video controls to async media request fields', () => {
    expect(
      getCreationVideoRequestOptions({ resolution: '2k', duration: '10' })
    ).toMatchObject({
      seconds: '10',
      size: '2560x1440',
    })

    expect(
      getCreationVideoRequestOptions(DEFAULT_CREATION_VIDEO_OPTIONS)
      .estimateSeconds
    ).toBeGreaterThan(0)
  })

  it('uses linksky sora2 duration options from the media API docs', () => {
    expect(getCreationDurationOptions('sora2').map((item) => item.value)).toEqual(
      ['4', '8', '12']
    )
    expect(
      getCreationVideoRequestOptions(
        { resolution: '1080p', duration: '8' },
        'sora2'
      )
    ).toMatchObject({
      seconds: '8',
      size: '1920x1080',
    })
    expect(
      getCreationVideoRequestOptions(
        { resolution: '1080p', duration: '5' },
        'sora2'
      ).seconds
    ).toBe('4')
  })

  it('keeps the latest history item first and updates duplicate tasks', () => {
    const items = upsertCreationHistoryItem(
      [
        {
          createdAt: 100,
          id: 'old-task',
          mode: 'video',
          model: 'sora2',
          prompt: 'old prompt',
          result: {
            mode: 'video',
            model: 'sora2',
            status: 'queued',
            taskId: 'old-task',
          },
        },
      ],
      {
        createdAt: 200,
        id: 'old-task',
        mode: 'video',
        model: 'sora2',
        prompt: 'updated prompt',
        result: {
          mode: 'video',
          model: 'sora2',
          status: 'processing',
          taskId: 'old-task',
        },
      }
    )

    expect(items).toHaveLength(1)
    expect(items[0].createdAt).toBe(200)
    expect(items[0].prompt).toBe('updated prompt')
    expect(items[0].result.status).toBe('processing')
  })

  it('saves and restores bounded browser history', () => {
    const storage = createMemoryStorage()
    const key = 'creation-history:test'
    const items = Array.from({ length: 25 }, (_, index) => ({
      createdAt: index,
      id: `task-${index}`,
      mode: 'video' as const,
      model: 'kling-v3',
      prompt: `prompt ${index}`,
      result: {
        mode: 'video' as const,
        model: 'kling-v3',
        status: 'queued' as const,
        taskId: `task-${index}`,
      },
    }))

    saveCreationHistory(storage, key, items)
    const restored = loadCreationHistory(storage, key)

    expect(restored).toHaveLength(20)
    expect(restored[0].id).toBe('task-24')
    expect(restored.at(-1)?.id).toBe('task-5')
  })

  it('formats countdown seconds for the workspace', () => {
    expect(formatCreationCountdown(65)).toBe('01:05')
    expect(formatCreationCountdown(0)).toBe('00:00')
  })

  it('detects when an async media task exceeds its estimate', () => {
    expect(getCreationTimedOut(1_000, 90, 91_001)).toBe(true)
    expect(getCreationTimedOut(1_000, 90, 45_000)).toBe(false)
  })
})
