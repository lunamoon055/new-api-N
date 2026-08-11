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
import { Copy01Icon, Tick02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'

interface FailReasonDialogProps {
  failReason: string
  rawFailReason?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function FailReasonDialog({
  failReason,
  rawFailReason,
  open,
  onOpenChange,
}: FailReasonDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Fail Reason Details')}</DialogTitle>
          <DialogDescription>
            {rawFailReason
              ? t('Compare the downstream message with the upstream raw error')
              : t('View the complete error message and details')}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className='max-h-[500px] pr-4'>
          <div className='flex flex-col gap-4 py-4'>
            <div className='flex flex-col gap-2'>
              <h3 className='text-sm font-semibold'>
                {rawFailReason ? t('Downstream Message') : t('Error Message')}
              </h3>
              <div className='bg-muted/50 relative rounded-md border p-3'>
                <Button
                  variant='ghost'
                  size='icon'
                  className='absolute top-2 right-2'
                  onClick={() => copyToClipboard(failReason)}
                  title={t('Copy to clipboard')}
                  aria-label={t('Copy to clipboard')}
                >
                  {copiedText === failReason ? (
                    <HugeiconsIcon
                      icon={Tick02Icon}
                      strokeWidth={2}
                      className='text-primary'
                    />
                  ) : (
                    <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} />
                  )}
                </Button>
                <p className='overflow-wrap-anywhere pr-10 text-sm leading-relaxed break-all whitespace-pre-wrap'>
                  {failReason || '-'}
                </p>
              </div>
            </div>

            {rawFailReason && (
              <div className='flex flex-col gap-2'>
                <div className='flex flex-col gap-0.5'>
                  <h3 className='text-sm font-semibold'>
                    {t('Upstream Raw Error')}
                  </h3>
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Only super administrators can view this error; sensitive values are automatically masked'
                    )}
                  </p>
                </div>
                <div className='border-destructive/20 bg-destructive/5 relative rounded-md border p-3'>
                  <Button
                    variant='ghost'
                    size='icon'
                    className='absolute top-2 right-2'
                    onClick={() => copyToClipboard(rawFailReason)}
                    title={t('Copy to clipboard')}
                    aria-label={t('Copy to clipboard')}
                  >
                    {copiedText === rawFailReason ? (
                      <HugeiconsIcon
                        icon={Tick02Icon}
                        strokeWidth={2}
                        className='text-primary'
                      />
                    ) : (
                      <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} />
                    )}
                  </Button>
                  <p className='text-destructive overflow-wrap-anywhere pr-10 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap'>
                    {rawFailReason}
                  </p>
                </div>
              </div>
            )}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
