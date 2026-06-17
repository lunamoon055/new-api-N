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
  buildPlaygroundMediaRequest,
  parsePlaygroundMediaResult,
  getPlaygroundModelMode,
  getPlaygroundMediaEndpoint,
} from '../src/features/playground/lib/media-routing'
import type { Message } from '../src/features/playground/types'

const userMessage = (content: string): Message => ({
  key: `user-${content}`,
  from: 'user',
  versions: [{ id: `version-${content}`, content }],
})

const assistantMessage = (content: string): Message => ({
  key: `assistant-${content}`,
  from: 'assistant',
  versions: [{ id: `version-${content}`, content }],
})

describe('playground media routing', () => {
  it('routes linksky video models away from chat completions', () => {
    for (const model of [
      'video-2.0',
      'video-2.0-fast',
      'sora2',
      'ko3',
      'veo31',
      'veo31-fast',
      'veo31-ref',
      'grok-imagine-video',
    ]) {
      expect(getPlaygroundModelMode(model)).toBe('video')
      expect(getPlaygroundMediaEndpoint(model)).toBe(
        '/api/creation/video/async-generations'
      )
    }
  })

  it('routes image models to image generations', () => {
    for (const model of ['gpt-image2', 'nano-banana', 'nano-banana-pro']) {
      expect(getPlaygroundModelMode(model)).toBe('image')
      expect(getPlaygroundMediaEndpoint(model)).toBe(
        '/api/creation/images/generations'
      )
    }
  })

  it('keeps ordinary chat models on chat completions', () => {
    expect(getPlaygroundModelMode('gpt-5.4')).toBe('chat')
    expect(getPlaygroundMediaEndpoint('gpt-5.4')).toBeNull()
  })

  it('builds media payloads from the latest user prompt', () => {
    const messages = [
      userMessage('old prompt'),
      assistantMessage('old answer'),
      userMessage('  make a short API website video  '),
      assistantMessage(''),
    ]

    expect(buildPlaygroundMediaRequest('video-2.0', messages)).toEqual({
      model: 'video-2.0',
      prompt: 'make a short API website video',
      seconds: '5',
      size: '1920x1080',
    })

    expect(buildPlaygroundMediaRequest('sora2', messages)).toEqual({
      model: 'sora2',
      prompt: 'make a short API website video',
      seconds: '4',
      size: '1920x1080',
    })

    expect(buildPlaygroundMediaRequest('gpt-image2', messages)).toEqual({
      model: 'gpt-image2',
      prompt: 'make a short API website video',
      n: 1,
    })
  })

  it('uses direct completed video urls when async task responses include one', () => {
    const result = parsePlaygroundMediaResult(
      {
        data: {
          task_id: 'task_video_123',
          status: 'completed',
          result_url: 'https://example.com/video.mp4',
        },
      },
      'video-2.0'
    )

    expect(result).toMatchObject({
      mode: 'video',
      taskId: 'task_video_123',
      status: 'completed',
      mediaUrl: 'https://example.com/video.mp4',
    })
    expect(result.content).toContain('视频生成完成')
  })

  it('falls back to the authenticated video proxy when completed tasks have no direct url', () => {
    const result = parsePlaygroundMediaResult(
      {
        data: {
          task_id: 'task_video_123',
          status: 'completed',
        },
      },
      'video-2.0'
    )

    expect(result).toMatchObject({
      mode: 'video',
      taskId: 'task_video_123',
      status: 'completed',
      mediaUrl: '/v1/videos/task_video_123/content',
    })
  })

  it('uses the authenticated video proxy for upstream api content urls', () => {
    const result = parsePlaygroundMediaResult(
      {
        data: {
          task_id: 'task_video_123',
          status: 'completed',
          result_url:
            'https://api.example.com/v1/video/async-generations/upstream_123/content',
        },
      },
      'video-2.0'
    )

    expect(result.mediaUrl).toBe('/v1/videos/task_video_123/content')
  })
})
