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
import type { PricingModel } from '../types'
import type { SupportedParameter } from './mock-stats'

export function buildStandardImageParameters(
  model: PricingModel
): SupportedParameter[] | undefined {
  const parameters = model.mapped_endpoints?.find(
    (endpoint) => endpoint.parameters?.image_options
  )?.parameters
  const options = parameters?.image_options
  if (!options) return undefined
  const references =
    model.mapped_endpoints?.find(
      (endpoint) => endpoint.path === '/v1/images/edits'
    )?.parameters?.max_images ?? options.max_references
  return [
    {
      name: 'model',
      type: 'string',
      required: true,
      enumValues: [model.model_name],
      descriptionKey: 'Model name',
    },
    {
      name: 'prompt',
      type: 'string',
      required: true,
      descriptionKey: 'Text description of the desired image',
    },
    ...(parameters?.resolution_field && parameters.resolutions?.length
      ? [
          {
            name: parameters.resolution_field,
            type: 'string' as const,
            enumValues: parameters.resolutions,
            descriptionKey: 'Resolution',
          },
        ]
      : []),
    ...(parameters?.ratio_field && parameters.ratios?.length
      ? [
          {
            name: parameters.ratio_field,
            type: 'string' as const,
            enumValues: parameters.ratios,
            descriptionKey: 'Aspect ratio',
          },
        ]
      : []),
    {
      name: 'size',
      type: 'string',
      defaultValue: 'auto',
      enumValues: options.sizes,
      descriptionKey: 'Custom image size',
    },
    {
      name: 'quality',
      type: 'string',
      defaultValue: 'auto',
      enumValues: options.qualities,
      descriptionKey: 'Image quality',
    },
    {
      name: 'n',
      type: 'integer',
      defaultValue: 1,
      range: `1 ~ ${options.max_outputs}`,
      descriptionKey: 'Number of images to generate',
    },
    {
      name: 'background',
      type: 'string',
      defaultValue: 'auto',
      enumValues: options.backgrounds,
      descriptionKey: 'Background',
    },
    {
      name: 'output_format',
      type: 'string',
      defaultValue: 'png',
      enumValues: options.output_formats,
      descriptionKey: 'Output format',
    },
    {
      name: 'output_compression',
      type: 'integer',
      range: '0 ~ 100',
      descriptionKey: 'Output compression',
    },
    {
      name: 'image[]',
      type: 'file[]',
      range: `1 ~ ${references}`,
      descriptionKey: 'Reference images',
    },
    { name: 'mask', type: 'file / object', descriptionKey: 'Image edit mask' },
    ...(options.supports_markup
      ? [
          {
            name: 'markup',
            type: 'file / object' as const,
            descriptionKey: 'Image edit markup',
          },
        ]
      : []),
  ]
}
