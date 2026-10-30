import { expect, test } from 'bun:test'
import {
  buildVideoDocumentationBody,
  hasVideoReferenceMedia,
  isVideoCapabilityUnknown,
} from '../src/features/pricing/lib/video-documentation'
import { getModelEndpointPreset } from '../src/features/pricing/lib/model-endpoint-presets'

test('partially published capabilities retain confirmed references without inventing missing limits', () => {
  const p = {image_field:'referenceImages',video_field:'referenceVideos',audio_field:'referenceAudios',max_images:3,max_videos:10,max_audios:0,unknown_fields:['max_videos']}
  expect(isVideoCapabilityUnknown(p,'max_images')).toBe(false)
  expect(isVideoCapabilityUnknown(p,'max_videos')).toBe(true)
  expect(hasVideoReferenceMedia(p)).toBe(true)
  const body = buildVideoDocumentationBody('public-model','demo',p,{includeReferenceMedia:true})
  expect(body.referenceImages).toEqual(['https://cdn.example.com/reference.jpg'])
  expect(body.referenceVideos).toBeUndefined()
  expect(body.referenceAudios).toBeUndefined()
})

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

test('reference examples include all supported media and satisfy minimum image counts', () => {
  const p = {
    duration_field: 'duration',
    durations: [5, 30],
    resolution_field: 'resolution',
    resolutions: ['720p'],
    image_field: 'referenceImages',
    video_field: 'referenceVideos',
    audio_field: 'referenceAudios',
    min_images: 2,
    max_images: 30,
    max_videos: 10,
    max_audios: 10,
    audio_requires_image: true,
  }
  const before = structuredClone(p)
  const body = buildVideoDocumentationBody('public-model', 'demo', p, {
    includeReferenceMedia: true,
  })
  expect(body.referenceImages).toEqual([
    'https://cdn.example.com/reference.jpg',
    'https://cdn.example.com/reference-2.jpg',
  ])
  expect(body.referenceVideos).toEqual([
    'https://cdn.example.com/reference.mp4',
  ])
  expect(body.referenceAudios).toEqual([
    'https://cdn.example.com/reference.mp3',
  ])
  expect(body.model).toBe('public-model')
  expect(p).toEqual(before)
  expect(hasVideoReferenceMedia(p)).toBe(true)
})

test('basic examples remain minimal and reference examples exclude unsupported and unknown capabilities', () => {
  const p = {
    image_field: 'referenceImages',
    video_field: 'referenceVideos',
    audio_field: 'referenceAudios',
    max_images: 30,
    max_videos: 0,
    max_audios: 10,
  }
  const minimal = buildVideoDocumentationBody('public-model', 'demo', p)
  expect(minimal).not.toHaveProperty('referenceImages')
  expect(minimal).not.toHaveProperty('referenceAudios')
  const complete = buildVideoDocumentationBody('public-model', 'demo', p, {
    includeReferenceMedia: true,
  })
  expect(complete).toHaveProperty('referenceImages')
  expect(complete).toHaveProperty('referenceAudios')
  expect(complete).not.toHaveProperty('referenceVideos')
  const unknown = { ...p, capabilities_unknown: true }
  expect(hasVideoReferenceMedia(unknown)).toBe(false)
  const body = buildVideoDocumentationBody('public-model', 'demo', unknown, {
    includeReferenceMedia: true,
  })
  expect(body).not.toHaveProperty('referenceImages')
  expect(body).not.toHaveProperty('referenceVideos')
  expect(body).not.toHaveProperty('referenceAudios')
})
