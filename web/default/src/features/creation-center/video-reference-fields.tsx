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
import { ChevronDown, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Field, FieldLabel, FieldTitle } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { CreationVideoReferences } from './session'

type VideoReferenceFieldsProps = {
  mode: 'images' | 'all'
  value: CreationVideoReferences
  onChange: (value: CreationVideoReferences) => void
}

export function VideoReferenceFields(props: VideoReferenceFieldsProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  return (
    <TooltipProvider>
      <Collapsible open={open} onOpenChange={setOpen} className='sm:col-span-3'>
        <CollapsibleTrigger className='hover:bg-muted/50 flex h-9 w-full items-center justify-between rounded-lg border px-3 text-sm font-medium transition-colors'>
          <span>{t('Remote reference media')}</span>
          <ChevronDown
            className={cn(
              'text-muted-foreground size-4 transition-transform',
              open && 'rotate-180'
            )}
          />
        </CollapsibleTrigger>
        <CollapsibleContent className='grid gap-3 pt-3 sm:grid-cols-2'>
          <URLListField
            label={t('Image URL')}
            addLabel={t('Add image URL')}
            removeLabel={t('Remove URL')}
            values={props.value.imageUrls}
            onChange={(imageUrls) =>
              props.onChange({ ...props.value, imageUrls })
            }
          />
          {props.mode === 'all' && (
            <>
              <URLField
                id='creation-video-start-image-url'
                label={t('Start frame URL')}
                value={props.value.startImageUrl}
                onChange={(startImageUrl) =>
                  props.onChange({ ...props.value, startImageUrl })
                }
              />
              <URLField
                id='creation-video-end-image-url'
                label={t('End frame URL')}
                value={props.value.endImageUrl}
                onChange={(endImageUrl) =>
                  props.onChange({ ...props.value, endImageUrl })
                }
              />
              <URLListField
                label={t('Video URL')}
                addLabel={t('Add video URL')}
                removeLabel={t('Remove URL')}
                values={props.value.videoUrls}
                onChange={(videoUrls) =>
                  props.onChange({ ...props.value, videoUrls })
                }
              />
              <URLField
                id='creation-video-audio-url'
                label={t('Audio URL')}
                value={props.value.audioUrl}
                onChange={(audioUrl) =>
                  props.onChange({ ...props.value, audioUrl })
                }
              />
            </>
          )}
        </CollapsibleContent>
      </Collapsible>
    </TooltipProvider>
  )
}

function URLField(props: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Field>
      <FieldLabel htmlFor={props.id}>{props.label}</FieldLabel>
      <Input
        id={props.id}
        type='url'
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
        placeholder='https://'
      />
    </Field>
  )
}

function URLListField(props: {
  label: string
  addLabel: string
  removeLabel: string
  values: string[]
  onChange: (values: string[]) => void
}) {
  const values = props.values.length ? props.values : ['']
  const canRemove = props.values.length > 0

  return (
    <Field>
      <div className='flex min-h-7 items-center justify-between gap-2'>
        <FieldTitle>{props.label}</FieldTitle>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type='button'
                size='icon-sm'
                variant='ghost'
                aria-label={props.addLabel}
                onClick={() => props.onChange([...values, ''])}
              />
            }
          >
            <Plus />
          </TooltipTrigger>
          <TooltipContent>{props.addLabel}</TooltipContent>
        </Tooltip>
      </div>
      {values.map((value, index) => (
        <div key={index} className='flex items-center gap-1.5'>
          <Input
            type='url'
            value={value}
            aria-label={`${props.label} ${index + 1}`}
            onChange={(event) =>
              props.onChange(
                values.map((item, itemIndex) =>
                  itemIndex === index ? event.target.value : item
                )
              )
            }
            placeholder='https://'
          />
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type='button'
                  size='icon-sm'
                  variant='ghost'
                  aria-label={props.removeLabel}
                  disabled={!canRemove}
                  onClick={() =>
                    props.onChange(
                      values.filter((_, itemIndex) => itemIndex !== index)
                    )
                  }
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>{props.removeLabel}</TooltipContent>
          </Tooltip>
        </div>
      ))}
    </Field>
  )
}
