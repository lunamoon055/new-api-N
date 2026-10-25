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
import {
  CREDIT_PURCHASE_TIERS,
  normalizeCreditPurchaseLinks,
  parseCreditPurchaseIntent,
} from './credit-purchase'

test('each storefront tier keeps its own URL and explicit empty tiers stay disabled', () => {
  const config = Object.fromEntries(
    CREDIT_PURCHASE_TIERS.map((tier) => [tier, `https://shop.example/${tier}`])
  )
  assert.deepEqual(normalizeCreditPurchaseLinks(JSON.stringify(config)), config)
  assert.equal(
    normalizeCreditPurchaseLinks({ other: '' }, 'https://shop.example/legacy')
      .other,
    ''
  )
  assert.equal(
    normalizeCreditPurchaseLinks({}, 'https://shop.example/legacy').other,
    'https://shop.example/legacy'
  )
  assert.equal(
    normalizeCreditPurchaseLinks({
      '20': 'javascript:alert(1)',
      '50': 'https://u:p@shop.example',
    })['20'],
    ''
  )
  assert.equal(
    normalizeCreditPurchaseLinks({ '50': 'https://u:p@shop.example' })['50'],
    ''
  )
})

test('purchase return markers reject stale or malformed intents', () => {
  const now = 200000000
  const valid = {
    id: 'test-return',
    tier: '20',
    userId: 7,
    createdAt: now - 1000,
  }
  assert.deepEqual(parseCreditPurchaseIntent(JSON.stringify(valid), now), valid)
  for (const value of [
    { ...valid, tier: '500' },
    { ...valid, userId: 0 },
    { ...valid, createdAt: now + 1 },
    { ...valid, createdAt: now - 86400000 },
  ])
    assert.equal(parseCreditPurchaseIntent(JSON.stringify(value), now), null)
  assert.equal(parseCreditPurchaseIntent('invalid', now), null)
})
