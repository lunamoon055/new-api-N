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
import {
  Copy01Icon,
  ExternalLinkIcon,
  Image02Icon,
  MusicNote01Icon,
  Tick02Icon,
  Video01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { Button, buttonVariants } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from '@/components/ui/item'
import { ScrollArea } from '@/components/ui/scroll-area'
import type {
  TaskInputMaterial,
  TaskInputMaterialKind,
} from '../../lib/task-preview'
import type { TaskRequestSnapshot } from '../../types'

interface PromptDialogProps {
  prompt: string
  promptEn?: string
  inputMaterials?: TaskInputMaterial[]
  requestBody?: string
  requestSnapshot?: TaskRequestSnapshot
  requestBodyNotice?: 'historical' | 'restricted'
  open: boolean
  onOpenChange: (open: boolean) => void
}

const MATERIAL_LABELS: Record<TaskInputMaterialKind, string> = {
  image: 'Image',
  video: 'Video',
  audio: 'Audio',
}

function MaterialIcon({ kind }: { kind: TaskInputMaterialKind }) {
  if (kind === 'image') {
    return <HugeiconsIcon icon={Image02Icon} strokeWidth={2} />
  }
  if (kind === 'video') {
    return <HugeiconsIcon icon={Video01Icon} strokeWidth={2} />
  }
  return <HugeiconsIcon icon={MusicNote01Icon} strokeWidth={2} />
}

function MaterialPreview({ material }: { material: TaskInputMaterial }) {
  const { t } = useTranslation()
  const label = t(MATERIAL_LABELS[material.kind])

  if (material.kind === 'image') {
    return (
      <img
        src={material.url}
        alt={label}
        loading='lazy'
        className='max-h-64 w-full rounded-md object-contain'
      />
    )
  }
  if (material.kind === 'video') {
    return (
      <video
        src={material.url}
        controls
        preload='metadata'
        className='max-h-64 w-full rounded-md'
      />
    )
  }
  return (
    <audio src={material.url} controls preload='metadata' className='w-full' />
  )
}

export function PromptDialog({
  prompt,
  promptEn,
  inputMaterials = [],
  requestBody,
  requestSnapshot,
  requestBodyNotice,
  open,
  onOpenChange,
}: PromptDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const requestMode = requestBody !== undefined
  let description = t('View the complete prompt and its English translation')
  if (inputMaterials.length > 0)
    description = t('View the complete details for this log entry')
  if (requestMode)
    description = t('Incoming request body before upstream model conversion.')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={requestMode ? 'sm:max-w-4xl' : 'sm:max-w-2xl'}>
        <DialogHeader>
          <DialogTitle>
            {requestMode ? t('Request Body') : t('Prompt Details')}
          </DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <ScrollArea className='max-h-[65vh] pr-4'>
          <div className='flex flex-col gap-4 py-4'>
            {requestMode && (
              <div className='flex min-w-0 flex-col gap-3'>
                {requestSnapshot && (
                  <div className='flex flex-wrap items-center gap-2 text-xs'>
                    <code className='bg-muted rounded px-2 py-1'>
                      {requestSnapshot.method} {requestSnapshot.path}
                    </code>
                    <span className='text-muted-foreground'>
                      {requestSnapshot.content_type}
                    </span>
                  </div>
                )}
                {(requestBodyNotice ||
                  requestSnapshot?.omitted ||
                  requestSnapshot?.redacted ||
                  requestSnapshot?.format === 'multipart') && (
                  <Alert>
                    <AlertDescription className='flex flex-col gap-1'>
                      {requestBodyNotice === 'historical' && (
                        <p>
                          {t(
                            'This historical record only contains saved fields. The original request body was not recorded.'
                          )}
                        </p>
                      )}
                      {requestBodyNotice === 'restricted' && (
                        <p>
                          {t(
                            'Full request bodies are available to super administrators only. Only visible fields are shown here.'
                          )}
                        </p>
                      )}
                      {requestSnapshot?.redacted && (
                        <p>
                          {t(
                            'Authentication fields and signed URL credentials have been redacted.'
                          )}
                        </p>
                      )}
                      {requestSnapshot?.omitted && (
                        <p>
                          {t(
                            'Some content was omitted because it was oversized, inline media, or unavailable. This snapshot is incomplete.'
                          )}
                        </p>
                      )}
                      {requestSnapshot?.format === 'multipart' && (
                        <p>
                          {t(
                            'Multipart requests show form fields and file metadata. File contents are not recorded.'
                          )}
                        </p>
                      )}
                    </AlertDescription>
                  </Alert>
                )}
                {requestBody && (
                  <>
                    <div className='flex justify-end'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => copyToClipboard(requestBody)}
                      >
                        <HugeiconsIcon
                          icon={
                            copiedText === requestBody ? Tick02Icon : Copy01Icon
                          }
                          data-icon='inline-start'
                        />
                        {t('Copy request body')}
                      </Button>
                    </div>
                    <pre className='bg-muted/50 max-w-full overflow-x-auto rounded-md border p-4 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap'>
                      {requestBody}
                    </pre>
                  </>
                )}
              </div>
            )}
            {/* Original Prompt */}
            {!requestMode && prompt && (
              <div className='flex flex-col gap-2'>
                <h3 className='text-sm font-semibold'>{t('Prompt')}</h3>
                <div className='bg-muted/50 relative rounded-md border p-3'>
                  <Button
                    variant='ghost'
                    size='icon'
                    className='absolute top-2 right-2'
                    onClick={() => copyToClipboard(prompt)}
                    title={t('Copy to clipboard')}
                    aria-label={t('Copy to clipboard')}
                  >
                    {copiedText === prompt ? (
                      <HugeiconsIcon
                        icon={Tick02Icon}
                        strokeWidth={2}
                        className='text-primary'
                      />
                    ) : (
                      <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} />
                    )}
                  </Button>
                  <p className='pr-10 text-sm leading-relaxed break-words whitespace-pre-wrap'>
                    {prompt}
                  </p>
                </div>
              </div>
            )}

            {/* English Prompt */}
            {promptEn && (
              <div className='flex flex-col gap-2'>
                <h3 className='text-sm font-semibold'>{t('Prompt (EN)')}</h3>
                <div className='bg-muted/50 relative rounded-md border p-3'>
                  <Button
                    variant='ghost'
                    size='icon'
                    className='absolute top-2 right-2'
                    onClick={() => copyToClipboard(promptEn)}
                    title={t('Copy to clipboard')}
                    aria-label={t('Copy to clipboard')}
                  >
                    {copiedText === promptEn ? (
                      <HugeiconsIcon
                        icon={Tick02Icon}
                        strokeWidth={2}
                        className='text-primary'
                      />
                    ) : (
                      <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} />
                    )}
                  </Button>
                  <p className='pr-10 text-sm leading-relaxed break-words whitespace-pre-wrap'>
                    {promptEn}
                  </p>
                </div>
              </div>
            )}

            {inputMaterials.length > 0 && (
              <div className='flex flex-col gap-2'>
                <h3 className='text-sm font-semibold'>
                  {t('Reference assets')}
                </h3>
                <ItemGroup className='gap-3'>
                  {inputMaterials.map((material, index) => (
                    <Item
                      key={`${material.kind}:${material.url}`}
                      variant='outline'
                      className='items-start'
                    >
                      <ItemMedia variant='icon'>
                        <MaterialIcon kind={material.kind} />
                      </ItemMedia>
                      <ItemContent className='min-w-0'>
                        <ItemTitle>
                          {t(MATERIAL_LABELS[material.kind])} {index + 1}
                        </ItemTitle>
                        <ItemDescription className='line-clamp-none break-all'>
                          <a
                            href={material.url}
                            target='_blank'
                            rel='noreferrer'
                          >
                            {material.url}
                          </a>
                        </ItemDescription>
                      </ItemContent>
                      <ItemActions>
                        <Button
                          variant='ghost'
                          size='icon-sm'
                          onClick={() => copyToClipboard(material.url)}
                          title={t('Copy Link')}
                          aria-label={t('Copy Link')}
                        >
                          {copiedText === material.url ? (
                            <HugeiconsIcon
                              icon={Tick02Icon}
                              strokeWidth={2}
                              className='text-primary'
                            />
                          ) : (
                            <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} />
                          )}
                        </Button>
                        <a
                          href={material.url}
                          target='_blank'
                          rel='noreferrer'
                          className={buttonVariants({
                            variant: 'ghost',
                            size: 'icon-sm',
                          })}
                          title={t('View')}
                          aria-label={t('View')}
                        >
                          <HugeiconsIcon
                            icon={ExternalLinkIcon}
                            strokeWidth={2}
                          />
                        </a>
                      </ItemActions>
                      <ItemFooter className='bg-muted/30 overflow-hidden rounded-md border p-2'>
                        <MaterialPreview material={material} />
                      </ItemFooter>
                    </Item>
                  ))}
                </ItemGroup>
              </div>
            )}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
