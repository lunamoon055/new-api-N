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
import { useMutation, useQueryClient } from '@tanstack/react-query'
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
  DropdownMenuGroup,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu'
import { refreshChannelCapabilities } from '../api'

export function ChannelCapabilitiesAction(props: { channelId: number }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const queryClient = useQueryClient()
  const refresh = useMutation({
    mutationFn: () => refreshChannelCapabilities(props.channelId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        predicate: (query) =>
          query.queryKey.some(
            (part) =>
              typeof part === 'string' && /pricing|models|playground/.test(part)
          ),
      })
    },
    onError: () =>
      toast.error(
        t(
          'Failed to refresh model capabilities. Existing information has been retained.'
        )
      ),
  })

  const handleRefresh = () => {
    setOpen(true)
    refresh.mutate()
  }

  return (
    <>
      <DropdownMenuGroup>
        <DropdownMenuItem
          onSelect={(event) => {
            event.preventDefault()
            handleRefresh()
          }}
          disabled={refresh.isPending}
        >
          {t('Refresh model capabilities')}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent closeLabel={t('Close')}>
          <DialogHeader>
            <DialogTitle>{t('Refresh model capabilities')}</DialogTitle>
            <DialogDescription>
              {t(
                'Update documentation and API parameters from the current upstream model catalog. Capabilities also refresh after channel saves and on the server schedule.'
              )}
            </DialogDescription>
          </DialogHeader>
          <div className='flex flex-col gap-3' aria-live='polite'>
            {refresh.isPending && (
              <p>{t('Refreshing model capabilities...')}</p>
            )}
            {refresh.isError && (
              <p>
                {t(
                  'Failed to refresh model capabilities. Existing information has been retained.'
                )}
              </p>
            )}
            {refresh.isSuccess && (
              <>
                <p>
                  {t('Updated capability information for {{count}} models.', {
                    count: refresh.data.updated_models,
                  })}
                </p>
                {refresh.data.unconfirmed_models.length > 0 ? (
                  <>
                    <p className='text-muted-foreground text-sm'>
                      {t(
                        'The upstream has not published all capabilities for these models. Unconfirmed parameters remain visible:'
                      )}
                    </p>
                    <ul className='max-h-64 list-disc overflow-y-auto pl-5 text-sm'>
                      {refresh.data.unconfirmed_models.map((name) => (
                        <li key={name}>{name}</li>
                      ))}
                    </ul>
                  </>
                ) : (
                  <p>
                    {t(
                      'All synchronized models have published capability information.'
                    )}
                  </p>
                )}
              </>
            )}
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setOpen(false)}>
              {t('Close')}
            </Button>
            <Button
              onClick={() => refresh.mutate()}
              disabled={refresh.isPending}
            >
              {t('Refresh model capabilities')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
