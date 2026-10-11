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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Field,
  FieldGroup,
  FieldLabel,
  FieldDescription,
} from '@/components/ui/field'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { uploadCreationReferenceFile } from '@/features/creation-center/api'
import {
  getPlaygroundIntegration,
  resolveIntegrationChoice,
  parsePlaygroundReferenceURLs,
} from '../lib/model-integration'
import type { PlaygroundConfig, PlaygroundModelInfo } from '../types'

type MediaFieldsProps = {
  config: PlaygroundConfig
  modelInfo?: PlaygroundModelInfo
  mode: 'image' | 'video'
  disabled?: boolean
  onChange: (value: Partial<PlaygroundConfig>) => void
  onUploadingChange: (value: boolean) => void
}

export function PlaygroundMediaFields(props: MediaFieldsProps) {
  const { t } = useTranslation()
  const [uploading, setUploading] = useState(false)
  const parameters = getPlaygroundIntegration(props.modelInfo)?.parameters
  if (!parameters) return null
  const disabled = props.disabled || uploading
  const references = [
    {
      kind: 'image' as const,
      key: 'referenceImages' as const,
      max: parameters.max_images,
      label: t('Reference images'),
      accept: 'image/*',
    },
    {
      kind: 'video' as const,
      key: 'referenceVideos' as const,
      max: parameters.max_videos,
      label: t('Reference videos'),
      accept: 'video/*',
    },
    {
      kind: 'audio' as const,
      key: 'referenceAudios' as const,
      max: parameters.max_audios,
      label: t('Reference audio'),
      accept: 'audio/*',
    },
  ].filter((reference) => reference.max > 0 || props.config[reference.key])

  async function upload(
    files: FileList | null,
    reference: (typeof references)[number]
  ) {
    if (!files?.length) return
    try {
      const existing = parsePlaygroundReferenceURLs(props.config[reference.key])
      if (existing.length + files.length > reference.max) {
        throw new Error(
          t(
            'Too many references for this model: {{kind}} (maximum {{count}})',
            { kind: reference.label, count: reference.max }
          )
        )
      }
      setUploading(true)
      props.onUploadingChange(true)
      // Save each successful upload so a later failed file can be retried.
      for (const file of Array.from(files)) {
        existing.push(await uploadCreationReferenceFile(file, reference.kind))
        props.onChange({ [reference.key]: existing.join('\n') })
      }
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Unable to upload reference file.')
      )
    } finally {
      setUploading(false)
      props.onUploadingChange(false)
    }
  }

  return (
    <FieldGroup className='gap-3 px-1'>
      {props.mode === 'image' && (
        <div className='flex flex-wrap gap-3'>
          {[
            {
              key: 'imageAspectRatio' as const,
              choices: parameters.ratios,
              label: t('Aspect ratio'),
            },
            {
              key: 'imageResolution' as const,
              choices: parameters.resolutions,
              label: t('Resolution'),
            },
          ].map((field) =>
            field.choices?.length ? (
              <Field key={field.key} className='w-auto min-w-28 gap-1.5'>
                <FieldLabel htmlFor={`playground-${field.key}`}>
                  {field.label}
                </FieldLabel>
                <Select
                  value={resolveIntegrationChoice(
                    props.config[field.key],
                    field.choices
                  )}
                  onValueChange={(value) => {
                    if (value) props.onChange({ [field.key]: value })
                  }}
                  disabled={disabled || field.choices.length === 1}
                >
                  <SelectTrigger id={`playground-${field.key}`}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {field.choices.map((value) => (
                        <SelectItem key={value} value={value}>
                          {value}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
            ) : null
          )}
        </div>
      )}
      {!!references.length && (
        <details>
          <summary className='cursor-pointer text-sm'>
            {t('Reference assets')}{' '}
            {parameters.min_images ? `· ${t('Reference image required')}` : ''}
          </summary>
          <FieldGroup className='mt-3 gap-3 md:grid md:grid-cols-3'>
            {references.map((reference) => (
              <Field key={reference.key}>
                <FieldLabel htmlFor={`playground-${reference.key}`}>
                  {reference.label} ·{' '}
                  {t('Maximum {{count}}', { count: reference.max })}
                </FieldLabel>
                <Textarea
                  id={`playground-${reference.key}`}
                  value={props.config[reference.key]}
                  onChange={(event) =>
                    props.onChange({ [reference.key]: event.target.value })
                  }
                  disabled={disabled}
                  placeholder={t('One public URL per line')}
                  className='min-h-16 text-xs'
                />
                <FieldLabel
                  htmlFor={`playground-upload-${reference.kind}`}
                  className='text-xs'
                >
                  {t('Upload file')}
                </FieldLabel>
                <Input
                  id={`playground-upload-${reference.kind}`}
                  type='file'
                  accept={reference.accept}
                  multiple
                  disabled={disabled || reference.max === 0}
                  onChange={(event) => {
                    void upload(event.target.files, reference)
                    event.target.value = ''
                  }}
                  className='text-xs'
                />
              </Field>
            ))}
          </FieldGroup>
          <FieldDescription className='mt-2'>
            {t(
              'Reference order matches image, video and audio numbers in your prompt.'
            )}
          </FieldDescription>
          {uploading && (
            <p role='status' className='text-muted-foreground text-xs'>
              {t('Uploading reference assets...')}
            </p>
          )}
        </details>
      )}
    </FieldGroup>
  )
}
