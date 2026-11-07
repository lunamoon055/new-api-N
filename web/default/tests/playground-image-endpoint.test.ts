import { describe, expect, it } from 'bun:test'
import {
  buildPlaygroundMediaRequest,
  getPlaygroundImageEndpointChoices,
  getPlaygroundMediaEndpoint,
  resolvePlaygroundImageEndpoint,
} from '../src/features/playground/lib/media-routing'
import type {
  Message,
  PlaygroundModelInfo,
} from '../src/features/playground/types'
import { buildSupportedParameters } from '../src/features/pricing/lib/mock-stats'
import { getCreationImageRequestOptions } from '../src/features/media-generation/image-options'

const messages: Message[] = [
  {
    key: 'prompt',
    from: 'user',
    versions: [{ id: 'v1', content: 'a quiet landscape' }],
  },
]

function modelInfo(name: string, paths: string[]): PlaygroundModelInfo {
  return {
    model: {
      id: 1,
      model_name: name,
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['vip', 'default'],
      mapped_endpoints: paths.map((path) => ({
        type: 'image-generation',
        kind: 'image',
        method: 'POST',
        path,
        parameters: {
          ratios: ['1:1', '16:9'],
          resolutions: ['1K', '2K'],
          ratio_field: 'aspect_ratio',
          resolution_field: 'output_resolution',
          image_field: path.endsWith('/async-generations')
            ? 'image_urls'
            : 'images',
          max_images: 17,
          max_videos: 0,
          max_audios: 0,
        },
      })),
    },
    endpoints: {},
  }
}

