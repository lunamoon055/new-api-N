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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { batchClearModelTags } from '../../api'
import { modelsQueryKeys } from '../../lib/query-keys'

interface ClearTagsDialogProps {
  modelIds: number[]
  onClose: () => void
  onSuccess: () => void
}

export function ClearTagsDialog(props: ClearTagsDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => batchClearModelTags(props.modelIds),
    onSuccess: async (response) => {
      if (!response.success) return
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: modelsQueryKeys.all }),
        queryClient.invalidateQueries({ queryKey: ['pricing'] }),
      ])
      toast.success(
        t('Tags cleared from {{count}} model(s).', {
          count: response.data.updated_count,
        })
      )
      props.onSuccess()
      props.onClose()
    },
  })

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) props.onClose()
      }}
    >
      <DialogContent showCloseButton={!mutation.isPending}>
        <DialogHeader>
          <DialogTitle>{t('Clear model tags?')}</DialogTitle>
          <DialogDescription>
            {t(
              'Remove all tags from {{count}} selected model(s)? You can add tags again afterwards.',
              {
                count: props.modelIds.length,
              }
            )}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant='outline'
            disabled={mutation.isPending}
            onClick={props.onClose}
          >
            {t('Cancel')}
          </Button>
          <Button
            variant='destructive'
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending && (
              <Loader2 data-icon='inline-start' className='animate-spin' />
            )}
            {t('Confirm clear tags')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
