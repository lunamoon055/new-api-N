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
import {
  useIsMutating,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useIsAdmin } from '@/hooks/use-admin'
import { StatusBadge } from '@/components/status-badge'
import { Field, FieldLabel } from '@/components/ui/field'
import { Switch } from '@/components/ui/switch'
import { updateModelDisplayStatus } from '../api'
import type { PricingData } from '../types'

interface ModelAvailabilityProps {
  modelName: string
  available?: boolean
}

export function ModelAvailability(props: ModelAvailabilityProps) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const available = props.available !== false

  return (
    <div className='flex shrink-0 items-center gap-3'>
      <StatusBadge
        label={t(available ? 'Available' : 'Unavailable')}
        variant={available ? 'success' : 'danger'}
        copyable={false}
        size='sm'
      />
      {isAdmin && (
        <AdminModelAvailability
          modelName={props.modelName}
          available={available}
        />
      )}
    </div>
  )
}

function AdminModelAvailability(props: {
  modelName: string
  available: boolean
}) {
  const { t } = useTranslation()
  const id = useId()
  const queryClient = useQueryClient()
  const mutationKey = ['model-display-status', props.modelName]
  const pending = useIsMutating({ mutationKey }) > 0
  const mutation = useMutation({
    mutationKey,
    mutationFn: (available: boolean) =>
      updateModelDisplayStatus(props.modelName, available),
    onSuccess: async (response) => {
      queryClient.setQueryData<PricingData>(['pricing'], (current) => {
        if (!current) return current
        return {
          ...current,
          data: current.data.map((model) =>
            model.model_name === response.data.model_name
              ? { ...model, display_available: response.data.available }
              : model
          ),
        }
      })
      await queryClient.invalidateQueries({ queryKey: ['pricing'] })
      toast.success(t('Model availability display updated.'))
    },
    onError: () =>
      toast.error(t('Unable to update model availability display.')),
  })

  return (
    <Field
      orientation='horizontal'
      className='w-auto'
      data-disabled={pending}
      onClick={(event) => event.stopPropagation()}
    >
      <FieldLabel htmlFor={id}>{t('Public display')}</FieldLabel>
      <Switch
        id={id}
        size='sm'
        checked={props.available}
        onCheckedChange={(checked) => mutation.mutate(checked)}
        disabled={pending}
        aria-label={t('Public availability for {{model}}', {
          model: props.modelName,
        })}
        title={t('Only changes Model Square display; API access is unchanged.')}
      />
    </Field>
  )
}
