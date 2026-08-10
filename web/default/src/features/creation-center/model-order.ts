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
import { CREATION_MODES } from './constants'
import type {
  CreationMode,
  CreationModel,
  CreationModelGroup,
  CreationModelOrder,
} from './types'

export type CreationModelsByMode = Record<CreationMode, CreationModel[]>

export function getCreationModelsByMode(
  groups: CreationModelGroup[]
): CreationModelsByMode {
  const modelsByMode: CreationModelsByMode = {
    chat: [],
    image: [],
    video: [],
  }
  for (const group of groups) {
    if (!CREATION_MODES.includes(group.mode)) continue
    modelsByMode[group.mode] = [...group.models]
  }
  return modelsByMode
}

export function moveCreationModel(
  models: CreationModel[],
  modelId: string,
  targetIndex: number
): CreationModel[] {
  const sourceIndex = models.findIndex((model) => model.id === modelId)
  if (sourceIndex < 0 || models.length < 2) return models

  const nextIndex = Math.max(0, Math.min(targetIndex, models.length - 1))
  if (sourceIndex === nextIndex) return models

  const next = [...models]
  const [model] = next.splice(sourceIndex, 1)
  next.splice(nextIndex, 0, model)
  return next
}

export function serializeCreationModelOrder(
  modelsByMode: CreationModelsByMode
): CreationModelOrder {
  return CREATION_MODES.reduce<CreationModelOrder>((order, mode) => {
    order[mode] = modelsByMode[mode].map((model) => model.id)
    return order
  }, {})
}
