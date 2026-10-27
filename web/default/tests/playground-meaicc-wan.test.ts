import { describe, expect, it } from 'bun:test'
import { buildPlaygroundMediaRequest } from '../src/features/playground/lib/media-routing'
import type {
  Message,
  PlaygroundModelInfo,
} from '../src/features/playground/types'
import { getModelEndpointPreset } from '../src/features/pricing/lib/model-endpoint-presets'

const model = 'Wan 3.0'
const info: PlaygroundModelInfo = {
  model: {
    id: 1,
    model_name: model,
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    mapped_endpoints: [
      {
        type: 'openai-video',
        kind: 'video',
        path: '/v1/videos',
        method: 'POST',
        parameters: {
          durations: Array.from({ length: 26 }, (_, index) => index + 5),
          resolutions: ['720p', '1080p'],
          ratios: ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9'],
          duration_field: 'duration',
          ratio_field: 'ratio',
          resolution_field: 'resolution',
          image_field: 'referenceImages',
          video_field: 'referenceVideos',
          audio_field: 'referenceAudios',
          max_images: 10,
          max_videos: 5,
          max_audios: 5,
          allow_frames: true,
        },
      },
    ],
  },
  endpoints: {},
}
const message: Message = {
  key: 'prompt',
  from: 'user',
  versions: [{ id: 'prompt', content: 'A quiet sunrise over the sea' }],
}
const selection = { duration: '30', aspectRatio: '21:9', resolution: '1080p' }

describe('Meaicc Wan 3.0 public video contract', () => {
  it('classifies the public alias and exact provider ID as video tasks', () => {
    for (const name of [model, 'w3-c1']) {
      expect(getModelEndpointPreset(name)?.path).toBe('/v1/videos')
    }
    expect(getModelEndpointPreset('w3-c2')).toBeNull()
  })

  it('keeps the public model ID and sends all three reference types', () => {
    const request = buildPlaygroundMediaRequest(
      model,
      [message],
      'auto',
      'auto',
      {},
      selection,
      info,
      {
        images: Array(10).fill('https://cdn.example/image.png').join('\n'),
        videos: Array(5).fill('https://cdn.example/video.mp4').join('\n'),
        audios: Array(5).fill('https://cdn.example/audio.mp3').join('\n'),
      }
    )
    expect(request).toMatchObject({
      model,
      duration: 30,
      ratio: '21:9',
      resolution: '1080p',
    })
    expect(request.referenceImages).toHaveLength(10)
    expect(request.referenceVideos).toHaveLength(5)
    expect(request.referenceAudios).toHaveLength(5)
  })

  it('blocks references exceeding the published limits before submission', () => {
    for (const extra of [
      { referenceImages: Array(11).fill('https://cdn.example/image.png') },
      { referenceVideos: Array(6).fill('https://cdn.example/video.mp4') },
      { referenceAudios: Array(6).fill('https://cdn.example/audio.mp3') },
    ]) {
      expect(() =>
        buildPlaygroundMediaRequest(
          model,
          [message],
          'auto',
          'auto',
          extra,
          selection,
          info
        )
      ).toThrow()
    }
  })

  it('keeps selected capability values authoritative over extra JSON', () => {
    const request = buildPlaygroundMediaRequest(
      model,
      [message],
      'auto',
      'auto',
      { duration: 31, resolution: '480p' },
      selection,
      info
    )
    expect(request).toMatchObject({ duration: 30, resolution: '1080p' })
  })
})
