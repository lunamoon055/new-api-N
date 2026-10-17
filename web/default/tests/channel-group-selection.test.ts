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
import { resolvePlaygroundGroup } from '../src/features/playground/lib/model-integration'
import type { ModelOption } from '../src/features/playground/types'

const models: ModelOption[] = [
  {
    label: 'my-video',
    value: 'my-video',
    modelInfo: {
      model: {
        id: 1,
        model_name: 'my-video',
        quota_type: 1,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: ['vip', 'SVIP'],
      },
      endpoints: {},
    },
  },
]

describe('playground group selection', () => {
  const groups = [{ value: 'default' }, { value: 'SVIP' }, { value: 'vip' }]
  it('recovers from a saved default after the channel becomes VIP-only', () => {
    expect(resolvePlaygroundGroup(groups, models, 'default', 'vip')).toBe('vip')
    expect(resolvePlaygroundGroup(groups, models, 'default', 'SVIP')).toBe(
      'SVIP'
    )
  })
  it('preserves explicit available selections and cannot select unauthorized groups', () => {
    expect(resolvePlaygroundGroup(groups, models, 'SVIP', 'vip')).toBe('SVIP')
    expect(
      resolvePlaygroundGroup([{ value: 'default' }], models, 'default', 'vip')
    ).toBe('default')
    expect(resolvePlaygroundGroup([], models, 'default', 'vip')).toBe('default')
  })
})
