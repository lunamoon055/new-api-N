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
import { useEffect } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

type WorkbenchLinkSectionProps = {
  value: string
}

export function WorkbenchLinkSection(props: WorkbenchLinkSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = z.object({
    link: z
      .string()
      .trim()
      .refine((value) => {
        if (!value) return true
        try {
          const url = new URL(value)
          return (
            ['http:', 'https:'].includes(url.protocol) &&
            !!url.hostname &&
            !url.username &&
            !url.password &&
            !/\s/.test(value)
          )
        } catch {
          return false
        }
      }, t('Provide a valid URL starting with http:// or https://')),
  })
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { link: props.value },
  })

  useEffect(() => {
    form.reset({ link: props.value })
  }, [props.value, form])

  const onSubmit = async (values: z.infer<typeof schema>) => {
    const result = await updateOption.mutateAsync({
      key: 'general_setting.workbench_link',
      value: values.link,
    })
    if (result.success) form.reset(values)
  }

  return (
    <SettingsSection
      title={t('Workbench link')}
      description={t(
        'Configure the destination of the Workbench navigation link.'
      )}
    >
      <FormNavigationGuard when={form.formState.isDirty} />
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className='flex flex-col gap-6'
      >
        <FieldGroup>
          <Controller
            name='link'
            control={form.control}
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldLabel htmlFor='workbench-link'>
                  {t('Workbench link')}
                </FieldLabel>
                <Input
                  {...field}
                  id='workbench-link'
                  type='url'
                  placeholder='https://unline.visualwk.com'
                  aria-invalid={fieldState.invalid}
                  aria-describedby='workbench-link-description'
                />
                <FieldDescription id='workbench-link-description'>
                  {t(
                    'Open the Workbench in a new tab. Leave blank to hide the navigation link.'
                  )}
                </FieldDescription>
                {fieldState.invalid && (
                  <FieldError errors={[fieldState.error]} />
                )}
              </Field>
            )}
          />
        </FieldGroup>
        <Button
          type='submit'
          className='self-start'
          disabled={updateOption.isPending || form.formState.isSubmitting}
        >
          {updateOption.isPending ? t('Saving...') : t('Save Changes')}
        </Button>
      </form>
    </SettingsSection>
  )
}
