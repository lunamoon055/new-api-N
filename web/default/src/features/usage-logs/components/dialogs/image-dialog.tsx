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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '@/lib/api'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

interface ImageDialogProps {
  imageUrls: string[]
  taskId?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ImageDialog({
  imageUrls,
  taskId,
  open,
  onOpenChange,
}: ImageDialogProps) {
  const { t } = useTranslation()

  const handleOpenChange = (newOpen: boolean) => {
    onOpenChange(newOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-5xl'>
        <DialogHeader>
          <DialogTitle>{t('Image Preview')}</DialogTitle>
          <DialogDescription>
            {taskId
              ? `${t('Task ID:')} ${taskId}`
              : t('View the generated image')}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className='max-h-[600px]'>
          <div
            className={cn(
              'grid gap-4 py-4',
              imageUrls.length > 1 && 'md:grid-cols-2'
            )}
          >
            {imageUrls.map((imageUrl, index) => (
              <ImagePreviewItem
                key={`${imageUrl}-${index}`}
                imageUrl={imageUrl}
              />
            ))}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

function ImagePreviewItem({ imageUrl }: { imageUrl: string }) {
  const { t } = useTranslation()
  const isLocalResult = imageUrl.startsWith('/api/image-results/')
  const [isLoading, setIsLoading] = useState(true)
  const [hasError, setHasError] = useState(false)
  const [displayUrl, setDisplayUrl] = useState(isLocalResult ? '' : imageUrl)

  useEffect(() => {
    if (!isLocalResult) return

    let active = true
    let objectUrl = ''
    void api
      .get(imageUrl, {
        responseType: 'blob',
        skipBusinessError: true,
        skipErrorHandler: true,
      } as Record<string, unknown>)
      .then((response) => {
        if (!active) return
        objectUrl = URL.createObjectURL(response.data as Blob)
        setDisplayUrl(objectUrl)
      })
      .catch(() => {
        if (!active) return
        setIsLoading(false)
        setHasError(true)
      })

    return () => {
      active = false
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [imageUrl, isLocalResult])

  return (
    <div>
      <div className='bg-muted/50 relative flex min-h-[260px] items-center justify-center overflow-hidden rounded-lg border'>
        {(isLoading || hasError) && (
          <Skeleton className='absolute inset-0 h-full w-full rounded-lg' />
        )}
        {displayUrl && (
          <img
            src={displayUrl}
            alt={t('Generated image')}
            className={cn(
              'max-h-[520px] w-full rounded-lg object-contain transition-opacity',
              isLoading || hasError ? 'opacity-0' : 'opacity-100'
            )}
            onLoad={() => {
              setIsLoading(false)
              setHasError(false)
            }}
            onError={() => {
              setIsLoading(false)
              setHasError(true)
            }}
            loading='lazy'
          />
        )}
        {hasError && (
          <div className='absolute inset-0 flex items-center justify-center'>
            <p className='text-muted-foreground text-sm'>
              {t('Failed to load image')}
            </p>
          </div>
        )}
      </div>
      <div className='bg-muted mt-2 rounded-md p-3'>
        <p className='text-muted-foreground font-mono text-xs break-all'>
          {imageUrl}
        </p>
      </div>
    </div>
  )
}
