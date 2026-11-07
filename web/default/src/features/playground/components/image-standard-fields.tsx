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
import { useTranslation } from 'react-i18next'
import { useEffect } from 'react'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { IntegrationParameters } from '@/features/pricing/types'
import type { PlaygroundConfig } from '../types'

export function ImageStandardFields({
  options,
  config,
  disabled,
  onChange,
}: {
  options: NonNullable<IntegrationParameters['image_options']>
  config: PlaygroundConfig
  disabled?: boolean
  onChange: (value: Partial<PlaygroundConfig>) => void
}) {
  const { t } = useTranslation()
  useEffect(() => {
    const changes: Partial<PlaygroundConfig> = {}
    if (config.imageQuality && !options.qualities.includes(config.imageQuality))
      changes.imageQuality = ''
    if (
      config.imageOutputFormat &&
      !options.output_formats.includes(config.imageOutputFormat)
    )
      changes.imageOutputFormat = ''
    if (
      config.imageBackground &&
      !options.backgrounds.includes(config.imageBackground)
    )
      changes.imageBackground = ''
    if (Object.keys(changes).length) onChange(changes)
  }, [
    config.imageQuality,
    config.imageOutputFormat,
    config.imageBackground,
    options,
    onChange,
  ])
  const selects = [
    {
      key: 'imageQuality' as const,
      label: t('Image quality'),
      choices: options.qualities,
    },
    {
      key: 'imageOutputFormat' as const,
      label: t('Output format'),
      choices: options.output_formats,
    },
    {
      key: 'imageBackground' as const,
      label: t('Background'),
      choices: options.backgrounds,
    },
  ]
  return (
    <FieldGroup className='flex flex-row flex-wrap gap-3'>
      {selects.map((field) => (
        <Field key={field.key} className='w-auto min-w-28 gap-1.5'>
          <FieldLabel htmlFor={`playground-${field.key}`}>
            {field.label}
          </FieldLabel>
          <Select
            value={config[field.key] || 'default'}
            onValueChange={(value) =>
              onChange({
                [field.key]: value === 'default' ? '' : (value ?? ''),
              })
            }
            disabled={disabled}
          >
            <SelectTrigger id={`playground-${field.key}`}>
              <SelectValue>{config[field.key] || t('Default')}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value='default'>{t('Default')}</SelectItem>
                {field.choices.map((value) => (
                  <SelectItem key={value} value={value}>
                    {value}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>
      ))}
      <Field className='w-28 gap-1.5'>
        <FieldLabel htmlFor='playground-image-count'>
          {t('Image count')}
        </FieldLabel>
        <Input
          id='playground-image-count'
          type='number'
          min={1}
          max={options.max_outputs}
          value={config.imageCount ?? 1}
          disabled={disabled}
          onChange={(event) =>
            onChange({ imageCount: Number(event.target.value) })
          }
        />
      </Field>
      <Field className='w-44 gap-1.5'>
        <FieldLabel htmlFor='playground-image-size'>
          {t('Custom image size')}
        </FieldLabel>
        <Input
          id='playground-image-size'
          placeholder='auto / 1536x864'
          value={config.imageSize ?? ''}
          disabled={disabled}
          onChange={(event) => onChange({ imageSize: event.target.value })}
        />
      </Field>
      {['jpeg', 'webp'].includes(config.imageOutputFormat ?? '') && (
        <Field className='w-36 gap-1.5'>
          <FieldLabel htmlFor='playground-image-compression'>
            {t('Output compression')}
          </FieldLabel>
          <Input
            id='playground-image-compression'
            type='number'
            min={0}
            max={100}
            placeholder='0–100'
            value={config.imageOutputCompression ?? ''}
            disabled={disabled}
            onChange={(event) =>
              onChange({
                imageOutputCompression:
                  event.target.value === '' ? '' : Number(event.target.value),
              })
            }
          />
        </Field>
      )}
    </FieldGroup>
  )
}
