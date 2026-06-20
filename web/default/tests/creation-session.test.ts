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
import { formatCreationModelCost } from '../src/features/creation-center/cost'
import { formatQuota } from '../src/lib/format'
import {
  DEFAULT_CREATION_VIDEO_OPTIONS,
  EMPTY_CREATION_VIDEO_REFERENCES,
  formatCreationCountdown,
  getCreationDurationOptions,
  getCreationResolutionOptions,
  getCreationTimedOut,
  getCreationVideoCapabilities,
  getCreationVideoOptionsError,
  getCreationVideoReferenceError,
  getCreationVideoRequestOptions,
  loadCreationHistory,
  normalizeCreationVideoReferences,
  sanitizeCreationHistoryItem,
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
  it('shows only the final quota for fixed-price video model costs', () => {
    const t = (key: string) => key

    const label = formatCreationModelCost(
      {
        billing_mode: 'per_request',
        request_price: 0.0000072,
        request_quota: 3.6,
        group_ratio: 2,
      },
      t,
      'video'
    )

    expect(label).toBe(formatQuota(3.6))
    expect(label).not.toContain('per request')
    expect(label).not.toContain('·')
  })

  it('keeps detailed fixed-price labels for non-video model costs', () => {
    const t = (key: string) => key

    const label = formatCreationModelCost(
      {
        billing_mode: 'per_request',
        request_price: 0.0000072,
        request_quota: 3.6,
      },
      t,
      'image'
    )

    expect(label).toContain('per request')
    expect(label).toContain('·')
  })

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
      getCreationDurationOptions('sora-2').map((item) => item.value)
    ).toEqual(['4', '8', '12'])
    expect(
      getCreationDurationOptions('sora-2-pro').map((item) => item.value)
    ).toEqual(['4', '8', '12'])
    expect(getCreationResolutionOptions('sora2')).toEqual([
      {
        value: '720p',
        label: '720p',
        size: '720x1280',
        estimateMultiplier: 1,
      },
    ])
    expect(getCreationVideoCapabilities('sora2')).toMatchObject({
      kind: 'sora2',
      aspectRatios: ['9:16', '16:9'],
      showResolution: false,
      durationControl: 'select',
    })
    expect(
      getCreationVideoRequestOptions(
        { resolution: '1080p', duration: '8', aspectRatio: '16:9' },
        'sora2',
        {
          ...EMPTY_CREATION_VIDEO_REFERENCES,
          imageUrls: ['data:image/png;base64,AAAA'],
        }
      )
    ).toMatchObject({
      seconds: '8',
      size: '1280x720',
      aspect_ratio: '16:9',
      input_reference: 'data:image/png;base64,AAAA',
    })
    expect(
      getCreationVideoRequestOptions(
        { resolution: '1080p', duration: '8', aspectRatio: '9:16' },
        'sora-2-pro'
      )
    ).toMatchObject({
      seconds: '8',
      size: '720x1280',
      aspect_ratio: '9:16',
    })
    expect(
      getCreationVideoRequestOptions(
        { resolution: '1080p', duration: '5' },
        'sora-2'
      ).seconds
    ).toBe('4')
    expect(
      getCreationVideoReferenceError('sora2', {
        ...EMPTY_CREATION_VIDEO_REFERENCES,
        imageUrls: [
          'https://cdn.example/a.png',
          'https://cdn.example/b.png',
        ],
      })
    ).toBe('Sora2 accepts at most 1 reference image.')
  })

  it('uses documented Video2 capabilities and payload fields', () => {
    expect(
      getCreationDurationOptions('video-2.0').map((item) => item.value)
    ).toEqual(Array.from({ length: 12 }, (_, index) => String(index + 4)))
    expect(getCreationResolutionOptions('video-2.0')).toEqual([
      {
        value: '720p',
        label: '720p',
        size: '720x1280',
        estimateMultiplier: 1,
      },
    ])
    expect(getCreationVideoCapabilities('video-2.0')?.aspectRatios).toEqual([
      '9:16',
      '16:9',
      '1:1',
    ])
    expect(
      getCreationVideoRequestOptions(
        { resolution: '720p', duration: '15', aspectRatio: '1:1' },
        'video-2.0'
      )
    ).toMatchObject({
      duration: 15,
      aspect_ratio: '1:1',
      resolution: '720p',
      async: true,
    })
  })

  it('normalizes references per Video2 model', () => {
    const references = {
      imageUrls: ['https://cdn.example/a.png'],
      startImageUrl: 'https://cdn.example/start.png',
      endImageUrl: 'https://cdn.example/end.png',
      videoUrls: ['https://cdn.example/a.mp4'],
      audioUrl: 'https://cdn.example/a.mp3',
    }

    expect(normalizeCreationVideoReferences(references, 'video-2.0')).toEqual({
      ...EMPTY_CREATION_VIDEO_REFERENCES,
      imageUrls: ['https://cdn.example/a.png'],
    })
    expect(
      normalizeCreationVideoReferences(references, 'video-2.0-fast')
    ).toEqual(references)
  })

  it('validates Video2 option bounds and remote references', () => {
    expect(
      getCreationVideoOptionsError(
        { resolution: '720p', duration: '16', aspectRatio: '9:16' },
        'video-2.0'
      )
    ).toBe('Duration must be between 4 and 15 seconds.')
    expect(
      getCreationVideoReferenceError('video-2.0-fast', {
        ...EMPTY_CREATION_VIDEO_REFERENCES,
        audioUrl: 'file:///tmp/a.mp3',
      })
    ).toBe('Reference URL must use HTTP or HTTPS.')
    expect(
      getCreationVideoReferenceError('video-2.0', {
        ...EMPTY_CREATION_VIDEO_REFERENCES,
        imageUrls: ['data:image/png;base64,AAAA'],
      })
    ).toBe('Reference image URL must use HTTP or HTTPS.')
    expect(
      getCreationVideoReferenceError('video-2.0', {
        ...EMPTY_CREATION_VIDEO_REFERENCES,
        imageUrls: ['data:text/plain;base64,AAAA'],
      })
    ).toBe('Reference images must be images or HTTP URLs.')
    expect(
      getCreationVideoReferenceError('video-2.0', {
        ...EMPTY_CREATION_VIDEO_REFERENCES,
        imageUrls: Array.from(
          { length: 5 },
          (_, index) => `https://cdn.example/${index}.png`
        ),
      })
    ).toBe('Video2 accepts at most 4 image references.')
  })

  it('maps Video2 Fast references to documented singular and array fields', () => {
    expect(
      getCreationVideoRequestOptions(
        { resolution: '720p', duration: '8', aspectRatio: '16:9' },
        'video-2.0-fast',
        {
          imageUrls: [
            'https://cdn.example/a.png',
            'https://cdn.example/b.png',
          ],
          startImageUrl: 'https://cdn.example/start.png',
          endImageUrl: 'https://cdn.example/end.png',
          videoUrls: [
            'https://cdn.example/a.mp4',
            'https://cdn.example/b.mp4',
          ],
          audioUrl: 'https://cdn.example/a.mp3',
        }
      )
    ).toMatchObject({
      duration: 8,
      aspect_ratio: '16:9',
      resolution: '720p',
      async: true,
      image_urls: [
        'https://cdn.example/a.png',
        'https://cdn.example/b.png',
      ],
      start_image_url: 'https://cdn.example/start.png',
      end_image_url: 'https://cdn.example/end.png',
      video_reference: [
        { url: 'https://cdn.example/a.mp4' },
        { url: 'https://cdn.example/b.mp4' },
      ],
      audio_url: 'https://cdn.example/a.mp3',
    })
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

  it('keeps local reference image data urls out of persisted history', () => {
    const item = sanitizeCreationHistoryItem({
      createdAt: 1,
      id: 'task-1',
      mode: 'video',
      model: 'video-2.0',
      prompt: 'prompt',
      result: {
        mode: 'video',
        model: 'video-2.0',
        status: 'queued',
        taskId: 'task-1',
      },
      videoReferences: {
        imageUrls: [
          'data:image/png;base64,AAAA',
          'https://cdn.example/keep.png',
        ],
        startImageUrl: 'data:image/png;base64,BBBB',
        endImageUrl: 'https://cdn.example/end.png',
        videoUrls: [],
        audioUrl: '',
      },
    })

    expect(item.videoReferences).toMatchObject({
      imageUrls: ['https://cdn.example/keep.png'],
      startImageUrl: '',
      endImageUrl: 'https://cdn.example/end.png',
    })
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
