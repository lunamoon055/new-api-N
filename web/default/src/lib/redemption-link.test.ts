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
import test from 'node:test'
import { buildRedemptionLink, readRedemptionLinkCode } from './redemption-link'

test('delivery links keep the code in the fragment and round-trip safely', () => {
  const key = '0123456789abcdef0123456789abcdef'
  const url = new URL(buildRedemptionLink(key, 'https://ghlink.top'))
  assert.equal(url.pathname, '/redeem')
  assert.equal(url.search, '')
  assert.equal(readRedemptionLinkCode(url.hash), key)
  assert.equal(readRedemptionLinkCode('#code=a%2Bb%26c'), 'a+b&c')
})

test('incomplete or excessive delivery codes cannot initialize a redemption', () => {
  for (const hash of [
    '',
    '#other=value',
    '#code=',
    '#code=hello%0Aworld',
    `#code=${'a'.repeat(129)}`,
  ]) {
    assert.equal(readRedemptionLinkCode(hash), null)
  }
})
