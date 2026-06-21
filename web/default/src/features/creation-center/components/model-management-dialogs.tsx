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
import { FileText, RefreshCw, RotateCcw, Settings2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import {
  getCreationModeLabel,
  type CreationCategoryRow,
} from '../category-rows'
import { CREATION_MODES } from '../constants'
import type {
  CreationMode,
  CreationModelCategories,
  CreationModelDescriptions,
} from '../types'

type ModelCategoryDialogProps = {
  open: boolean
  models: CreationCategoryRow[]
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSave: (categories: CreationModelCategories) => void
  onReset: () => void
}

export function ModelCategoryDialog(props: ModelCategoryDialogProps) {
  const { t } = useTranslation()
  const save = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const formData = new FormData(event.currentTarget)
    const categories = props.models.reduce<CreationModelCategories>(
      (next, model) => {
        const value = formData.get(model.id)
        next[model.id] =
          typeof value === 'string' &&
          CREATION_MODES.includes(value as CreationMode)
            ? (value as CreationMode)
            : model.mode
        return next
      },
      {}
    )
    props.onSave(categories)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-w-3xl'>
        <form onSubmit={save}>
          <DialogHeader>
            <DialogTitle>{t('Creation model category management')}</DialogTitle>
            <DialogDescription>
              {t(
                'Manually assign visible creation models to chat, image, or video filters.'
              )}
            </DialogDescription>
          </DialogHeader>

          <div className='mt-4 rounded-lg border'>
            <div className='bg-muted/40 grid grid-cols-[minmax(0,1fr)_8.5rem] gap-3 border-b px-3 py-2 text-xs font-medium'>
              <span>{t('Model')}</span>
              <span>{t('Category')}</span>
            </div>
            <div className='max-h-[min(28rem,60svh)] overflow-auto'>
              {props.models.length === 0 ? (
                <div className='text-muted-foreground px-3 py-8 text-center text-sm'>
                  {t('No creation models available.')}
                </div>
              ) : (
                <div className='divide-y'>
                  {props.models.map((model) => (
                    <div
                      key={model.id}
                      className='grid grid-cols-[minmax(0,1fr)_8.5rem] items-center gap-3 px-3 py-3'
                    >
                      <div className='min-w-0'>
                        <div className='truncate text-sm font-medium'>
                          {model.id}
                        </div>
                        <div className='text-muted-foreground mt-1 line-clamp-1 text-xs'>
                          {model.description || t('Ready for creation tasks.')}
                        </div>
                      </div>
                      <NativeSelect
                        size='sm'
                        className='w-full'
                        aria-label={t('Category')}
                        name={model.id}
                        defaultValue={model.mode}
                        disabled={props.saving}
                      >
                        {CREATION_MODES.map((item) => (
                          <NativeSelectOption key={item} value={item}>
                            {getCreationModeLabel(item, t)}
                          </NativeSelectOption>
                        ))}
                      </NativeSelect>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <DialogFooter className='mt-4 sm:justify-between'>
            <Button
              type='button'
              variant='outline'
              onClick={props.onReset}
              disabled={props.saving}
            >
              <RotateCcw className='size-4' />
              {t('Reset to auto')}
            </Button>
            <Button
              type='submit'
              disabled={props.saving || props.models.length === 0}
            >
              {props.saving ? (
                <RefreshCw className='size-4 animate-spin' />
              ) : (
                <Settings2 className='size-4' />
              )}
              {t('Save categories')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

type ModelDescriptionDialogProps = {
  open: boolean
  models: CreationCategoryRow[]
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSave: (descriptions: CreationModelDescriptions) => void
  onReset: () => void
}

export function ModelDescriptionDialog(props: ModelDescriptionDialogProps) {
  const { t } = useTranslation()
  const save = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const formData = new FormData(event.currentTarget)
    const descriptions = props.models.reduce<CreationModelDescriptions>(
      (next, model) => {
        const value = formData.get(model.id)
        const description = typeof value === 'string' ? value.trim() : ''
        if (description) {
          next[model.id] = description
        }
        return next
      },
      {}
    )
    props.onSave(descriptions)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-w-4xl'>
        <form onSubmit={save}>
          <DialogHeader>
            <DialogTitle>
              {t('Creation model description management')}
            </DialogTitle>
            <DialogDescription>
              {t(
                'Manually write descriptions for visible creation models. Blank fields keep automatic descriptions.'
              )}
            </DialogDescription>
          </DialogHeader>

          <div className='mt-4 rounded-lg border'>
            <div className='bg-muted/40 grid grid-cols-[minmax(0,0.75fr)_minmax(0,1.25fr)] gap-3 border-b px-3 py-2 text-xs font-medium'>
              <span>{t('Model')}</span>
              <span>{t('Description')}</span>
            </div>
            <div className='max-h-[min(30rem,60svh)] overflow-auto'>
              {props.models.length === 0 ? (
                <div className='text-muted-foreground px-3 py-8 text-center text-sm'>
                  {t('No creation models available.')}
                </div>
              ) : (
                <div className='divide-y'>
                  {props.models.map((model) => (
                    <div
                      key={model.id}
                      className='grid grid-cols-[minmax(0,0.75fr)_minmax(0,1.25fr)] gap-3 px-3 py-3'
                    >
                      <div className='min-w-0'>
                        <div className='truncate text-sm font-medium'>
                          {model.id}
                        </div>
                        <div className='text-muted-foreground mt-1 flex flex-wrap gap-1 text-xs'>
                          <Badge variant='secondary'>
                            {getCreationModeLabel(model.mode, t)}
                          </Badge>
                          {model.manual_description && (
                            <Badge variant='outline'>
                              {t('Manual description')}
                            </Badge>
                          )}
                        </div>
                      </div>
                      <Textarea
                        name={model.id}
                        defaultValue={model.manual_description ?? ''}
                        placeholder={
                          model.description || t('No description yet.')
                        }
                        rows={3}
                        disabled={props.saving}
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <DialogFooter className='mt-4 sm:justify-between'>
            <Button
              type='button'
              variant='outline'
              onClick={props.onReset}
              disabled={props.saving}
            >
              <RotateCcw className='size-4' />
              {t('Reset to auto')}
            </Button>
            <Button
              type='submit'
              disabled={props.saving || props.models.length === 0}
            >
              {props.saving ? (
                <RefreshCw className='size-4 animate-spin' />
              ) : (
                <FileText className='size-4' />
              )}
              {t('Save descriptions')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
