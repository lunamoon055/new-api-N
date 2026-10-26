import { describe, expect, it } from 'bun:test'
import {
  buildPlaygroundMediaRequest,
  getPlaygroundVideoChoices,
} from '../src/features/playground/lib/media-routing'
import type { Message, PlaygroundModelInfo } from '../src/features/playground/types'

const ratios = ['16:9', '9:16', '1:1', '4:3', '3:4', '2:3', '3:2', '21:9']
const message: Message = { key: 'prompt', from: 'user', versions: [{ id: 'one', content: '海边日落' }] }
const info: PlaygroundModelInfo = {
  model: {
    id: 1, model_name: 'future-video', quota_type: 1, model_ratio: 1,
    completion_ratio: 1, enable_groups: ['default'],
    mapped_endpoints: [{type: 'openai-video',kind: 'video',path: '/v1/videos',method: 'POST',
      parameters: {omit_ratio: true, durations:[5], duration_field:'seconds', max_images:0,max_videos:0,max_audios:0},
    }],
  }, endpoints: {},
}

describe('all unconfigured video ratios', () => {
  it('offers every ratio and sends the selected value even with older omit hints', () => {
    expect(getPlaygroundVideoChoices('future-video', info).aspectRatios).toEqual(ratios)
    for (const ratio of ratios) {
      expect(buildPlaygroundMediaRequest('future-video',[message],'video','auto',{},
        {duration:'5',aspectRatio:ratio},info)).toMatchObject({seconds:5,ratio})
    }
  })

  it('covers models without mapped parameters and preserves explicit ranges', () => {
    const missing = structuredClone(info)
    missing.model.mapped_endpoints![0].parameters = undefined
    expect(getPlaygroundVideoChoices('future-video',missing).aspectRatios).toEqual(ratios)
    for (const ratio of ratios) {
      const request = buildPlaygroundMediaRequest('future-video',[message],'video','auto',{},
        {duration:'5',aspectRatio:ratio},missing)
      expect(request).toHaveProperty('aspect_ratio',ratio)
      expect(request).toHaveProperty('size')
    }
    const known = structuredClone(info)
    known.model.mapped_endpoints![0].parameters!.ratios = ['9:16','1:1']
    expect(getPlaygroundVideoChoices('future-video',known).aspectRatios).toEqual(['9:16','1:1'])
    expect(buildPlaygroundMediaRequest('future-video',[message],'video','auto',{},
      {duration:'5',aspectRatio:'21:9'},known)).toHaveProperty('ratio','9:16')
  })
})
