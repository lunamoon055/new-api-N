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
import { FileImage, Trash2, Upload } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldGroup } from '@/components/ui/field'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  CREATION_IMAGE_REFERENCE_MAX_COUNT,
  getCreationReferencePreviewURL,
  getCreationReferenceURL,
  type CreationImageReferences,
} from './session'

const IMAGE_REFERENCE_ACCEPT =
  'image/avif,image/gif,image/jpeg,image/png,image/webp,.avif,.gif,.jpeg,.jpg,.png,.webp'

type ImageReferenceFieldsProps = {
  value: CreationImageReferences
  onFilesSelected: (files: File[]) => void
  onRemoveImage: (index: number) => void
}

type ImagePreview = {
  url: string
  title: string
}

export function ImageReferenceFields(props: ImageReferenceFieldsProps) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<ImagePreview | null>(null)
  const imageReferences = props.value.imageUrls.filter((reference) =>
    getCreationReferenceURL(reference)
  )
  const uploadDisabled =
    imageReferences.length >= CREATION_IMAGE_REFERENCE_MAX_COUNT

  return (
    <TooltipProvider>
      <FieldGroup className='mt-3 gap-2'>
        <p className='text-muted-foreground text-[11px] leading-4'>
          {t('Gpt-image2 image reference upload tip')}
        </p>
        <Field>
          <div className='flex flex-wrap items-center gap-2'>
            <label
              className='border-input hover:bg-muted inline-flex h-9 cursor-pointer items-center justify-center gap-1.5 rounded-lg border bg-transparent px-3 text-sm font-medium whitespace-nowrap transition-colors data-[disabled=true]:pointer-events-none data-[disabled=true]:opacity-50'
              data-disabled={uploadDisabled ? 'true' : undefined}
            >
              <Upload data-icon='inline-start' />
              {t('Reference images')}
              <input
                type='file'
                accept={IMAGE_REFERENCE_ACCEPT}
                multiple
                disabled={uploadDisabled}
                className='sr-only'
                onChange={(event) => {
                  props.onFilesSelected(
                    event.currentTarget.files
                      ? Array.from(event.currentTarget.files)
                      : []
                  )
                  event.currentTarget.value = ''
                }}
              />
            </label>
            <span className='text-muted-foreground text-xs'>
              {imageReferences.length
                ? t('{{count}} reference image(s)', {
                    count: imageReferences.length,
                  })
                : t('No reference images')}
            </span>
          </div>
        </Field>
        {!!imageReferences.length && (
          <div className='flex max-w-full flex-wrap gap-1.5'>
            {imageReferences.map((reference, index) => (
              <span
                key={`${getCreationReferenceURL(reference)}-${index}`}
                className='bg-muted text-muted-foreground inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-1 text-[11px]'
              >
                <button
                  type='button'
                  className='hover:text-foreground inline-flex min-w-0 items-center gap-1.5 transition-colors'
                  aria-label={`${t('Open reference preview')}: ${t(
                    'Reference image'
                  )} ${index + 1}`}
                  onClick={() =>
                    setPreview({
                      url: getCreationReferencePreviewURL(reference),
                      title: `${t('Reference image')} ${index + 1}`,
                    })
                  }
                >
                  <FileImage className='size-3 shrink-0' />
                  <span className='max-w-28 truncate'>
                    {t('Reference image')} {index + 1}
                  </span>
                </button>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        type='button'
                        size='icon-xs'
                        variant='ghost'
                        aria-label={`${t('Remove reference image')} ${
                          index + 1
                        }`}
                        onClick={() => props.onRemoveImage(index)}
                      />
                    }
                  >
                    <Trash2 data-icon='inline-start' />
                  </TooltipTrigger>
                  <TooltipContent>
                    {t('Remove reference image')} {index + 1}
                  </TooltipContent>
                </Tooltip>
              </span>
            ))}
          </div>
        )}
      </FieldGroup>
      <Dialog
        open={!!preview}
        onOpenChange={(open) => {
          if (!open) setPreview(null)
        }}
      >
        <DialogContent className='sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>
              {preview?.title ?? t('Reference preview')}
            </DialogTitle>
          </DialogHeader>
          {preview && (
            <img
              src={preview.url}
              alt={preview.title}
              className='max-h-[70vh] w-full rounded-md object-contain'
            />
          )}
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  )
}
