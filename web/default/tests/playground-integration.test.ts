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
import { describe, expect, it } from 'bun:test'
import { buildPlaygroundMediaRequest, getPlaygroundMediaEndpoint, getPlaygroundVideoChoices, parsePlaygroundMediaResult, resolvePlaygroundMode, resolvePlaygroundVideoSelection } from '../src/features/playground/lib/media-routing'
import { pollPlaygroundTask } from '../src/features/playground/lib/media-task-polling'
import { getPlaygroundGroupModels } from '../src/features/playground/lib/model-integration'
import type { IntegrationParameters } from '../src/features/pricing/types'
import type { PlaygroundModelInfo, Message } from '../src/features/playground/types'

const message: Message = { key: 'prompt', from: 'user', versions: [{ id: 'prompt', content: 'A quiet sunrise over the sea' }] }
const parameters: IntegrationParameters = { durations: [6, 7, 8, 9, 10, 11, 12, 13, 14, 15], resolutions: ['480p', '720p'], ratios: ['16:9', '9:16', '1:1'], duration_field: 'duration', ratio_field: 'ratio', resolution_field: 'resolution', image_field: 'images', max_images: 7, max_videos: 0, max_audios: 0 }
function info(name: string, kind: 'video' | 'image', contract: IntegrationParameters, path = '/v1/videos'): PlaygroundModelInfo {
  return { model: { id: 1, model_name: name, quota_type: 1, model_ratio: 1, completion_ratio: 1, enable_groups: ['default'], mapped_endpoints: [{ type: kind === 'video' ? 'openai-video' : 'image-generation', kind, path, method: 'POST', parameters: contract }] }, endpoints: {} }
}

describe('mapped playground integration', () => {
  it('keeps the model selector inside the selected group availability', () => {
    const metadata = info('Grok', 'video', parameters)
    const models = [{ label: 'Grok', value: 'Grok', modelInfo: metadata }]
    expect(getPlaygroundGroupModels(models, 'default')).toHaveLength(1)
    expect(getPlaygroundGroupModels(models, 'unavailable')).toHaveLength(0)
    expect(getPlaygroundGroupModels(models, 'auto')).toHaveLength(1)
  })
  it('uses the mapped contract for an opaque public alias and keeps the alias in the body', () => {
    const metadata = info('Grok', 'video', parameters)
    expect(resolvePlaygroundMode('Grok', 'auto', metadata)).toBe('video')
    expect(getPlaygroundMediaEndpoint('Grok', 'auto', 'auto', 'auto', metadata)).toBe('/api/creation/videos')
    expect(getPlaygroundVideoChoices('Grok', metadata).resolutions).toEqual(['480p', '720p'])
    expect(buildPlaygroundMediaRequest('Grok', [message], 'auto', 'auto', {}, { duration: '6', aspectRatio: '16:9', resolution: '720p' }, metadata, { images: 'https://cdn.example/ref.png' })).toEqual({ model: 'Grok', prompt: message.versions[0].content, duration: 6, ratio: '16:9', resolution: '720p', images: ['https://cdn.example/ref.png'] })
  })
  it('normalizes persisted choices to fixed duration/resolution and the documented field name', () => {
    const metadata = info('dola 2.5 900', 'video', { ...parameters, durations: [30], resolutions: ['720p'], request_resolution: '720P', min_images: 1, image_field: 'reference_images' })
    expect(resolvePlaygroundVideoSelection('dola 2.5 900', { duration: '15', aspectRatio: '16:9', resolution: '4K' }, metadata)).toEqual({ duration: '30', aspectRatio: '16:9', resolution: '720p' })
    expect(() => buildPlaygroundMediaRequest('dola 2.5 900', [message], 'auto', 'auto', {}, { duration: '15', aspectRatio: '16:9' }, metadata)).toThrow()
    const request = buildPlaygroundMediaRequest('dola 2.5 900', [message], 'auto', 'auto', {}, { duration: '15', aspectRatio: '16:9' }, metadata, { images: 'https://cdn.example/ref.png' })
    expect(request).toMatchObject({ duration: 30, resolution: '720P', reference_images: ['https://cdn.example/ref.png'] })
    expect(request).not.toHaveProperty('seconds')
    expect(request).not.toHaveProperty('size')
  })
  it('keeps image resolution, aspect ratio and references in synchronous requests', () => {
    const metadata = info('nano-banana2', 'image', { ratios: ['1:1', '16:9'], resolutions: ['1K', '2K', '4K'], ratio_field: 'aspect_ratio', resolution_field: 'output_resolution', image_field: 'images', max_images: 14, max_videos: 0, max_audios: 0 }, '/v1/images/generations')
    expect(getPlaygroundMediaEndpoint('nano-banana2', 'auto', 'auto', 'auto', metadata)).toBe('/api/creation/images/generations')
    const request = buildPlaygroundMediaRequest('nano-banana2', [message], 'auto', 'auto', {}, { duration: '', aspectRatio: '' }, metadata, { aspectRatio: '16:9', resolution: '4K', images: 'https://cdn.example/1.png\nhttps://cdn.example/2.png' })
    expect(request).toMatchObject({ n: 1, aspect_ratio: '16:9', output_resolution: '4K', images: ['https://cdn.example/1.png', 'https://cdn.example/2.png'] })
    expect(request).not.toHaveProperty('resolution')
  })
  it('rejects unsupported reference kinds and invalid URLs before submission', () => {
    const metadata = info('Grok', 'video', parameters)
    for (const references of [{ videos: 'https://cdn.example/ref.mp4' }, { images: 'data:image/png;base64,abc' }, { images: Array(8).fill('https://cdn.example/ref.png').join('\n') }]) {
      expect(() => buildPlaygroundMediaRequest('Grok', [message], 'auto', 'auto', {}, { duration: '6', aspectRatio: '16:9' }, metadata, references)).toThrow()
    }
  })
  it('accepts a submission message and normalizes succeeded/failed states', () => {
    expect(parsePlaygroundMediaResult({ id: 'job', status: 'queued', message: 'accepted' }, 'alias', 'video')).toMatchObject({ taskId: 'job', status: 'queued' })
    expect(parsePlaygroundMediaResult({ id: 'job', status: 'succeeded', video_url: 'https://cdn.example/video.mp4' }, 'alias', 'video')).toMatchObject({ status: 'completed', mediaUrl: 'https://cdn.example/video.mp4' })
    expect(() => parsePlaygroundMediaResult({ id: 'job', status: 'failed' }, 'alias', 'video')).toThrow()
  })
})

