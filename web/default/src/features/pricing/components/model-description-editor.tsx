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
import { useId, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useIsAdmin } from '@/hooks/use-admin'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { updateModelDescription } from '../api'

const MAX_MODEL_DESCRIPTION_LENGTH = 2000

interface ModelDescriptionEditorProps {
  modelName: string
  description?: string
  compact?: boolean
}

export function ModelDescriptionEditor(props: ModelDescriptionEditorProps) {
  const isAdmin = useIsAdmin()

  if (!isAdmin) return null

  return <AdminModelDescriptionEditor {...props} />
}

function AdminModelDescriptionEditor(props: ModelDescriptionEditorProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const fieldId = useId()
  const [open, setOpen] = useState(false)
  const [description, setDescription] = useState(props.description ?? '')

  const updateMutation = useMutation({
    mutationFn: () =>
      updateModelDescription(props.modelName, description.trim()),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Unable to save note.'))
        return
      }
      await queryClient.invalidateQueries({ queryKey: ['pricing'] })
      toast.success(t('Note saved.'))
      setOpen(false)
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Unable to save note.'))
    },
  })

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setDescription(props.description ?? '')
    }
    setOpen(nextOpen)
  }

  return (
    <>
      <Button
        type='button'
        variant='ghost'
        size={props.compact ? 'xs' : 'sm'}
        className='shrink-0'
        onClick={(event) => {
          event.stopPropagation()
          handleOpenChange(true)
        }}
      >
        {t(props.description ? 'Edit note' : 'Add note')}
      </Button>

      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t(props.description ? 'Edit model note' : 'Add model note')}
            </DialogTitle>
            <DialogDescription>
              {t(
                'The note for {{model}} is visible to everyone in Model Square.',
                { model: props.modelName }
              )}
            </DialogDescription>
          </DialogHeader>

          <Field>
            <FieldLabel htmlFor={fieldId}>{t('Model note')}</FieldLabel>
            <Textarea
              id={fieldId}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder={t('Write a short note about this model...')}
              maxLength={MAX_MODEL_DESCRIPTION_LENGTH}
              rows={5}
              disabled={updateMutation.isPending}
            />
            <FieldDescription>
              {t(
                'Leave blank to remove the note. {{count}}/{{max}} characters.',
                {
                  count: [...description].length,
                  max: MAX_MODEL_DESCRIPTION_LENGTH,
                }
              )}
            </FieldDescription>
          </Field>

          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => handleOpenChange(false)}
              disabled={updateMutation.isPending}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              onClick={() => updateMutation.mutate()}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending && <Spinner data-icon='inline-start' />}
              {t(updateMutation.isPending ? 'Saving...' : 'Save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
