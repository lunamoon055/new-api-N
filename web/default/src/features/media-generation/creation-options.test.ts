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
  getCreationPromptMaxLength,
  getCreationResolutionOptions,
  getCreationVideoCapabilities,
  getCreationVideoOptionsError,
  getCreationVideoReferenceError,
  getCreationVideoReferenceLimits,
  getCreationVideoRequestOptions,
  normalizeCreationVideoOptions,
  normalizeCreationVideoReferences,
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
      getCreationDurationOptions('sd2满血')
        .slice(0, 3)
        .map((item) => item.value),
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

  test('supports the prefixed Seedance 2.0 mapping and documented references', () => {
    const model = '(线路3)sd-2.0-933'
    assert.equal(getCreationVideoCapabilities(model)?.kind, 'videos')
    assert.deepEqual(
      getCreationResolutionOptions(model).map((item) => item.value),
      ['720p']
    )
    assert.deepEqual(getCreationVideoCapabilities(model)?.aspectRatios, [
      '16:9',
      '9:16',
      '1:1',
      '4:3',
      '3:4',
    ])

    const options = normalizeCreationVideoOptions(
      { resolution: '480p', duration: '10', aspectRatio: '4:3' },
      model
    )
    assert.deepEqual(
      getCreationVideoRequestOptions(options, model, {
        referenceMode: 'frames',
        imageUrls: [],
        startImageUrl: 'https://example.com/start.png',
        endImageUrl: 'https://example.com/end.png',
        videoUrls: [],
        audioUrls: [],
        audioUrl: '',
      }),
      {
        duration: 10,
        ratio: '4:3',
        resolution: '720p',
        estimateSeconds: 210,
        start_image_url: 'https://example.com/start.png',
        end_image_url: 'https://example.com/end.png',
      }
    )

    assert.deepEqual(
      getCreationVideoRequestOptions(options, model, {
        referenceMode: 'image',
        imageUrls: [
          { url: 'https://example.com/first.png' },
          { url: 'https://example.com/reference.png' },
        ],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [],
        audioUrls: [],
        audioUrl: '',
      }),
      {
        duration: 10,
        ratio: '4:3',
        resolution: '720p',
        estimateSeconds: 210,
        start_image_url: 'https://example.com/first.png',
        referenceImages: ['https://example.com/reference.png'],
      }
    )
  })

  test('pairs Seedance 2.5 with the Seedance upload controls and request fields', () => {
    const model = 'Seedance-2.5'
    const capability = getCreationVideoCapabilities(model)

    assert.equal(capability?.kind, 'videos')
    assert.deepEqual(capability?.referenceModes, [
      'text',
      'image',
      'video',
      'multimodal',
    ])
    assert.deepEqual(capability?.aspectRatios, ['16:9', '9:16', '1:1'])
    assert.deepEqual(
      getCreationResolutionOptions(model).map((item) => item.value),
      ['720p', '480p']
    )
    assert.deepEqual(
      {
        maxImages: getCreationVideoReferenceLimits(model).maxImages,
        maxVideos: getCreationVideoReferenceLimits(model).maxVideos,
        maxAudios: getCreationVideoReferenceLimits(model).maxAudios,
      },
      { maxImages: 30, maxVideos: 10, maxAudios: 10 }
    )
    assert.deepEqual(
      getCreationDurationOptions(model).map((item) => item.value),
      Array.from({ length: 26 }, (_, index) => String(index + 4))
    )
    assert.equal(
      getCreationVideoReferenceLimits(model).maxVideoTotalSizeMB,
      667
    )
    assert.equal(
      getCreationVideoReferenceLimits(model).maxAudioTotalSizeMB,
      50
    )
    assert.equal(
      getCreationVideoReferenceLimits(model)
        .maxReferenceVideoTotalDurationSeconds,
      29
    )
    assert.equal(
      getCreationVideoReferenceLimits(model)
        .maxReferenceAudioTotalDurationSeconds,
      29
    )
    assert.equal(capability?.uploadTipProfile, 'seedance-2.5')

    const options = normalizeCreationVideoOptions(
      { resolution: '480p', duration: '29', aspectRatio: '9:16' },
      model
    )
    assert.equal(
      getCreationVideoOptionsError(
        { resolution: '720p', duration: '30', aspectRatio: '9:16' },
        model
      ),
      'Seedance 2.5 duration must be between 4 and 29 seconds.'
    )
    assert.deepEqual(
      getCreationVideoRequestOptions(options, model, {
        referenceMode: 'multimodal',
        imageUrls: [{ url: 'https://example.com/ref.png' }],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [{ url: 'https://example.com/ref.mp4' }],
        audioUrls: [{ url: 'https://example.com/ref.wav' }],
        audioUrl: { url: 'https://example.com/ref.wav' },
      }),
      {
        duration: 29,
        ratio: '9:16',
        resolution: '480p',
        estimateSeconds: 495,
        referenceImages: ['https://example.com/ref.png'],
        referenceVideos: ['https://example.com/ref.mp4'],
        referenceAudios: ['https://example.com/ref.wav'],
      }
    )

    assert.equal(
      getCreationVideoReferenceError(model, {
        referenceMode: 'video',
        imageUrls: [],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [
          {
            url: 'https://example.com/ref.mp4',
            sizeBytes: 668 * 1024 * 1024,
          },
        ],
        audioUrls: [],
        audioUrl: '',
      }),
      'Seedance 2.5 reference videos must not exceed 667 MB in total.'
    )

    assert.equal(
      getCreationVideoReferenceError(model, {
        referenceMode: 'video',
        imageUrls: [],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [
          {
            url: 'https://example.com/ref-1.mp4',
            durationSeconds: 15,
          },
          {
            url: 'https://example.com/ref-2.mp4',
            durationSeconds: 15,
          },
        ],
        audioUrls: [],
        audioUrl: '',
      }),
      'Seedance 2.5 reference videos must not exceed 29 seconds in total.'
    )

    assert.equal(
      getCreationVideoReferenceError(model, {
        referenceMode: 'multimodal',
        imageUrls: [],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [],
        audioUrls: [
          {
            url: 'https://example.com/ref-1.wav',
            durationSeconds: 15,
          },
          {
            url: 'https://example.com/ref-2.wav',
            durationSeconds: 15,
          },
        ],
        audioUrl: '',
      }),
      'Seedance 2.5 reference audios must not exceed 29 seconds in total.'
    )

    assert.equal(
      getCreationVideoReferenceError(model, {
        referenceMode: 'multimodal',
        imageUrls: [],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [],
        audioUrls: Array.from({ length: 4 }, (_, index) => ({
          url: `https://example.com/ref-${index + 1}.wav`,
          sizeBytes: 13 * 1024 * 1024,
        })),
        audioUrl: '',
      }),
      'Seedance 2.5 reference audios must not exceed 50 MB in total.'
    )
  })

  test('uses videos-4 controls, limits, and videos api request fields', () => {
    for (const model of [
      'videos-4 (4图3视频1音频)',
      'videos-4-fast (4图3视频1音频)',
      'videos-4-mini (4图3视频1音频)',
    ]) {
      assert.equal(getCreationVideoCapabilities(model)?.kind, 'videos')
      assert.deepEqual(
        getCreationResolutionOptions(model).map((item) => item.value),
        ['720p', '480p']
      )
      assert.deepEqual(
        {
          maxImages: getCreationVideoReferenceLimits(model).maxImages,
          maxVideos: getCreationVideoReferenceLimits(model).maxVideos,
          maxAudios: getCreationVideoReferenceLimits(model).maxAudios,
        },
        {
          maxImages: 4,
          maxVideos: 3,
          maxAudios: 1,
        }
      )
    }

    const model = 'videos-4-mini (4图3视频1音频)'
    const options = normalizeCreationVideoOptions(
      { resolution: '480p', duration: '6', aspectRatio: '9:16' },
      model
    )
    assert.deepEqual(
      getCreationVideoRequestOptions(options, model, {
        referenceMode: 'multimodal',
        imageUrls: [{ url: 'https://example.com/ref.png' }],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [{ url: 'https://example.com/ref.mp4' }],
        audioUrls: [{ url: 'https://example.com/ref.wav' }],
        audioUrl: { url: 'https://example.com/ref.wav' },
      }),
      {
        duration: 6,
        ratio: '9:16',
        resolution: '480p',
        estimateSeconds: 150,
        referenceImages: ['https://example.com/ref.png'],
        referenceVideos: ['https://example.com/ref.mp4'],
        referenceAudios: ['https://example.com/ref.wav'],
      }
    )
  })

  test('ignores catalog display suffixes when matching videos api models', () => {
    const model = 'sd2-mini (9图3视频3音频)'

    assert.equal(getCreationVideoCapabilities(model)?.kind, 'videos')
    assert.deepEqual(
      getCreationResolutionOptions(model).map((item) => item.value),
      ['720p', '480p']
    )
    assert.deepEqual(getCreationVideoReferenceLimits(model), {
      maxImages: 9,
      maxVideos: 3,
      maxAudios: 3,
      maxMediaFiles: 15,
      maxImageSizeMB: 20,
      maxVideoSizeMB: 200,
      maxAudioSizeMB: 50,
      maxImageSizeBytes: 20 * 1024 * 1024,
      maxVideoSizeBytes: 200 * 1024 * 1024,
      maxAudioSizeBytes: 50 * 1024 * 1024,
    })

    assert.equal(
      getCreationVideoCapabilities('sd2-fast（9图3视频3音频）')?.kind,
      'videos'
    )
  })

  test('uses fixed resolutions for the 993 1080p and 4k models', () => {
    for (const [model, expectedResolution] of [
      ['sd2-1080P(993按秒)', '1080p'],
      ['sd2-4k(993按秒)', '4k'],
    ] as const) {
      const capability = getCreationVideoCapabilities(model)
      assert.equal(capability?.kind, 'videos')
      assert.equal(capability?.showResolution, false)
      assert.equal(capability?.includeResolutionInRequest, false)
      assert.deepEqual(
        getCreationResolutionOptions(model).map((item) => item.value),
        [expectedResolution]
      )

      const options = normalizeCreationVideoOptions(
        { resolution: '480p', duration: '8', aspectRatio: '16:9' },
        model
      )
      assert.equal(options.resolution, expectedResolution)
      assert.deepEqual(getCreationVideoRequestOptions(options, model), {
        duration: 8,
        ratio: '16:9',
        estimateSeconds: 180,
      })
    }
  })

  test('offers 480p and 720p for the 993 720p model', () => {
    const model = 'sd2-720P(993)'
    const capability = getCreationVideoCapabilities(model)

    assert.equal(capability?.kind, 'videos')
    assert.equal(capability?.showResolution, true)
    assert.equal(capability?.includeResolutionInRequest, true)
    assert.deepEqual(
      getCreationResolutionOptions(model).map((item) => item.value),
      ['720p', '480p']
    )

    const options = normalizeCreationVideoOptions(
      { resolution: '480p', duration: '6', aspectRatio: '9:16' },
      model
    )
    assert.deepEqual(getCreationVideoRequestOptions(options, model), {
      duration: 6,
      ratio: '9:16',
      resolution: '480p',
      estimateSeconds: 150,
    })
  })
})

