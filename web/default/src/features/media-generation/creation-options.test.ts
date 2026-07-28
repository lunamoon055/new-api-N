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
import {
  getCreationImageAspectRatioOptions,
  getCreationImageReferenceLimits,
  getCreationImageRequestOptions,
  normalizeCreationImageOptions,
} from './image-options'
import {
  getCreationDurationOptions,
  getCreationResolutionOptions,
  getCreationVideoCapabilities,
  getCreationVideoReferenceLimits,
  getCreationVideoRequestOptions,
  normalizeCreationVideoOptions,
} from './video-options'

const sanbaoImageModel = {
  id: 'sanbao-image',
  metadata: {
    provider: 'sanbao',
    type: 'image',
    aspect_ratios: ['1:1', '16:9', '9:16'],
    max_images: 3,
    max_image_size_mb: 12,
  },
}

const sanbaoVideoModel = {
  id: 'sanbao-video',
  metadata: {
    provider: 'sanbao',
    type: 'video',
    resolutions: ['480p', '720p'],
    ratios: ['9:16', '16:9'],
    durations: [5, 10],
    max_images: 2,
    max_videos: 1,
    max_audios: 1,
    max_media_files: 3,
    max_image_size_mb: 8,
    max_video_size_mb: 80,
    max_audio_size_mb: 10,
    concurrency_options: [1, 2],
  },
}

describe('Sanbao creation model options', () => {
  test('uses Sanbao image metadata for aspect ratios, limits, and request fields', () => {
    assert.deepEqual(getCreationImageAspectRatioOptions(sanbaoImageModel), [
      '1:1',
      '16:9',
      '9:16',
    ])
    assert.deepEqual(
      normalizeCreationImageOptions({ aspectRatio: '16:9' }, sanbaoImageModel),
      {
        aspectRatio: '16:9',
      }
    )
    assert.equal(getCreationImageReferenceLimits(sanbaoImageModel).maxImages, 3)

    const request = getCreationImageRequestOptions(
      'poster',
      sanbaoImageModel,
      { imageUrls: [{ url: 'https://example.com/ref.png' }] },
      { aspectRatio: '16:9' }
    )

    assert.deepEqual(request, {
      aspect_ratio: '16:9',
      images: ['https://example.com/ref.png'],
      quality: 'high',
      concurrency: 1,
    })
  })

  test('uses Sanbao video metadata for controls, limits, and request fields', () => {
    assert.deepEqual(
      getCreationResolutionOptions(sanbaoVideoModel).map((item) => item.value),
      ['480p', '720p']
    )
    assert.deepEqual(
      getCreationDurationOptions(sanbaoVideoModel).map((item) => item.value),
      ['5', '10']
    )
    assert.deepEqual(
      getCreationVideoCapabilities(sanbaoVideoModel)?.aspectRatios,
      ['9:16', '16:9']
    )
    assert.equal(getCreationVideoReferenceLimits(sanbaoVideoModel).maxVideos, 1)

    const options = normalizeCreationVideoOptions(
      { resolution: '720p', duration: '10', aspectRatio: '16:9' },
      sanbaoVideoModel
    )
    const request = getCreationVideoRequestOptions(options, sanbaoVideoModel, {
      referenceMode: 'multimodal',
      imageUrls: [{ url: 'https://example.com/ref.png' }],
      startImageUrl: '',
      endImageUrl: '',
      videoUrls: [{ url: 'https://example.com/ref.mp4' }],
      audioUrls: [{ url: 'https://example.com/ref.wav' }],
      audioUrl: { url: 'https://example.com/ref.wav' },
    })

    assert.deepEqual(request, {
      duration: 10,
      ratio: '16:9',
      resolution: '720p',
      concurrency: 1,
      estimateSeconds: 210,
      images: ['https://example.com/ref.png'],
      videos: ['https://example.com/ref.mp4'],
      audios: ['https://example.com/ref.wav'],
      reference: 'all',
    })
  })

  test('uses videos api aliases for controls, limits, and request fields', () => {
    assert.deepEqual(
      getCreationResolutionOptions('sd2满血').map((item) => item.value),
      ['720p', '480p']
    )
    assert.deepEqual(
      getCreationDurationOptions('sd2满血').slice(0, 3).map((item) => item.value),
      ['4', '5', '6']
    )
    assert.equal(getCreationVideoReferenceLimits('sd2满血').maxAudios, 3)
    assert.equal(getCreationVideoCapabilities('sd2满血')?.kind, 'videos')

    const options = normalizeCreationVideoOptions(
      { resolution: '480p', duration: '8', aspectRatio: '1:1' },
      'sd2满血'
    )
    const request = getCreationVideoRequestOptions(options, 'sd2满血', {
      referenceMode: 'multimodal',
      imageUrls: [{ url: 'https://example.com/ref.png' }],
      startImageUrl: '',
      endImageUrl: '',
      videoUrls: [{ url: 'https://example.com/ref.mp4' }],
      audioUrls: [{ url: 'https://example.com/ref.wav' }],
      audioUrl: { url: 'https://example.com/ref.wav' },
    })

    assert.deepEqual(request, {
      duration: 8,
      ratio: '1:1',
      resolution: '480p',
      estimateSeconds: 180,
      referenceImages: ['https://example.com/ref.png'],
      referenceVideos: ['https://example.com/ref.mp4'],
      referenceAudios: ['https://example.com/ref.wav'],
    })
  })
})
