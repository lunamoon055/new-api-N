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
import { Film } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'

interface VideoPreviewDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  videoUrl: string
  taskId?: string
}

export function VideoPreviewDialog(props: VideoPreviewDialogProps) {
  const { t } = useTranslation()
  const [isLoading, setIsLoading] = useState(true)
  const [hasError, setHasError] = useState(false)

  const handleOpenChange = (open: boolean) => {
    if (open) {
      setIsLoading(true)
      setHasError(false)
    }
    props.onOpenChange(open)
  }

  return (
    <Dialog open={props.open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-4xl'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <Film className='size-5' />
            {t('Video Preview')}
          </DialogTitle>
          <DialogDescription>
            {props.taskId
              ? `${t('Task ID:')} ${props.taskId}`
              : t('View the generated video')}
          </DialogDescription>
        </DialogHeader>

        <div className='py-2'>
          <div className='bg-muted/50 relative flex min-h-[320px] items-center justify-center overflow-hidden rounded-lg border'>
            {(isLoading || hasError) && (
              <Skeleton className='absolute inset-0 h-full w-full rounded-lg' />
            )}
            <video
              controls
              playsInline
              preload='metadata'
              src={props.videoUrl}
              className={`max-h-[70vh] w-full bg-black object-contain ${
                isLoading || hasError ? 'opacity-0' : 'opacity-100'
              }`}
              onLoadedMetadata={() => {
                setIsLoading(false)
                setHasError(false)
              }}
              onCanPlay={() => {
                setIsLoading(false)
                setHasError(false)
              }}
              onError={() => {
                setIsLoading(false)
                setHasError(true)
              }}
            />
            {hasError && (
              <div className='absolute inset-0 flex items-center justify-center'>
                <p className='text-muted-foreground text-sm'>
                  {t('Failed to load video')}
                </p>
              </div>
            )}
          </div>
          <div className='bg-muted mt-3 rounded-md p-3'>
            <p className='text-muted-foreground font-mono text-xs break-all'>
              {props.videoUrl}
            </p>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
