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
import type { CreationMode, CreationModel, CreationModelGroup } from './types'

export type CreationCategoryRow = CreationModel & {
  mode: CreationMode
}

export function getCreationCategoryRows(groups: CreationModelGroup[]) {
  const rows = new Map<string, CreationCategoryRow>()
  for (const group of groups) {
    for (const model of group.models) {
      const key = model.id.toLowerCase()
      if (rows.has(key)) continue
      rows.set(key, { ...model, mode: group.mode })
    }
  }
  return [...rows.values()].sort((a, b) => a.id.localeCompare(b.id))
}

export function getCreationModeLabel(
  mode: CreationMode,
  t: (key: string) => string
) {
  switch (mode) {
    case 'chat':
      return t('Chat')
    case 'image':
      return t('Image')
    case 'video':
      return t('Video')
  }
}
