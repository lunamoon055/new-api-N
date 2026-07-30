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
import { parseModelMappingRows } from './model-mapping-state'

describe('model mapping editor state', () => {
  test('keeps row identities stable while mapping text changes', () => {
    const initialRows = parseModelMappingRows(
      JSON.stringify({ original: 'upstream' })
    )
    assert.ok(initialRows)

    const updatedRows = parseModelMappingRows(
      JSON.stringify({ original2: 'upstream2' }),
      initialRows
    )
    assert.ok(updatedRows)
    assert.equal(updatedRows[0].id, initialRows[0].id)
  })

  test('only assigns a new identity to newly added mappings', () => {
    const initialRows = parseModelMappingRows(
      JSON.stringify({ first: 'one', second: 'two' })
    )
    assert.ok(initialRows)

    const updatedRows = parseModelMappingRows(
      JSON.stringify({ first: 'one', second: 'two', third: 'three' }),
      initialRows
    )
    assert.ok(updatedRows)
    assert.deepEqual(
      updatedRows.slice(0, 2).map((row) => row.id),
      initialRows.map((row) => row.id)
    )
    assert.notEqual(updatedRows[2].id, initialRows[1].id)
  })

  test('does not replace visual rows for invalid JSON', () => {
    assert.equal(parseModelMappingRows('{invalid'), null)
  })
})
