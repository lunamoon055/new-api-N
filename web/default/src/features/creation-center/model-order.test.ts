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
  getCreationModelsByMode,
  moveCreationModel,
  serializeCreationModelOrder,
} from './model-order'
import type { CreationModel, CreationModelGroup } from './types'

function model(id: string): CreationModel {
  return { id, supported_endpoint_types: [] }
}

describe('Creation Center model ordering', () => {
  const groups: CreationModelGroup[] = [
    { mode: 'chat', models: [model('chat-a'), model('chat-b')] },
    { mode: 'image', models: [model('image-a')] },
    { mode: 'video', models: [model('video-a'), model('video-b')] },
  ]

  test('builds independent editable lists for every creation mode', () => {
    const modelsByMode = getCreationModelsByMode(groups)

    assert.deepEqual(
      modelsByMode.chat.map((item) => item.id),
      ['chat-a', 'chat-b']
    )
    modelsByMode.chat.reverse()
    assert.deepEqual(
      groups[0].models.map((item) => item.id),
      ['chat-a', 'chat-b']
    )
  })

  test('moves a model to the requested bounded position', () => {
    const models = [model('a'), model('b'), model('c')]

    assert.deepEqual(
      moveCreationModel(models, 'a', 2).map((item) => item.id),
      ['b', 'c', 'a']
    )
    assert.deepEqual(
      moveCreationModel(models, 'c', -4).map((item) => item.id),
      ['c', 'a', 'b']
    )
    assert.equal(moveCreationModel(models, 'missing', 1), models)
  })

  test('serializes the visible order for persistence', () => {
    const modelsByMode = getCreationModelsByMode(groups)
    modelsByMode.video = moveCreationModel(modelsByMode.video, 'video-b', 0)

    assert.deepEqual(serializeCreationModelOrder(modelsByMode), {
      chat: ['chat-a', 'chat-b'],
      image: ['image-a'],
      video: ['video-b', 'video-a'],
    })
  })
})
