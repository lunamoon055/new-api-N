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
import {
  ArrowDown01Icon,
  ArrowUp01Icon,
  DragDropVerticalIcon,
  FloppyDiskIcon,
  ReloadIcon,
  Sorting01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { FileText, RefreshCw, RotateCcw, Settings2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
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
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  getCreationModeLabel,
  type CreationCategoryRow,
} from '../category-rows'
import { CREATION_MODES } from '../constants'
import {
  getCreationModelsByMode,
  moveCreationModel,
  serializeCreationModelOrder,
} from '../model-order'
import type {
  CreationMode,
  CreationModelCategories,
  CreationModelDescriptions,
  CreationModelGroup,
  CreationModelOrder,
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

type ModelOrderDialogProps = {
  open: boolean
  groups: CreationModelGroup[]
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSave: (order: CreationModelOrder) => void
  onReset: () => void
}

export function ModelOrderDialog(props: ModelOrderDialogProps) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      {props.open && <ModelOrderEditor {...props} />}
    </Dialog>
  )
}

function ModelOrderEditor(props: ModelOrderDialogProps) {
  const { t } = useTranslation()
  const initialModelsByMode = getCreationModelsByMode(props.groups)
  const [modelsByMode, setModelsByMode] = useState(initialModelsByMode)
  const [activeMode, setActiveMode] = useState<CreationMode>(
    () =>
      CREATION_MODES.find((mode) => initialModelsByMode[mode].length > 0) ??
      'chat'
  )
  const [draggedModelId, setDraggedModelId] = useState<string>()

  const moveModel = (
    mode: CreationMode,
    modelId: string,
    targetIndex: number
  ) => {
    setModelsByMode((current) => ({
      ...current,
      [mode]: moveCreationModel(current[mode], modelId, targetIndex),
    }))
  }

  const save = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    props.onSave(serializeCreationModelOrder(modelsByMode))
  }

  return (
    <DialogContent className='max-w-3xl'>
      <form onSubmit={save}>
        <DialogHeader>
          <DialogTitle>{t('Creation model order management')}</DialogTitle>
          <DialogDescription>
            {t(
              'Adjust the display order of models in each Creation Center category.'
            )}
          </DialogDescription>
        </DialogHeader>

        <Tabs
          className='mt-4 gap-3'
          value={activeMode}
          onValueChange={(value) => {
            if (CREATION_MODES.includes(value as CreationMode)) {
              setActiveMode(value as CreationMode)
              setDraggedModelId(undefined)
            }
          }}
        >
          <TabsList className='grid w-full grid-cols-3'>
            {CREATION_MODES.map((mode) => (
              <TabsTrigger key={mode} value={mode}>
                {getCreationModeLabel(mode, t)}
                <Badge variant='secondary' className='ml-1 tabular-nums'>
                  {modelsByMode[mode].length}
                </Badge>
              </TabsTrigger>
            ))}
          </TabsList>

          {CREATION_MODES.map((mode) => {
            const modeModels = modelsByMode[mode]
            return (
              <TabsContent key={mode} value={mode}>
                <div className='text-muted-foreground mb-2 text-xs'>
                  {t(
                    'Drag model rows or use the arrow buttons to adjust their order.'
                  )}
                </div>
                <ScrollArea className='h-[min(30rem,55svh)] rounded-lg border'>
                  {modeModels.length === 0 ? (
                    <Empty className='min-h-52 border-0'>
                      <EmptyHeader>
                        <EmptyMedia variant='icon'>
                          <HugeiconsIcon icon={Sorting01Icon} strokeWidth={2} />
                        </EmptyMedia>
                        <EmptyTitle>
                          {t('No creation models available.')}
                        </EmptyTitle>
                        <EmptyDescription>
                          {t(
                            'Configure models for this category before sorting.'
                          )}
                        </EmptyDescription>
                      </EmptyHeader>
                    </Empty>
                  ) : (
                    <div className='divide-y p-1'>
                      {modeModels.map((model, index) => (
                        <div
                          key={model.id}
                          draggable={!props.saving}
                          onDragStart={(event) => {
                            setDraggedModelId(model.id)
                            event.dataTransfer.effectAllowed = 'move'
                            event.dataTransfer.setData('text/plain', model.id)
                          }}
                          onDragOver={(event) => {
                            if (draggedModelId && draggedModelId !== model.id) {
                              event.preventDefault()
                              event.dataTransfer.dropEffect = 'move'
                            }
                          }}
                          onDrop={(event) => {
                            event.preventDefault()
                            const modelId =
                              event.dataTransfer.getData('text/plain') ||
                              draggedModelId
                            if (modelId) moveModel(mode, modelId, index)
                            setDraggedModelId(undefined)
                          }}
                          onDragEnd={() => setDraggedModelId(undefined)}
                          className={cn(
                            'grid grid-cols-[auto_auto_minmax(0,1fr)_auto] items-center gap-2 rounded-md px-2 py-2.5 transition-colors',
                            draggedModelId === model.id
                              ? 'bg-muted opacity-60'
                              : 'hover:bg-muted/50'
                          )}
                        >
                          <span
                            className='text-muted-foreground cursor-grab active:cursor-grabbing'
                            aria-hidden='true'
                          >
                            <HugeiconsIcon
                              icon={DragDropVerticalIcon}
                              strokeWidth={2}
                            />
                          </span>
                          <Badge
                            variant='outline'
                            className='min-w-8 justify-center tabular-nums'
                          >
                            {index + 1}
                          </Badge>
                          <div className='min-w-0'>
                            <div className='truncate text-sm font-medium'>
                              {model.id}
                            </div>
                            <div className='text-muted-foreground mt-0.5 line-clamp-1 text-xs'>
                              {model.description ||
                                t('Ready for creation tasks.')}
                            </div>
                          </div>
                          <div className='flex gap-1'>
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon-sm'
                              aria-label={t('Move {{model}} up', {
                                model: model.id,
                              })}
                              disabled={props.saving || index === 0}
                              onClick={() =>
                                moveModel(mode, model.id, index - 1)
                              }
                            >
                              <HugeiconsIcon
                                data-icon='inline-start'
                                icon={ArrowUp01Icon}
                                strokeWidth={2}
                              />
                            </Button>
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon-sm'
                              aria-label={t('Move {{model}} down', {
                                model: model.id,
                              })}
                              disabled={
                                props.saving || index === modeModels.length - 1
                              }
                              onClick={() =>
                                moveModel(mode, model.id, index + 1)
                              }
                            >
                              <HugeiconsIcon
                                data-icon='inline-start'
                                icon={ArrowDown01Icon}
                                strokeWidth={2}
                              />
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </ScrollArea>
              </TabsContent>
            )
          })}
        </Tabs>

        <DialogFooter className='mt-4 sm:justify-between'>
          <Button
            type='button'
            variant='outline'
            onClick={props.onReset}
            disabled={props.saving}
          >
            <HugeiconsIcon
              data-icon='inline-start'
              icon={ReloadIcon}
              strokeWidth={2}
            />
            {t('Reset to alphabetical order')}
          </Button>
          <Button
            type='submit'
            disabled={
              props.saving ||
              CREATION_MODES.every((mode) => modelsByMode[mode].length === 0)
            }
          >
            {props.saving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <HugeiconsIcon
                data-icon='inline-start'
                icon={FloppyDiskIcon}
                strokeWidth={2}
              />
            )}
            {t('Save order')}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  )
}