describe('Playground image endpoint follows the current channel contract', () => {
  it('sends the verified single reference as a file-edit input without unconfirmed size options', () => {
    const info = modelInfo('gpt-image-2.5-flare', ['/v1/images/generations'])
    info.model.mapped_endpoints![0].parameters = {
      capabilities_unknown: true,
      max_images: 0,
      max_videos: 0,
      max_audios: 0,
    }
    info.model.mapped_endpoints!.push({
      type: 'image-edit',
      kind: 'image',
      path: '/v1/images/edits',
      method: 'POST',
      parameters: {
        image_field: 'image[]',
        min_images: 1,
        max_images: 1,
        max_videos: 0,
        max_audios: 0,
        unknown_fields: ['max_images'],
      },
    })
    expect(
      buildPlaygroundMediaRequest(
        'gpt-image-2.5-flare', messages, 'image', 'auto', {}, undefined, info,
        { images: 'https://example.com/reference.png', aspectRatio: '16:9', resolution: '4K' }
      )
    ).toEqual({
      model: 'gpt-image-2.5-flare', prompt: 'a quiet landscape', n: 1,
      'image[]': ['https://example.com/reference.png'],
    })
  })

  it('uses the published minimum for MegaAi without borrowing other provider fields', () => {
    for (const name of [
      'gpt-image-2',
      'gpt-image-2.5-flare',
      'gpt-image-2.5-sunburst',
    ]) {
      const info = modelInfo(name, ['/v1/images/generations'])
      info.model.mapped_endpoints![0].parameters = {
        capabilities_unknown: true,
        max_images: 0,
        max_videos: 0,
        max_audios: 0,
      }
      expect(
        buildPlaygroundMediaRequest(
          name,
          messages,
          'image',
          'auto',
          {},
          undefined,
          info,
          { aspectRatio: '16:9', resolution: '4K' }
        )
      ).toEqual({ model: name, prompt: 'a quiet landscape', n: 1 })
      expect(
        getCreationImageRequestOptions('cup', {
          id: name,
          mapped_endpoints: info.model.mapped_endpoints,
        })
      ).toEqual({})
      const parameters = buildSupportedParameters(info.model)
      expect(parameters.map((parameter) => parameter.name)).toEqual([
        'model',
        'prompt',
        'n',
      ])
      expect(parameters.find((parameter) => parameter.name === 'n')?.range).toBeUndefined()
    }
  })

  it('recovers a persisted async selection for the GHLINK synchronous image models and aliases', () => {
    for (const name of [
      'gpt-image-2',
      'gpt-image-2.5-flare',
      'nano-banana2',
      'public-image-alias',
    ]) {
      const info = modelInfo(name, ['/v1/images/generations'])
      expect(getPlaygroundImageEndpointChoices(info)).toEqual(['auto', 'sync'])
      expect(resolvePlaygroundImageEndpoint('async', info)).toBe('auto')
      expect(
        getPlaygroundMediaEndpoint(name, 'image', 'async', 'auto', info)
      ).toBe('/api/creation/images/generations')
      expect(
        buildPlaygroundMediaRequest(
          name,
          messages,
          'image',
          'async',
          {},
          undefined,
          info,
          {
            aspectRatio: '16:9',
            resolution: '2K',
            images: 'https://cdn.example/reference.png',
          }
        )
      ).toEqual({
        model: name,
        prompt: 'a quiet landscape',
        n: 1,
        aspect_ratio: '16:9',
        output_resolution: '2K',
        images: ['https://cdn.example/reference.png'],
      })
    }
  })

  it('keeps asynchronous providers on their documented route and payload', () => {
    const info = modelInfo('gpt-image-2', ['/v1/images/async-generations'])
    expect(getPlaygroundImageEndpointChoices(info)).toEqual(['auto', 'async'])
    expect(resolvePlaygroundImageEndpoint('sync', info)).toBe('auto')
    expect(
      getPlaygroundMediaEndpoint('gpt-image-2', 'image', 'sync', 'auto', info)
    ).toBe('/api/creation/images/async-generations')
    expect(
      buildPlaygroundMediaRequest(
        'gpt-image-2',
        messages,
        'image',
        'sync',
        {},
        undefined,
        info,
        {
          images: 'https://cdn.example/reference.png',
        }
      )
    ).toMatchObject({ image_urls: ['https://cdn.example/reference.png'] })
  })

  it('preserves a supported override and uses its own request contract when both routes are available', () => {
    const info = modelInfo('shared-alias', [
      '/v1/images/async-generations',
      '/v1/images/generations',
    ])
    expect(getPlaygroundImageEndpointChoices(info)).toEqual([
      'auto',
      'async',
      'sync',
    ])
    expect(resolvePlaygroundImageEndpoint('sync', info)).toBe('sync')
    expect(
      getPlaygroundMediaEndpoint('shared-alias', 'image', 'sync', 'auto', info)
    ).toBe('/api/creation/images/generations')
    const payload = buildPlaygroundMediaRequest(
      'shared-alias',
      messages,
      'image',
      'sync',
      {},
      undefined,
      info,
      {
        images: 'https://cdn.example/reference.png',
      }
    )
    expect(payload).toHaveProperty('images', [
      'https://cdn.example/reference.png',
    ])
    expect(payload).not.toHaveProperty('image_urls')
  })

  it('reads supported endpoint metadata when no mapped contract is available', () => {
    const info = modelInfo('gpt-image-2', [])
    info.model.supported_endpoint_types = ['image-generation']
    info.endpoints = {
      'image-generation': { path: '/v1/images/generations', method: 'POST' },
    }
    expect(
      getPlaygroundMediaEndpoint('gpt-image-2', 'image', 'async', 'auto', info)
    ).toBe('/api/creation/images/generations')
  })

  it('keeps manual testing available for models with no published image endpoint', () => {
    expect(getPlaygroundImageEndpointChoices()).toEqual([
      'auto',
      'sync',
      'async',
    ])
    expect(resolvePlaygroundImageEndpoint('async')).toBe('async')
    expect(getPlaygroundMediaEndpoint('new-model', 'image', 'async')).toBe(
      '/api/creation/images/async-generations'
    )
  })
})
