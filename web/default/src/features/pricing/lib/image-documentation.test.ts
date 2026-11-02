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
import { buildNativeImageSample } from './image-documentation'

test('image examples keep edits multipart and generation JSON for every language', () => {
  for (const lang of ['curl', 'python', 'typescript', 'javascript'] as const) {
    const options = {
      baseUrl: 'https://gateway.example',
      modelName: 'public-image',
      apiKeyEnv: 'NEW_API_KEY',
    }
    const edit = buildNativeImageSample(lang, {
      ...options,
      endpointPath: '/v1/images/edits',
    })
    assert.match(edit, /image\[\]/)
    assert.match(edit, /\/v1\/images\/edits/)
    assert.doesNotMatch(edit, /Content-Type.*application\/json/)
    const generation = buildNativeImageSample(lang, {
      ...options,
      endpointPath: '/v1/images/generations',
    })
    assert.match(generation, /resolution/)
    assert.match(generation, /ratio/)
    assert.doesNotMatch(generation, /image\[\]/)
  }
})
