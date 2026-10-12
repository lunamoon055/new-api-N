import { describe, expect, it } from 'bun:test'
import { buildPlaygroundMediaRequest } from '../src/features/playground/lib/media-routing'
import type {
  Message,
  PlaygroundModelInfo,
} from '../src/features/playground/types'

const model = 'Seedance2.5-c'
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
          durations: [6],
          resolutions: ['720p'],
          ratios: ['16:9'],
          duration_field: 'duration',
          ratio_field: 'ratio',
          resolution_field: 'resolution',
          image_field: 'referenceImages',
          video_field: 'referenceVideos',
          audio_field: 'referenceAudios',
          max_images: 10,
          max_videos: 0,
          max_audios: 0,
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
const selection = { duration: '6', aspectRatio: '16:9', resolution: '720p' }
const images = Array(10).fill('https://cdn.example/ref.png') as string[]

describe('Seedance2.5-c reference contract', () => {
  it('sends ten images with the public model ID', () => {
    const request = buildPlaygroundMediaRequest(
      model,
      [message],
      'auto',
      'auto',
      {},
      selection,
      info,
      { images: images.join('\n') }
    )
    expect(request).toMatchObject({ model, referenceImages: images })
    expect(request).not.toHaveProperty('referenceVideos')
    expect(request).not.toHaveProperty('referenceAudios')
  })

  it('rejects eleven images, video and audio before submission', () => {
    for (const references of [
      { images: [...images, 'https://cdn.example/extra.png'].join('\n') },
      { videos: 'https://cdn.example/ref.mp4' },
      { audios: 'https://cdn.example/ref.mp3' },
    ]) {
      expect(() =>
        buildPlaygroundMediaRequest(
          model,
          [message],
          'auto',
          'auto',
          {},
          selection,
          info,
          references
        )
      ).toThrow()
    }
  })

  it('applies the same limits to extra JSON parameters', () => {
    for (const extra of [
      { referenceImages: [...images, 'https://cdn.example/extra.png'] },
      { referenceVideos: ['https://cdn.example/ref.mp4'] },
      { referenceAudios: ['https://cdn.example/ref.mp3'] },
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
})
