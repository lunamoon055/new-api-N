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
import { test } from 'node:test'
import {
  buildPlaygroundResponsesPayload,
  parsePlaygroundResponsesText,
} from './responses'
import type { Message } from '../types'

test('Responses request preserves the selected model and group', () => {
  const messages: Message[] = [
    { key: 'one', from: 'user', versions: [{ id: 'v1', content: 'hello' }] },
  ]
  assert.deepEqual(
    buildPlaygroundResponsesPayload('new-response-model', 'paid', messages, {
      group: 'other',
      max_output_tokens: 64,
    }),
    {
      model: 'new-response-model',
      group: 'paid',
      input: [{ role: 'user', content: 'hello' }],
      stream: false,
      max_output_tokens: 64,
    }
  )
})

test('Responses output text is displayed from content parts', () => {
  assert.equal(
    parsePlaygroundResponsesText({
      output: [{ content: [{ type: 'output_text', text: 'hello' }] }],
    }),
    'hello'
  )
})
