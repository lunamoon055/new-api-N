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

export type ModelMappingRow = {
  id: string
  from: string
  to: string
}

let modelMappingRowIdCounter = 0

export function createModelMappingRow(from = '', to = ''): ModelMappingRow {
  modelMappingRowIdCounter += 1
  return {
    id: `model_mapping_${modelMappingRowIdCounter}`,
    from,
    to,
  }
}

export function parseModelMappingRows(
  value: string,
  previousRows: ModelMappingRow[] = []
): ModelMappingRow[] | null {
  if (!value.trim()) {
    return []
  }

  try {
    const parsed = JSON.parse(value)
    if (parsed === null || typeof parsed !== 'object') {
      return null
    }

    return Object.entries(parsed).map(([from, to], index) => ({
      id: previousRows[index]?.id ?? createModelMappingRow().id,
      from,
      to: String(to),
    }))
  } catch (_error) {
    return null
  }
}
