/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
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
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Form, FormField } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { manuallyTransferMedia } from '../api'
import type { MediaStorageProvider, MediaTransferJob } from '../types'

const createSchema = (t: (key: string) => string) =>
  z
    .object({
      mode: z.enum(['file', 'url']),
      providerId: z.string().min(1, t('Select a storage provider')),
      sourceUrl: z.string(),
      file: z.instanceof(File).optional(),
    })
    .superRefine((values, context) => {
      if (values.mode === 'file') {
        if (
          !values.file ||
          !values.file.size ||
          values.file.size > 200 * 1024 * 1024
        ) {
          context.addIssue({
            code: 'custom',
            path: ['file'],
            message: t('Select an MP4 or WebM video up to 200 MB'),
          })
        }
        return
      }
      try {
        const url = new URL(values.sourceUrl.trim())
        if (
          !['http:', 'https:'].includes(url.protocol) ||
          url.username ||
          url.password
        )
          throw new Error()
      } catch {
        context.addIssue({
          code: 'custom',
          path: ['sourceUrl'],
          message: t('Provide a valid public HTTP download URL'),
        })
      }
    })

type FormValues = z.infer<ReturnType<typeof createSchema>>
type ManualMediaTransferDialogProps = {
  job: MediaTransferJob
  providers: MediaStorageProvider[]
  onClose: () => void
}

export function ManualMediaTransferDialog(
  props: ManualMediaTransferDialogProps
) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [progress, setProgress] = useState(0)
  const providers = props.providers.filter((provider) => provider.enabled)
  const form = useForm<FormValues>({
    resolver: zodResolver(createSchema(t)),
    defaultValues: {
      mode: 'file',
      providerId: providers[0]?.id ?? '',
      sourceUrl: '',
    },
  })
  const mode = form.watch('mode')
  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      manuallyTransferMedia({
        id: props.job.id,
        providerId: values.providerId,
        file: values.mode === 'file' ? values.file : undefined,
        sourceUrl: values.mode === 'url' ? values.sourceUrl : undefined,
        onProgress: setProgress,
      }),
    onSuccess: () => {
      toast.success(t('Manual media transfer queued'))
      queryClient.invalidateQueries({ queryKey: ['media-transfer-dashboard'] })
      props.onClose()
    },
    onError: () =>
      form.setError('root', {
        message: t(
          'Manual media transfer failed. Check the task status and try again.'
        ),
      }),
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) props.onClose()
      }}
    >
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Manual media transfer')}</DialogTitle>
          <DialogDescription>
            {t(
              'Upload the generated video or provide a replacement download URL to complete this task.'
            )}
          </DialogDescription>
        </DialogHeader>
        <p className='text-muted-foreground break-all font-mono text-xs'>
          {props.job.task_id || props.job.id}
        </p>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((values) => {
              setProgress(0)
              form.clearErrors('root')
              mutation.mutate(values)
            })}
            className='flex flex-col gap-5'
          >
            <FieldGroup>
              <Field>
                <FieldLabel>{t('Transfer source')}</FieldLabel>
                <ToggleGroup
                  variant='outline'
                  value={[mode]}
                  disabled={mutation.isPending}
                  onValueChange={(values) => {
                    if (values[0])
                      form.setValue('mode', values[0] as FormValues['mode'])
                  }}
                >
                  <ToggleGroupItem value='file'>
                    {t('Upload local video')}
                  </ToggleGroupItem>
                  <ToggleGroupItem value='url'>
                    {t('Download URL')}
                  </ToggleGroupItem>
                </ToggleGroup>
              </Field>
              {mode === 'file' && (
                <FormField
                  control={form.control}
                  name='file'
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor='manual-transfer-file'>
                        {t('Video file')}
                      </FieldLabel>
                      <Input
                        id='manual-transfer-file'
                        type='file'
                        accept='.mp4,.webm,video/mp4,video/webm'
                        disabled={mutation.isPending}
                        aria-invalid={fieldState.invalid}
                        onChange={(event) =>
                          field.onChange(event.target.files?.[0])
                        }
                      />
                      <FieldDescription>
                        {t('Select an MP4 or WebM video up to 200 MB')}
                      </FieldDescription>
                      <FieldError errors={[fieldState.error]} />
                    </Field>
                  )}
                />
              )}
              {mode === 'url' && (
                <FormField
                  control={form.control}
                  name='sourceUrl'
                  render={({ field, fieldState }) => (
                    <Field data-invalid={fieldState.invalid}>
                      <FieldLabel htmlFor='manual-transfer-url'>
                        {t('Download URL')}
                      </FieldLabel>
                      <Input
                        {...field}
                        id='manual-transfer-url'
                        type='url'
                        autoComplete='off'
                        placeholder='https://…/video.mp4'
                        disabled={mutation.isPending}
                        aria-invalid={fieldState.invalid}
                      />
                      <FieldDescription>
                        {t(
                          'Use a direct download URL. If it requires website login, download the video and upload the local file instead.'
                        )}
                      </FieldDescription>
                      <FieldError errors={[fieldState.error]} />
                    </Field>
                  )}
                />
              )}
              <FormField
                control={form.control}
                name='providerId'
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor='manual-transfer-provider'>
                      {t('Storage provider')}
                    </FieldLabel>
                    <Select
                      value={field.value}
                      onValueChange={(value) => field.onChange(value ?? '')}
                      disabled={mutation.isPending || !providers.length}
                      items={providers.map((provider) => ({
                        value: provider.id,
                        label: provider.name || provider.id,
                      }))}
                    >
                      <SelectTrigger
                        id='manual-transfer-provider'
                        aria-invalid={fieldState.invalid}
                      >
                        <SelectValue
                          placeholder={t('Select a storage provider')}
                        />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {providers.map((provider) => (
                            <SelectItem key={provider.id} value={provider.id}>
                              {provider.name || provider.id}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    {!providers.length && (
                      <FieldDescription>
                        {t(
                          'Enable and save a storage provider before transferring'
                        )}
                      </FieldDescription>
                    )}
                    <FieldError errors={[fieldState.error]} />
                  </Field>
                )}
              />
            </FieldGroup>
            {mutation.isPending && mode === 'file' && (
              <div className='flex flex-col gap-2' aria-live='polite'>
                <Progress value={progress} />
                <p className='text-muted-foreground text-xs'>
                  {t('Uploading video: {{percent}}%', { percent: progress })}
                </p>
              </div>
            )}
            <FieldError errors={[form.formState.errors.root]} />
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={props.onClose}
                disabled={mutation.isPending}
              >
                {t('Cancel')}
              </Button>
              <Button
                type='submit'
                disabled={mutation.isPending || !providers.length}
              >
                {mutation.isPending
                  ? t('Submitting...')
                  : t('Start manual transfer')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
