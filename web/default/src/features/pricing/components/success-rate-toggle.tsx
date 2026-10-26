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
import { useId } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useIsAdmin } from '@/hooks/use-admin'
import { Field, FieldLabel } from '@/components/ui/field'
import { Switch } from '@/components/ui/switch'
import { updateSuccessRateVisibility } from '@/features/performance-metrics/api'
import {
  successRateVisibilityQueryKey,
  useSuccessRateVisibility,
} from '@/features/performance-metrics/use-success-rate-visibility'

export function SuccessRateToggle() {
  const isAdmin = useIsAdmin()
  return isAdmin ? <AdminSuccessRateToggle /> : null
}

function AdminSuccessRateToggle() {
  const { t } = useTranslation()
  const id = useId()
  const queryClient = useQueryClient()
  const visibility = useSuccessRateVisibility()
  const mutation = useMutation({
    mutationFn: updateSuccessRateVisibility,
    onSuccess: async (response) => {
      queryClient.setQueryData(successRateVisibilityQueryKey, response)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['perf-metrics'] }),
        queryClient.invalidateQueries({ queryKey: ['perf-metrics-summary'] }),
      ])
      toast.success(t('Success rate display updated.'))
    },
    onError: () => toast.error(t('Unable to update success rate display.')),
  })
  const disabled =
    !visibility.isSuccess || visibility.isFetching || mutation.isPending

  return (
    <Field orientation='horizontal' className='w-auto' data-disabled={disabled}>
      <FieldLabel htmlFor={id}>{t('Show success rate')}</FieldLabel>
      <Switch
        id={id}
        size='sm'
        checked={visibility.showSuccessRate}
        onCheckedChange={(checked) => mutation.mutate(checked)}
        disabled={disabled}
      />
    </Field>
  )
}
