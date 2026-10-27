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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { TagInput } from '@/components/tag-input'
import { batchAddModelTags, getModelTags } from '../../api'
import { modelsQueryKeys } from '../../lib/query-keys'

const batchTagsSchema = z.object({
  tags: z
    .array(z.string().trim().min(1))
    .min(1)
    .refine((tags) => tags.every((tag) => !/[,\r\n]/.test(tag)))
    .refine((tags) => Array.from(tags.join(',')).length <= 255),
})
type BatchTagsValues = z.infer<typeof batchTagsSchema>

interface BatchTagsDialogProps {
  modelIds: number[]
  onClose: () => void
  onSuccess: () => void
}

export function BatchTagsDialog(props: BatchTagsDialogProps) {
  const { t } = useTranslation()
  const id = useId()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const form = useForm<BatchTagsValues>({
    resolver: zodResolver(batchTagsSchema),
    defaultValues: { tags: [] },
  })
  const tags = form.watch('tags')
  const tagsQuery = useQuery({
    queryKey: modelsQueryKeys.tags(),
    queryFn: getModelTags,
    staleTime: 0,
  })
  const mutation = useMutation({
    mutationFn: (values: BatchTagsValues) =>
      batchAddModelTags({ ids: props.modelIds, tags: values.tags }),
    onSuccess: async (response) => {
      if (!response.success) return
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: modelsQueryKeys.all }),
        queryClient.invalidateQueries({ queryKey: ['pricing'] }),
      ])
      toast.success(
        t('Tags added to {{count}} model(s).', {
          count: response.data.updated_count,
        })
      )
      props.onSuccess()
      props.onClose()
    },
  })
  const availableTags = (tagsQuery.data?.data ?? []).filter((tag) =>
    tag.toLocaleLowerCase().includes(search.trim().toLocaleLowerCase())
  )

  function updateTags(values: string[]) {
    form.setValue('tags', [...new Set(values)], { shouldValidate: true })
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) props.onClose()
      }}
    >
      <DialogContent
        className='max-h-[90dvh] overflow-y-auto sm:max-w-lg'
        showCloseButton={!mutation.isPending}
      >
        <DialogHeader>
          <DialogTitle>{t('Batch add tags')}</DialogTitle>
          <DialogDescription>
            {t(
              'Add tags to {{count}} selected model(s). Existing tags are kept and duplicates are skipped.',
              {
                count: props.modelIds.length,
              }
            )}
          </DialogDescription>
        </DialogHeader>
        <form
          className='flex min-h-0 flex-col gap-5'
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <FieldGroup>
            <Field
              data-invalid={!!form.formState.errors.tags}
              data-disabled={mutation.isPending}
            >
              <FieldLabel htmlFor={`${id}-tags`}>{t('Tags to add')}</FieldLabel>
              <TagInput
                id={`${id}-tags`}
                aria-invalid={!!form.formState.errors.tags}
                value={tags}
                onChange={updateTags}
                disabled={mutation.isPending}
                placeholder={t('Add tags...')}
              />
              <FieldDescription>
                {t('Press Enter or comma to add tags')}
              </FieldDescription>
              {form.formState.errors.tags && (
                <FieldError>
                  {t(
                    'Select or enter tags, up to 255 characters in total. Tags cannot contain commas or line breaks.'
                  )}
                </FieldError>
              )}
            </Field>
            <FieldSet disabled={mutation.isPending}>
              <FieldLegend variant='label'>
                {t('Existing tags (multiple selection)')}
              </FieldLegend>
              <Input
                id={`${id}-search`}
                aria-label={t('Search existing tags')}
                placeholder={t('Search existing tags')}
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
              <FieldGroup className='max-h-48 overflow-y-auto rounded-md border p-3'>
                {availableTags.map((tag, index) => (
                  <Field key={tag} orientation='horizontal'>
                    <Checkbox
                      id={`${id}-option-${index}`}
                      checked={tags.includes(tag)}
                      onCheckedChange={(checked) =>
                        updateTags(
                          checked
                            ? [...tags, tag]
                            : tags.filter((value) => value !== tag)
                        )
                      }
                    />
                    <FieldLabel
                      htmlFor={`${id}-option-${index}`}
                      className='min-w-0 break-all'
                    >
                      {tag}
                    </FieldLabel>
                  </Field>
                ))}
                {availableTags.length === 0 && (
                  <FieldDescription>
                    {tagsQuery.isPending
                      ? t('Loading...')
                      : t('No matching tags. Enter a new tag above.')}
                  </FieldDescription>
                )}
              </FieldGroup>
              {(tagsQuery.isError || tagsQuery.data?.success === false) && (
                <FieldError>
                  {t(
                    'Existing tags could not be loaded. You can still enter new tags.'
                  )}
                </FieldError>
              )}
            </FieldSet>
          </FieldGroup>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              disabled={mutation.isPending}
              onClick={props.onClose}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='submit'
              disabled={mutation.isPending || tags.length === 0}
            >
              {mutation.isPending && (
                <Loader2 data-icon='inline-start' className='animate-spin' />
              )}
              {t('Apply tags')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
