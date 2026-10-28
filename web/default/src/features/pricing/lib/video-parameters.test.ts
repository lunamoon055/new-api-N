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
import type { IntegrationParameters, PricingModel } from '../types'
import { buildSupportedParameters } from './mock-stats'
import { buildVideoSupportedParameters } from './video-parameters'

function modelWithContract(
  parameters?: IntegrationParameters,
  modelName = 'Grok 1.5'
): PricingModel {
  return {
    id: 1,
    model_name: modelName,
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
        parameters,
      },
    ],
  }
}

const grokParameters: IntegrationParameters = {
  durations: Array.from({ length: 13 }, (_, index) => index + 3),
  resolutions: ['720p'],
  ratios: ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9'],
  duration_field: 'duration',
  ratio_field: 'ratio',
  resolution_field: 'resolution',
  image_field: 'reference_images',
  video_field: 'reference_videos',
  audio_field: 'reference_audios',
  max_images: 7,
  max_videos: 0,
  max_audios: 0,
}

describe('video model supported parameters', () => {
  test('uses mapped capability data for public aliases without chat parameters', () => {
    const result = buildVideoSupportedParameters(
      modelWithContract(grokParameters)
    )!
    assert.equal(result.hasDetails, true)
    assert.deepEqual(
      result.parameters.map((parameter) => parameter.name),
      [
        'model',
        'prompt',
        'duration',
        'ratio',
        'resolution',
        'reference_images',
        'reference_videos',
        'reference_audios',
      ]
    )
    assert.deepEqual(result.parameters[0].enumValues, ['Grok 1.5'])
    assert.equal(
      result.parameters.find((parameter) => parameter.name === 'duration')
        ?.range,
      '3 ~ 15'
    )
    assert.deepEqual(
      result.parameters.find((parameter) => parameter.name === 'resolution')
        ?.enumValues,
      ['720p']
    )
    assert.equal(
      result.parameters.find(
        (parameter) => parameter.name === 'reference_images'
      )?.range,
      '0 ~ 7'
    )
    assert(
      result.parameters.every(
        (parameter) => parameter.defaultValue === undefined
      )
    )
  })

  test('keeps distinct Seedance contracts and does not infer limits from names', () => {
    const p = {
      ...grokParameters,
      durations: [5, 10, 15, 30],
      resolutions: ['480p', '720p', '1080p'],
      max_images: 30,
      max_videos: 10,
      max_audios: 10,
    }
    const result = buildVideoSupportedParameters(
      modelWithContract(p, 'Seedance 2.5:D')
    )!
    assert.deepEqual(
      result.parameters.find((parameter) => parameter.name === 'duration')
        ?.enumValues,
      ['5', '10', '15', '30']
    )
    assert.equal(
      result.parameters.find((parameter) => parameter.name === 'duration')
        ?.range,
      undefined
    )
    assert.equal(
      result.parameters.find(
        (parameter) => parameter.name === 'reference_videos'
      )?.range,
      '0 ~ 10'
    )
    const single = buildVideoSupportedParameters(
      modelWithContract(
        {
          ...p,
          image_field: 'referenceImages',
          max_images: 10,
          max_videos: 0,
          max_audios: 0,
        },
        'Seedance2.5-c'
      )
    )!
    assert.equal(
      single.parameters.find(
        (parameter) => parameter.name === 'referenceImages'
      )?.range,
      '0 ~ 10'
    )
    for (const name of ['reference_videos', 'reference_audios']) {
      assert.equal(
        single.parameters.find((parameter) => parameter.name === name)
          ?.availability,
        'unsupported'
      )
    }
  })

  test('uses request field names and fixed resolution values verbatim', () => {
    const result = buildVideoSupportedParameters(
      modelWithContract({
        ...grokParameters,
        duration_field: 'seconds',
        durations: [30],
        ratio_field: 'aspect_ratio',
        resolution_field: 'size',
        resolutions: undefined,
        request_resolution: '2K',
        image_field: 'image_url',
        max_images: 1,
        min_images: 1,
      })
    )!
    assert.equal(
      result.parameters.find((parameter) => parameter.name === 'seconds')?.type,
      'string'
    )
    assert.deepEqual(
      result.parameters.find((parameter) => parameter.name === 'seconds')
        ?.enumValues,
      ['30']
    )
    assert.deepEqual(
      result.parameters.find((parameter) => parameter.name === 'size')
        ?.enumValues,
      ['2K']
    )
    const image = result.parameters.find(
      (parameter) => parameter.name === 'image_url'
    )!
    assert.equal(image.type, 'string')
    assert.equal(image.required, true)
    assert.equal(image.range, '1 ~ 1')
  })

  test('shows frame alternatives and their shared image limit only when enabled', () => {
    const result = buildVideoSupportedParameters(
      modelWithContract({
        ...grokParameters,
        min_images: 1,
        allow_frames: true,
      })
    )!
    assert.equal(
      result.parameters.find(
        (parameter) => parameter.name === 'reference_images'
      )?.required,
      false
    )
    for (const field of ['start_image_url', 'end_image_url']) {
      const frame = result.parameters.find(
        (parameter) => parameter.name === field
      )!
      assert.equal(frame.type, 'string')
      assert.equal(frame.notes?.[0].values?.count, 7)
    }
    const unsupported = buildVideoSupportedParameters(
      modelWithContract(grokParameters)
    )!
    assert(
      !unsupported.parameters.some(
        (parameter) => parameter.name === 'start_image_url'
      )
    )
  })

  test('reflects media size limits and image-only audio prerequisites', () => {
    const result = buildVideoSupportedParameters(
      modelWithContract({
        ...grokParameters,
        max_audios: 3,
        max_audio_size_mb: 10,
        audio_requires_image: true,
        audio_needs_image: true,
      })
    )!
    const audio = result.parameters.find(
      (parameter) => parameter.name === 'reference_audios'
    )!
    assert.equal(
      audio.notes?.find((note) => note.values?.size)?.values?.size,
      10
    )
    assert(
      audio.notes?.some(
        (note) => note.key === 'Audio references require an image reference.'
      )
    )
    assert(
      !audio.notes?.some(
        (note) =>
          note.key === 'Audio references require an image or video reference'
      )
    )
    const visual = buildVideoSupportedParameters(
      modelWithContract({
        ...grokParameters,
        max_audios: 3,
        audio_needs_image: true,
      })
    )!
    assert(
      visual.parameters
        .find((parameter) => parameter.name === 'reference_audios')
        ?.notes?.some(
          (note) =>
            note.key === 'Audio references require an image or video reference'
        )
    )
  })

  test('does not invent limits or optional fields when the contract is unavailable', () => {
    for (const model of [
      modelWithContract(),
      {
        ...modelWithContract(),
        mapped_endpoints: undefined,
        supported_endpoint_types: ['openai-video'],
      },
      {
        ...modelWithContract(undefined, 'sora-2'),
        mapped_endpoints: undefined,
      },
    ]) {
      const result = buildVideoSupportedParameters(model)!
      assert.equal(result.hasDetails, false)
      assert.deepEqual(
        result.parameters.map((parameter) => parameter.name),
        ['model', 'prompt']
      )
    }
  })

  test('keeps unpublished ranges distinct from explicitly unsupported media', () => {
    const result = buildVideoSupportedParameters(
      modelWithContract({
        ...grokParameters,
        durations: [],
        ratios: [],
        resolutions: [],
        max_images: 0,
      })
    )!
    assert.equal(result.hasDetails, true)
    assert.equal(result.hasUnconfirmedParameters, true)
    assert.deepEqual(
      result.parameters.map((parameter) => parameter.name),
      [
        'model',
        'prompt',
        'duration',
        'ratio',
        'resolution',
        'reference_images',
        'reference_videos',
        'reference_audios',
      ]
    )
    for (const name of ['duration', 'ratio', 'resolution']) {
      assert.equal(
        result.parameters.find((parameter) => parameter.name === name)
          ?.availability,
        'unconfirmed'
      )
    }
    for (const name of [
      'reference_images',
      'reference_videos',
      'reference_audios',
    ]) {
      assert.equal(
        result.parameters.find((parameter) => parameter.name === name)
          ?.availability,
        'unsupported'
      )
    }
  })

  test('unknown mapped profiles keep gateway fields without claiming zero support or invented limits', () => {
    const model = modelWithContract({
      capabilities_unknown: true,
      duration_field: 'seconds',
      resolution_field: 'resolution',
      image_field: 'images',
      video_field: 'videos',
      audio_field: 'audios',
      max_images: 0,
      max_videos: 0,
      max_audios: 0,
    })
    model.mapped_endpoints![0].documentation_fields = {
      ...model.mapped_endpoints![0].parameters!,
      duration_field: 'duration',
      image_field: 'referenceImages',
      video_field: 'referenceVideos',
      audio_field: 'referenceAudios',
    }
    const result = buildVideoSupportedParameters(model)!
    assert.equal(result.hasUnconfirmedParameters, true)
    assert.deepEqual(
      result.parameters.map((parameter) => parameter.name),
      [
        'model',
        'prompt',
        'duration',
        'resolution',
        'referenceImages',
        'referenceVideos',
        'referenceAudios',
      ]
    )
    for (const parameter of result.parameters.slice(2)) {
      assert.equal(parameter.availability, 'unconfirmed')
      assert.equal(parameter.range, undefined)
      assert.equal(parameter.defaultValue, undefined)
      assert(
        !parameter.notes?.some(
          (note) => note.key === 'At most {{count}} references.'
        )
      )
    }
  })

  test('prefers a video endpoint with published parameters', () => {
    const model = modelWithContract(grokParameters)
    model.mapped_endpoints!.unshift({
      type: 'custom-video',
      kind: 'video',
      path: '/custom/videos',
      method: 'POST',
    })
    assert.equal(buildVideoSupportedParameters(model)?.hasDetails, true)
  })

  test('preserves chat, reasoning, embedding and image parameter tables', () => {
    for (const name of ['gpt-4o', 'o3', 'text-embedding-3-small', 'dall-e-3']) {
      const model = {
        ...modelWithContract(undefined, name),
        mapped_endpoints: undefined,
      }
      const previous = structuredClone(buildSupportedParameters(model))
      assert.equal(buildVideoSupportedParameters(model), undefined)
      assert.deepEqual(buildSupportedParameters(model), previous)
    }
    const explicitText = modelWithContract(undefined, 'sora-chat')
    explicitText.mapped_endpoints = [
      {
        type: 'openai',
        kind: 'text',
        path: '/v1/chat/completions',
        method: 'POST',
      },
    ]
    assert.equal(buildVideoSupportedParameters(explicitText), undefined)
  })

  test('does not mutate the catalog contract', () => {
    const model = modelWithContract({
      ...grokParameters,
      durations: [15, 3, 15, 6],
    })
    const previous = structuredClone(model)
    buildVideoSupportedParameters(model)
    assert.deepEqual(model, previous)
  })
})
