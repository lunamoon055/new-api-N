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
        model: createVideoModel('future-video-model'),
        prompt: '生成一个五秒的 1080p 视频',
        videoOptions: { resolution: '1080p', duration: '5' },
      }),
      {
        endpoint: '/pg/chat/completions',
        payload: {
          model: 'future-video-model',
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

  test('routes every videos-4 catalog model through videos api billing', () => {
    for (const model of [
      'videos-4 (4图3视频1音频)',
      'videos-4-fast (4图3视频1音频)',
      'videos-4-mini (4图3视频1音频)',
    ]) {
      assert.deepEqual(
        buildCreationVideoSubmitRequest({
          model: createVideoModel(model),
          prompt: '生成一段城市延时摄影',
          videoOptions: {
            resolution: '480p',
            duration: '6',
            aspectRatio: '16:9',
          },
        }),
        {
          endpoint: '/api/creation/video/async-generations',
          payload: {
            model,
            prompt: '生成一段城市延时摄影',
            duration: 6,
            ratio: '16:9',
            resolution: '480p',
          },
          transport: 'async-video',
        }
      )
    }
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

  test('routes the prefixed Seedance mapping with nested-reference fields', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('(线路3)sd-2.0-933'),
        prompt: '让角色从起点走到终点',
        videoOptions: {
          resolution: '480p',
          duration: '10',
          aspectRatio: '4:3',
        },
        videoReferences: {
          referenceMode: 'frames',
          imageUrls: [],
          startImageUrl: 'https://example.com/start.png',
          endImageUrl: 'https://example.com/end.png',
          videoUrls: [],
          audioUrls: [],
          audioUrl: '',
        },
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: '(线路3)sd-2.0-933',
          prompt: '让角色从起点走到终点',
          duration: 10,
          ratio: '4:3',
          resolution: '720p',
          start_image_url: 'https://example.com/start.png',
          end_image_url: 'https://example.com/end.png',
        },
        transport: 'async-video',
      }
    )
  })

  test('routes Seedance 2.5 through the documented flat videos interface', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('Seedance-2.5'),
        prompt: '让角色参考图片自然运动',
        videoOptions: {
          resolution: '480p',
          duration: '29',
          aspectRatio: '16:9',
        },
        videoReferences: {
          referenceMode: 'image',
          imageUrls: [
            { url: 'https://example.com/reference-01.png' },
            { url: 'https://example.com/reference-02.png' },
          ],
          startImageUrl: '',
          endImageUrl: '',
          videoUrls: [],
          audioUrls: [],
          audioUrl: '',
        },
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: 'Seedance-2.5',
          prompt: '让角色参考图片自然运动',
          duration: 29,
          ratio: '16:9',
          resolution: '480p',
          referenceImages: [
            'https://example.com/reference-01.png',
            'https://example.com/reference-02.png',
          ],
        },
        transport: 'async-video',
      }
    )
  })

  test('submits the 993 fixed-resolution models without an unsupported resolution field', () => {
    for (const model of ['sd2-1080P(993按秒)', 'sd2-4k(993按秒)']) {
      assert.deepEqual(
        buildCreationVideoSubmitRequest({
          model: createVideoModel(model),
          prompt: '生成一段城市延时摄影',
          videoOptions: {
            resolution: '480p',
            duration: '8',
            aspectRatio: '16:9',
          },
        }),
        {
          endpoint: '/api/creation/video/async-generations',
          payload: {
            model,
            prompt: '生成一段城市延时摄影',
            duration: 8,
            ratio: '16:9',
          },
          transport: 'async-video',
        }
      )
    }
  })

  test('submits the 993 720p model with the selected resolution', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('sd2-720P(993)'),
        prompt: '生成一段城市延时摄影',
        videoOptions: {
          resolution: '480p',
          duration: '6',
          aspectRatio: '9:16',
        },
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: 'sd2-720P(993)',
          prompt: '生成一段城市延时摄影',
          duration: 6,
          ratio: '9:16',
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

  test('routes JR Video 2.5 models with documented multi-reference fields', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('video-2.5'),
        prompt: '结合参考素材生成广告视频',
        videoOptions: {
          resolution: '720p',
          duration: '30',
          aspectRatio: '16:9',
        },
        videoReferences: {
          referenceMode: 'multimodal',
          imageUrls: [{ url: 'https://example.com/image.png' }],
          startImageUrl: '',
          endImageUrl: '',
          videoUrls: [
            { url: 'https://example.com/video-1.mp4' },
            { url: 'https://example.com/video-2.mp4' },
          ],
          audioUrls: [
            { url: 'https://example.com/audio-1.mp3' },
            { url: 'https://example.com/audio-2.wav' },
          ],
          audioUrl: '',
        },
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: 'video-2.5',
          prompt: '结合参考素材生成广告视频',
          duration: 30,
          aspect_ratio: '16:9',
          resolution: '720p',
          async: true,
          image_url: 'https://example.com/image.png',
          video_reference: [
            { url: 'https://example.com/video-1.mp4' },
            { url: 'https://example.com/video-2.mp4' },
          ],
          audio_reference: [
            { url: 'https://example.com/audio-1.mp3' },
            { url: 'https://example.com/audio-2.wav' },
          ],
        },
        transport: 'async-video',
      }
    )
  })

  test('submits MiniMax H3 with documented image and audio fields', () => {
    assert.deepEqual(
      buildCreationVideoSubmitRequest({
        model: createVideoModel('minimax-h3'),
        prompt: '让图中角色跟随音乐跳舞',
        videoOptions: {
          resolution: '480p',
          duration: '8',
          aspectRatio: '4:3',
        },
        videoReferences: {
          referenceMode: 'image-audio',
          imageUrls: [
            { url: 'https://example.com/character.png' },
            { url: 'https://example.com/scene.png' },
          ],
          startImageUrl: '',
          endImageUrl: '',
          videoUrls: [],
          audioUrls: [{ url: 'https://example.com/music.ogg' }],
          audioUrl: { url: 'https://example.com/music.ogg' },
        },
      }),
      {
        endpoint: '/api/creation/video/async-generations',
        payload: {
          model: 'minimax-h3',
          prompt: '让图中角色跟随音乐跳舞',
          duration: 8,
          aspect_ratio: '4:3',
          image_urls: [
            'https://example.com/character.png',
            'https://example.com/scene.png',
          ],
          audio_url: 'https://example.com/music.ogg',
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