describe('playground task polling', () => {
  it('retries rate limits and temporary server errors using the original task ID', async () => {
    let clock = 0, attempts = 0
    const ids: string[] = []
    const result = await pollPlaygroundTask({ mode: 'video', content: 'queued', taskId: 'accepted-job', status: 'queued' }, {
      now: () => clock,
      wait: async (ms) => { clock += ms },
      fetch: async (id) => {
        ids.push(id)
        attempts++
        if (attempts === 1) throw { response: { status: 429 } }
        if (attempts === 2) throw { response: { status: 503 } }
        if (attempts === 3) return { mode: 'video', content: 'processing', status: 'processing' }
        return { mode: 'video', content: 'done', status: 'completed', mediaUrl: 'https://cdn.example/result.mp4' }
      },
    })
    expect(ids).toEqual(Array(4).fill('accepted-job'))
    expect(result.mediaUrl).toBe('https://cdn.example/result.mp4')
  })
  it('does not retry an explicit generation failure or lose a pending task on timeout', async () => {
    const initial = { mode: 'video' as const, content: 'pending', status: 'queued', taskId: 'job' }
    let clock = 0
    const result = await pollPlaygroundTask(initial, { timeoutMs: 10, now: () => clock, wait: async () => { clock += 10 }, fetch: async () => ({ ...initial, status: 'processing' }) })
    expect(result.taskId).toBe('job')
    expect(result.status).toBe('processing')
    let attempts = 0
    await expect(pollPlaygroundTask(initial, { wait: async () => {}, fetch: async () => { attempts++; throw new Error('Provider generation failed') } })).rejects.toThrow('Provider generation failed')
    expect(attempts).toBe(1)
  })
})
