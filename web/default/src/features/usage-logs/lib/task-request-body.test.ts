import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { TaskLog } from '../types'
import { getTaskLogRequestBody } from './task-request-body'

const log: TaskLog = {
  id: 1,
  user_id: 1,
  platform: '1',
  task_id: 'task_test',
  model_name: 'public-model',
  action: 'generate',
  channel_id: 1,
  submit_time: 1,
  status: 'FAILURE',
  prompt: 'demo',
  properties: { input_images: ['https://cdn.example/image.png'] },
}

describe('task log request body', () => {
  test('copies server-formatted JSON text without rounding large integer seeds', () => {
    const text = '{\n  "seed": 9007199254740993,\n  "generate_audio": false\n}'
    const result = getTaskLogRequestBody(
      {
        ...log,
        request_body_snapshot: {
          method: 'POST',
          path: '/v1/videos',
          content_type: 'application/json',
          format: 'json',
          body_json: text,
        },
      },
      true
    )
    assert.equal(result.text, text)
    assert(result.preview?.includes('9007199254740993'))
  })
  test('shows captured client parameters exactly instead of synthesizing from task data', () => {
    const captured = {
      model: 'public-model',
      duration: 30,
      ratio: '16:9',
      referenceAudios: ['https://cdn.example/voice.mp3'],
      generate_audio: false,
      seed: 0,
    }
    const result = getTaskLogRequestBody(
      {
        ...log,
        request_body_snapshot: {
          method: 'POST',
          path: '/v1/videos',
          content_type: 'application/json',
          format: 'json',
          body: captured,
        },
      },
      true
    )
    assert.deepEqual(JSON.parse(result.text!), captured)
    assert.equal(result.notice, undefined)
  })

  test('labels historical partial records and never invents duration or ratio', () => {
    const result = getTaskLogRequestBody(log, true)
    const body = JSON.parse(result.text!)
    assert.equal(result.notice, 'historical')
    assert.deepEqual(body, {
      model: 'public-model',
      prompt: 'demo',
      referenceImages: ['https://cdn.example/image.png'],
    })
    assert(!('duration' in body))
  })

  test('does not expose request snapshots or reference URLs without permission', () => {
    const result = getTaskLogRequestBody(
      {
        ...log,
        request_body_snapshot: {
          method: 'POST',
          path: '/v1/videos',
          content_type: 'application/json',
          format: 'json',
          body: { private: 'root-only' },
        },
      },
      false
    )
    assert.equal(result.notice, 'restricted')
    assert.equal(result.snapshot, undefined)
    assert(!result.text?.includes('root-only'))
    assert(!result.text?.includes('cdn.example'))
  })

  test('keeps an omitted snapshot distinct from a historical record', () => {
    const result = getTaskLogRequestBody(
      {
        ...log,
        request_body_snapshot: {
          method: 'POST',
          path: '/v1/videos',
          content_type: 'application/json',
          format: 'unavailable',
          omitted: true,
        },
      },
      true
    )
    assert.equal(result.text, '')
    assert.equal(result.snapshot?.omitted, true)
    assert.equal(result.notice, undefined)
  })
})
