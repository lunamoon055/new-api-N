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
import { parseHeaderNavModules } from '../src/lib/nav-modules'

describe('parseHeaderNavModules', () => {
  it('enables the creation center when older navigation config omits it', () => {
    const modules = parseHeaderNavModules('{"home":true,"console":true}')

    expect(modules.creation).toBe(true)
  })

  it('keeps an explicit creation center disable flag', () => {
    const modules = parseHeaderNavModules('{"creation":false}')

    expect(modules.creation).toBe(false)
  })
})
