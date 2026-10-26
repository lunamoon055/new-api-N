import { describe, expect, it } from 'bun:test'
import {
  buildPlaygroundMediaRequest,
  getPlaygroundVideoChoices,
  resolvePlaygroundVideoSelection,
} from '../src/features/playground/lib/media-routing'
import type { Message, PlaygroundModelInfo } from '../src/features/playground/types'

const model = 'XXseadanco2.0M'
const info: PlaygroundModelInfo = {
  model: {
    id: 1, model_name: model, quota_type: 1, model_ratio: 1,
    completion_ratio: 1, enable_groups: ['default'],
    mapped_endpoints: [{ type: 'openai-video', kind: 'video', path: '/v1/videos', method: 'POST',
      parameters: {
        omit_ratio: true, durations: [5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
        resolutions: ['720p'], duration_field: 'duration', ratio_field: 'ratio', resolution_field: 'resolution',
        image_field: 'images', video_field: 'reference_videos', audio_field: 'reference_audios',
        max_images: 9, max_videos: 3, max_audios: 3,
      },
    }],
  }, endpoints: {},
}
const selection = { duration: '5', aspectRatio: '16:9', resolution: '720p' }
const message: Message = { key: 'prompt', from: 'user', versions: [{ id: 'prompt', content: '两个人在进行武术切磋，一人扮演进攻方，一人扮演防守方' }] }

describe('Zongheng playground video parameters', () => {
  it('offers the full site ratio menu when the model publishes none', () => {
    expect(getPlaygroundVideoChoices(model, info).aspectRatios).toEqual(['16:9', '9:16', '1:1', '4:3', '3:4', '2:3', '3:2', '21:9'])
    expect(resolvePlaygroundVideoSelection(model, selection, info).aspectRatio).toBe('16:9')
  })

  it('includes the selected ratio in the actual generation request', () => {
    const request = buildPlaygroundMediaRequest(model, [message], 'video', 'auto', {}, selection, info)
    expect(request).toEqual({ model, prompt: message.versions[0].content, duration: 5, resolution: '720p', ratio: '16:9' })
    expect(request).not.toHaveProperty('aspect_ratio')
    expect(request).not.toHaveProperty('size')
  })

  it('preserves published ratios on existing models', () => {
    const supported: PlaygroundModelInfo = structuredClone(info)
    const parameters = supported.model.mapped_endpoints![0].parameters!
    parameters.omit_ratio = false
    parameters.ratios = ['16:9', '9:16']
    expect(getPlaygroundVideoChoices(model, supported).aspectRatios).toEqual(['16:9', '9:16'])
    expect(buildPlaygroundMediaRequest(model, [message], 'video', 'auto', {}, selection, supported)).toHaveProperty('ratio', '16:9')
  })
})
