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
  classifyGroupPricingEcho,
  createGroupPricingEchoGuard,
} from './group-pricing-state'

describe('pricing group editor state', () => {
  const currentGroupRatio = JSON.stringify({ default: 1, vip: 0.9 })
  const currentUserUsableGroups = JSON.stringify({
    default: 'Default group',
    vip: 'VIP group',
  })
  const nextValues = {
    GroupRatio: JSON.stringify({ default: 1, svip: 0.9 }),
    UserUsableGroups: JSON.stringify({
      default: 'Default group',
      svip: 'VIP group',
    }),
  }

  test('recognizes both intermediate parent echoes without resetting rows', () => {
    const guard = createGroupPricingEchoGuard(
      currentGroupRatio,
      currentUserUsableGroups,
      nextValues
    )

    assert.deepEqual(
      classifyGroupPricingEcho(
        guard,
        nextValues.GroupRatio,
        currentUserUsableGroups
      ),
      { expected: true, complete: false }
    )
    assert.deepEqual(
      classifyGroupPricingEcho(
        guard,
        currentGroupRatio,
        nextValues.UserUsableGroups
      ),
      { expected: true, complete: false }
    )
  })

  test('recognizes the final parent echo and rejects external changes', () => {
    const guard = createGroupPricingEchoGuard(
      currentGroupRatio,
      currentUserUsableGroups,
      nextValues
    )

    assert.deepEqual(
      classifyGroupPricingEcho(
        guard,
        nextValues.GroupRatio,
        nextValues.UserUsableGroups
      ),
      { expected: true, complete: true }
    )
    assert.deepEqual(
      classifyGroupPricingEcho(
        guard,
        JSON.stringify({ external: 2 }),
        nextValues.UserUsableGroups
      ),
      { expected: false, complete: false }
    )
  })
})
