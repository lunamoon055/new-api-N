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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { CreationModel } from './types'
import {
  buildCreationVideoSubmitRequest,
  extractPromptVideoURL,
} from './video-request'

function createVideoModel(
  id: string,
  supportedEndpointTypes = ['openai']
): CreationModel {
  return {
    id,
    supported_endpoint_types: supportedEndpointTypes,
  }
}

describe('Creation Center video request payload', () => {
  test('routes unadapted OpenAI models through the prompt endpoint', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('videos-4 (4图3视频1音频)'),
        prompt: '生成一个五秒的 1080p 视频',
        videoOptions: { resolution: '1080p', duration: '5' },
      }),
      {
        endpoint: '/pg/chat/completions',
        payload: {
          model: 'videos-4 (4图3视频1音频)',
          messages: [
            {
              role: 'user',
              content: '生成一个五秒的 1080p 视频',
            },
          ],
          stream: false,
        },
        transport: 'prompt',
      }
    )
  })

  test('pairs catalog-suffixed sd2 models with the videos api UI and endpoint', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('sd2-mini (9图3视频3音频)'),
        prompt: '生成一段城市延时摄影',
        videoOptions: {
          resolution: '480p',
          duration: '8',
          aspectRatio: '1:1',
        },
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: 'sd2-mini (9图3视频3音频)',
          prompt: '生成一段城市延时摄影',
          duration: 8,
          ratio: '1:1',
          resolution: '480p',
        },
        transport: 'async-video',
      }
    )
  })

  test('keeps dedicated request fields for adapted video models', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('video-2.0-fast'),
        prompt: '生成一段城市延时摄影',
        videoOptions: {
          resolution: '720p',
          duration: '5',
          aspectRatio: '9:16',
        },
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: 'video-2.0-fast',
          prompt: '生成一段城市延时摄影',
          duration: 5,
          aspect_ratio: '9:16',
          resolution: '720p',
          async: true,
        },
        transport: 'async-video',
      }
    )
  })

  test('keeps prompt-only payloads on explicit video endpoints', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('future-video-model', ['openai-video']),
        prompt: '生成视频',
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: 'future-video-model',
          prompt: '生成视频',
        },
        transport: 'async-video',
      }
    )
  })

  test('extracts video URLs from JSON and Markdown prompt responses', () => {
    assert.equal(
      extractPromptVideoURL('{"data":{"video_url":"https://cdn.test/a.mp4"}}'),
      'https://cdn.test/a.mp4'
    )
    assert.equal(
      extractPromptVideoURL('生成完成：[下载视频](https://cdn.test/b.mp4)'),
      'https://cdn.test/b.mp4'
    )
  })
})
