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
import { supportsChannelCapabilitySync } from './channel-capabilities'

test('capability sync uses supported channel types and exact provider hosts', () => {
  for (const base_url of [
    'https://suanliai.top',
    'https://suanliai.top/v1/',
    'https://api.meaicc.com',
  ]) {
    assert.equal(supportsChannelCapabilitySync({ type: 1, base_url }), true)
  }
  for (const base_url of [
    'https://suanliai.top.other.test',
    'https://other.test/api.meaicc.com',
    'invalid',
  ]) {
    assert.equal(supportsChannelCapabilitySync({ type: 1, base_url }), false)
  }
  assert.equal(supportsChannelCapabilitySync({ type: 59, base_url: '' }), true)
  assert.equal(supportsChannelCapabilitySync({ type: 60, base_url: '' }), true)
  assert.equal(
    supportsChannelCapabilitySync({
      type: 14,
      base_url: 'https://suanliai.top',
    }),
    false
  )
})
