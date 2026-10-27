import { expect, test } from 'bun:test'
import { buildVideoDocumentationBody } from '../src/features/pricing/lib/video-documentation'
import { getModelEndpointPreset } from '../src/features/pricing/lib/model-endpoint-presets'

test('Suanliai 005 examples keep public model ID and valid video options', () => {
  expect(
    getModelEndpointPreset('005系列/sd2.5(30-10-10/5-30秒/720P)')?.path
  ).toBe('/v1/videos')
  expect(getModelEndpointPreset('005系列/other')).toBeNull()
  const body = buildVideoDocumentationBody(
    'Seedance2.5:720-S（条）',
    'demo',
    {
      duration_field: 'duration',
      durations: [5, 10, 30],
      ratio_field: 'ratio',
      ratios: ['16:9', '21:9'],
      resolution_field: 'resolution',
      resolutions: ['720p'],
      image_field: 'referenceImages',
      min_images: 1,
      max_images: 30,
      max_videos: 10,
      max_audios: 10,
    },
    { duration: 4, resolution: '1080p' }
  )
  expect(body).toEqual({
    model: 'Seedance2.5:720-S（条）',
    prompt: 'demo',
    duration: 5,
    ratio: '16:9',
    resolution: '720p',
    referenceImages: ['https://cdn.example.com/reference.jpg'],
  })
})

test('video examples retain ratio even when capabilities are unpublished', () => {
  expect(buildVideoDocumentationBody('public-model', 'demo')).toEqual({
    model: 'public-model',
    prompt: 'demo',
    ratio: '16:9',
  })
})
