import { describe, expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import {
  buildPlaygroundMediaRequest,
  getPlaygroundVideoChoices,
  resolvePlaygroundVideoSelection,
} from '../src/features/playground/lib/media-routing'
import type {
  Message,
  PlaygroundModelInfo,
} from '../src/features/playground/types'

const message: Message = {
  key: 'prompt',
  from: 'user',
  versions: [{ id: 'one', content: '山间日出' }],
}
const info: PlaygroundModelInfo = {
  model: {
    id: 1,
    model_name: 'dola2.0',
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
          durations: [15],
          resolutions: ['720p'],
          omit_ratio: true,
          duration_field: 'seconds',
          ratio_field: 'ratio',
          resolution_field: 'resolution',
          image_field: 'images',
          video_field: 'videos',
          audio_field: 'audios',
          max_images: 9,
          max_videos: 0,
          max_audios: 0,
        },
      },
    ],
  },
  endpoints: {},
}

describe('V6K unified video request', () => {
  it('uses the documented fixed duration and seconds field', () => {
    const selection = { duration: '5', aspectRatio: '16:9', resolution: '720p' }
    expect(
      resolvePlaygroundVideoSelection('dola2.0', selection, info).duration
    ).toBe('15')
    expect(
      buildPlaygroundMediaRequest(
        'dola2.0',
        [message],
        'video',
        'auto',
        {},
        selection,
        info
      )
    ).toEqual({
      model: 'dola2.0',
      prompt: '山间日出',
      seconds: 15,
      resolution: '720p',
    })
  })
  it('allows manual seconds when the provider does not publish a range', () => {
    const unpublished = structuredClone(info)
    unpublished.model.mapped_endpoints![0].parameters!.durations = []
    const selection = {
      duration: '23',
      aspectRatio: '16:9',
      resolution: '720p',
    }
    expect(
      getPlaygroundVideoChoices('XXsedanco2.0S', unpublished).manualDuration
    ).toBe(true)
    expect(
      getPlaygroundVideoChoices('XXsedanco2.0S', unpublished).durations
    ).toEqual([])
    expect(
      buildPlaygroundMediaRequest(
        'XXsedanco2.0S',
        [message],
        'video',
        'auto',
        {},
        selection,
        unpublished
      )
    ).toHaveProperty('seconds', 23)
  })
})


describe('ZH new video model catalog contracts', () => {
  type Parameters = NonNullable<NonNullable<PlaygroundModelInfo['model']['mapped_endpoints']>[number]['parameters']>
  const parameters = JSON.parse(readFileSync(new URL('../../../common/v6k_video_models.json', import.meta.url), 'utf8')) as Record<string, Parameters>
  for (const [name, duration, resolution] of [
    ['Cseadanco2.0-1','15','720p'], ['Cseadanco2.0-fast','15','720p'], ['Cseadanco2.0-mini','15','720p'],
    ['Cseadanco2.5-720','30','720p'], ['TTTseadanco2.5/480','30','480p'], ['TTTseadanco2.5/720','30','720p'],
    ['XXseadanco2.5-1080','30','1080p'], ['XXseadanco2.5-480','30','480p'], ['XXseadanco2.5-720','30','720p'],
    ['dola2.5','15','720p'], ['wan3.M','30','480p'],
  ]) {
    it(`uses the exact contract for ${name}`, () => {
      const selected = structuredClone(info)
      selected.model.model_name = name
      selected.model.mapped_endpoints![0].parameters = parameters[name]
      const choices = getPlaygroundVideoChoices(name,selected)
      expect(choices.durations).toContain(duration)
      expect(choices.resolutions).toContain(resolution)
      expect(choices.aspectRatios).toEqual([])
      const request = buildPlaygroundMediaRequest(name,[message],'video','auto',{}, { duration, aspectRatio:'16:9',resolution },selected)
      expect(request).toMatchObject({ model:name,seconds:Number(duration),resolution })
      expect(request).not.toHaveProperty('ratio')
      expect(request).not.toHaveProperty('duration')
    })
  }
})
