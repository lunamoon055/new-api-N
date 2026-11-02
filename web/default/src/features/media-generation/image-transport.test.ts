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
import { prepareImageTransport } from './image-transport'

test('native reference requests upload files to edits; text-only requests retain their endpoint', async () => {
  const originalFetch = globalThis.fetch
  globalThis.fetch = (async () =>
    new Response(
      new Blob(['reference'], { type: 'image/png' })
    )) as typeof fetch
  try {
    const plain = {
      model: 'gpt-image-2',
      prompt: 'cup',
      ratio: '1:1',
      resolution: '1K',
    }
    const unchanged = await prepareImageTransport(
      '/api/creation/images/generations',
      plain
    )
    assert.equal(unchanged.payload, plain)
    const edited = await prepareImageTransport(
      '/api/creation/images/generations',
      { ...plain, 'image[]': ['https://example.com/ref.png'] }
    )
    assert.equal(edited.endpoint, '/api/creation/images/edits')
    assert.ok(edited.payload instanceof FormData)
    assert.equal(edited.payload.get('model'), plain.model)
    assert.equal(edited.payload.get('resolution'), '1K')
    assert.equal((edited.payload.get('image[]') as Blob).type, 'image/png')
    assert.equal(edited.payload.get('images'), null)
  } finally {
    globalThis.fetch = originalFetch
  }
})