describe('MiniMax H3 creation model options', () => {
  const model = 'minimax-h3'

  test('matches the documented controls and limits', () => {
    const capability = getCreationVideoCapabilities(model)

    assert.equal(capability?.kind, 'minimax-h3')
    assert.equal(capability?.showResolution, false)
    assert.equal(capability?.includeResolutionInRequest, false)
    assert.deepEqual(capability?.aspectRatios, [
      '16:9',
      '9:16',
      '1:1',
      '4:3',
      '3:4',
      '21:9',
    ])
    assert.deepEqual(capability?.referenceModes, [
      'text',
      'image',
      'frames',
      'image-audio',
    ])
    assert.deepEqual(
      getCreationDurationOptions(model).map((item) => item.value),
      ['5', '6', '7', '8', '9', '10', '11', '12', '13', '14', '15']
    )
    assert.deepEqual(
      getCreationResolutionOptions(model).map((item) => item.value),
      ['2k']
    )
    assert.deepEqual(getCreationVideoReferenceLimits(model), {
      maxImages: 5,
      maxVideos: 0,
      maxAudios: 1,
      maxMediaFiles: undefined,
      maxImageSizeBytes: 20 * 1024 * 1024,
      maxVideoSizeBytes: 200 * 1024 * 1024,
      maxAudioSizeBytes: 15 * 1024 * 1024,
      maxImageSizeMB: 20,
      maxVideoSizeMB: 200,
      maxAudioSizeMB: 15,
      minReferenceAudioDurationSeconds: 2,
      maxReferenceAudioDurationSeconds: 15,
    })
    assert.equal(getCreationPromptMaxLength(model), 2000)
  })

  test('keeps the documented H3 contract when catalog metadata is present', () => {
    const catalogModel = {
      id: 'minimax-h3',
      metadata: {
        provider: 'sanbao',
        type: 'video',
        durations: [5, 10],
        resolutions: ['720p'],
        max_prompt_length: 5000,
      },
    }

    const capability = getCreationVideoCapabilities(catalogModel)
    assert.equal(capability?.kind, 'minimax-h3')
    assert.deepEqual(
      getCreationDurationOptions(catalogModel).map((item) => item.value),
      ['5', '6', '7', '8', '9', '10', '11', '12', '13', '14', '15']
    )
    assert.equal(getCreationPromptMaxLength(catalogModel), 2000)
  })

  test('builds text, multi-image, frame, and image-audio requests', () => {
    const options = normalizeCreationVideoOptions(
      { resolution: '480p', duration: '8', aspectRatio: '21:9' },
      model
    )
    assert.deepEqual(options, {
      resolution: '2k',
      duration: '8',
      aspectRatio: '21:9',
    })
    assert.deepEqual(getCreationVideoRequestOptions(options, model), {
      duration: 8,
      aspect_ratio: '21:9',
      estimateSeconds: 180,
    })

    assert.deepEqual(
      getCreationVideoRequestOptions(options, model, {
        referenceMode: 'image',
        imageUrls: [
          { url: 'https://example.com/one.png' },
          { url: 'https://example.com/two.png' },
        ],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [],
        audioUrls: [],
        audioUrl: '',
      }),
      {
        duration: 8,
        aspect_ratio: '21:9',
        estimateSeconds: 180,
        image_urls: [
          'https://example.com/one.png',
          'https://example.com/two.png',
        ],
      }
    )

    assert.deepEqual(
      getCreationVideoRequestOptions(options, model, {
        referenceMode: 'frames',
        imageUrls: [],
        startImageUrl: 'https://example.com/start.png',
        endImageUrl: 'https://example.com/end.png',
        videoUrls: [],
        audioUrls: [],
        audioUrl: '',
      }),
      {
        duration: 8,
        aspect_ratio: '21:9',
        estimateSeconds: 180,
        start_image_url: 'https://example.com/start.png',
        end_image_url: 'https://example.com/end.png',
      }
    )

    assert.deepEqual(
      getCreationVideoRequestOptions(options, model, {
        referenceMode: 'image-audio',
        imageUrls: [{ url: 'https://example.com/scene.png' }],
        startImageUrl: '',
        endImageUrl: '',
        videoUrls: [],
        audioUrls: [{ url: 'https://example.com/sound.m4a' }],
        audioUrl: { url: 'https://example.com/sound.m4a' },
      }),
      {
        duration: 8,
        aspect_ratio: '21:9',
        estimateSeconds: 180,
        image_url: 'https://example.com/scene.png',
        audio_url: 'https://example.com/sound.m4a',
      }
    )
  })

  test('enforces H3 reference-mode combinations and formats', () => {
    const missingImage = normalizeCreationVideoReferences(
      { referenceMode: 'image' },
      model
    )
    assert.equal(
      getCreationVideoReferenceError(model, missingImage),
      'Image reference mode requires at least one image reference.'
    )

    const incompleteFrames = normalizeCreationVideoReferences(
      {
        referenceMode: 'frames',
        startImageUrl: 'https://example.com/start.png',
      },
      model
    )
    assert.equal(
      getCreationVideoReferenceError(model, incompleteFrames),
      'Start/end frame mode requires both a start frame and an end frame.'
    )

    const audioWithoutImage = normalizeCreationVideoReferences(
      {
        referenceMode: 'image-audio',
        audioUrls: ['https://example.com/sound.m4a'],
      },
      model
    )
    assert.equal(
      getCreationVideoReferenceError(model, audioWithoutImage),
      'Audio reference requires at least one image reference.'
    )

    const imageWithoutAudio = normalizeCreationVideoReferences(
      {
        referenceMode: 'image-audio',
        imageUrls: ['https://example.com/scene.png'],
      },
      model
    )
    assert.equal(
      getCreationVideoReferenceError(model, imageWithoutAudio),
      'Image and audio mode requires at least one image and one audio reference.'
    )

    const validExtendedAudio = normalizeCreationVideoReferences(
      {
        referenceMode: 'image-audio',
        imageUrls: ['https://example.com/scene.png'],
        audioUrls: ['https://example.com/sound.webm'],
      },
      model
    )
    assert.equal(
      getCreationVideoReferenceError(model, validExtendedAudio),
      undefined
    )
  })
})
