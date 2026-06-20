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
import { FileImage, Images, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { CreationVideoReferences } from './session'

type VideoReferenceFieldsProps = {
  value: CreationVideoReferences
  onImagesSelected: (files: File[]) => void
  onRemoveImage: (index: number) => void
}

export function VideoReferenceFields(props: VideoReferenceFieldsProps) {
  const { t } = useTranslation()

  return (
    <TooltipProvider>
      <div className='flex min-w-0 flex-col gap-2'>
        <div className='flex flex-wrap items-center gap-2'>
          <label className='border-input hover:bg-muted inline-flex h-9 cursor-pointer items-center justify-center gap-1.5 rounded-lg border bg-transparent px-3 text-sm font-medium whitespace-nowrap transition-colors'>
            <Images className='size-4' />
            {t('Reference images')}
            <input
              type='file'
              accept='image/*'
              multiple
              className='sr-only'
              onChange={(event) => {
                props.onImagesSelected(
                  event.currentTarget.files
                    ? Array.from(event.currentTarget.files)
                    : []
                )
                event.currentTarget.value = ''
              }}
            />
          </label>
          <span className='text-muted-foreground text-xs'>
            {props.value.imageUrls.length
              ? t('{{count}} reference image(s)', {
                  count: props.value.imageUrls.length,
                })
              : t('No reference images')}
          </span>
        </div>
        {!!props.value.imageUrls.length && (
          <div className='flex max-w-full flex-wrap gap-1.5'>
            {props.value.imageUrls.map((url, index) => (
              <span
                key={`${url}-${index}`}
                className='bg-muted text-muted-foreground inline-flex max-w-full items-center gap-1.5 rounded-md border px-2 py-1 text-[11px]'
              >
                <FileImage className='size-3 shrink-0' />
                <span className='max-w-28 truncate'>
                  {t('Reference image')} {index + 1}
                </span>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        type='button'
                        size='icon-xs'
                        variant='ghost'
                        aria-label={`${t('Remove reference image')} ${index + 1}`}
                        onClick={() => props.onRemoveImage(index)}
                      />
                    }
                  >
                    <Trash2 />
                  </TooltipTrigger>
                  <TooltipContent>{t('Remove reference image')}</TooltipContent>
                </Tooltip>
              </span>
            ))}
          </div>
        )}
      </div>
    </TooltipProvider>
  )
}
