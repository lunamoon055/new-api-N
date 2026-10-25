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
import { useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Check, CircleAlert, Gift, Loader2, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { formatQuota } from '@/lib/format'
import { hasPendingRedemptionLink } from '@/lib/redemption-link'
import { logout } from '@/features/auth/api'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import {
  dismissRedemptionLink,
  previewRedemptionLink,
  redeemRedemptionLink,
} from '../../api'

interface RedemptionLinkDialogProps {
  onRedeemed: () => Promise<void>
}

export function RedemptionLinkDialog(props: RedemptionLinkDialogProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const titleRef = useRef<HTMLHeadingElement>(null)
  const user = useAuthStore((state) => state.auth.user)
  const [open, setOpen] = useState(hasPendingRedemptionLink)
  const [redeemedQuota, setRedeemedQuota] = useState<number | null>(null)
  const complete = redeemedQuota !== null
  const preview = useQuery({
    queryKey: ['redemption-link', user?.id],
    queryFn: previewRedemptionLink,
    enabled: open && !!user && !complete,
    retry: false,
    staleTime: 0,
    gcTime: 0,
    refetchOnWindowFocus: false,
  })
  const redeem = useMutation({
    mutationFn: () => redeemRedemptionLink(preview.data!),
    onSuccess: async (quota) => {
      // The code has already been consumed; a refresh failure must never turn
      // this into a retryable redemption or hide the successful result.
      setRedeemedQuota(quota)
      await props.onRedeemed()
    },
  })
  const dismiss = useMutation({
    mutationFn: dismissRedemptionLink,
    onSuccess: () => setOpen(false),
  })

  const switchAccount = useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      useAuthStore.getState().auth.reset()
      window.localStorage.removeItem('uid')
      await navigate({
        to: '/sign-in',
        search: { redirect: '/wallet' },
        replace: true,
      })
    },
  })

  const busy = redeem.isPending || dismiss.isPending || switchAccount.isPending
  const error = complete
    ? null
    : redeem.error || preview.error || dismiss.error || switchAccount.error
  const quota = complete ? redeemedQuota : preview.data?.quota
  const username = user?.display_name || user?.username
  const ready =
    preview.isSuccess &&
    preview.data.user_id === user?.id &&
    !preview.isFetching &&
    !redeem.isError

  const close = () => {
    if (busy) return
    if (complete) setOpen(false)
    else dismiss.mutate()
  }

  let amountContent = (
    <p className='text-muted-foreground text-xl font-medium'>
      {t('Redemption unavailable')}
    </p>
  )
  if (quota !== undefined) {
    amountContent = (
      <p className='text-4xl leading-tight font-semibold tabular-nums sm:text-5xl'>
        {formatQuota(quota)}
      </p>
    )
  } else if (preview.isPending) {
    amountContent = <Skeleton className='h-12 w-32' />
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) close()
      }}
    >
      <DialogContent
        className='max-h-[calc(100dvh-2rem)] gap-6 overflow-y-auto p-6 sm:max-w-[590px] sm:gap-7 sm:p-9 [&>[data-slot=dialog-close]]:top-5 [&>[data-slot=dialog-close]]:right-5'
        overlayClassName='bg-black/40 supports-backdrop-filter:backdrop-blur-none'
        initialFocus={titleRef}
        closeLabel={t('Close')}
        showCloseButton={!busy}
      >
        <DialogHeader className='items-center gap-4 text-center'>
          <div className='bg-muted text-muted-foreground flex size-16 items-center justify-center rounded-full sm:size-18'>
            {complete ? (
              <Check className='text-success size-8' aria-hidden />
            ) : (
              <Gift className='size-8' aria-hidden />
            )}
          </div>
          <DialogTitle
            ref={titleRef}
            tabIndex={-1}
            className='text-2xl leading-tight font-bold tracking-tight outline-none sm:text-[32px]'
          >
            {complete
              ? t('Credits redeemed successfully')
              : t('Redemption code recognized')}
          </DialogTitle>
          <DialogDescription className='sm:text-lg'>
            {complete
              ? t('Your credits have been added to this account.')
              : t('Confirm to add the credits to the account below.')}
          </DialogDescription>
        </DialogHeader>

        <div
          className='flex flex-col items-center gap-2 py-1'
          aria-live='polite'
        >
          {amountContent}
          <p className='text-muted-foreground text-sm'>{t('Credits')}</p>
        </div>

        <Separator />
        <div className='bg-muted/50 flex min-w-0 items-center gap-3 rounded-xl p-4'>
          <Avatar size='lg' className='size-12'>
            <AvatarFallback>
              <UserRound className='size-5' aria-hidden />
            </AvatarFallback>
          </Avatar>
          <div className='flex min-w-0 flex-1 flex-col gap-1'>
            <p className='text-muted-foreground text-sm'>
              {t('Current account')}
            </p>
            <p className='truncate font-semibold sm:text-lg' title={username}>
              {username || t('Loading...')}
            </p>
            {user?.display_name && user.display_name !== user.username && (
              <p className='text-muted-foreground truncate text-xs'>
                @{user.username}
              </p>
            )}
          </div>
          {!complete && (
            <Button
              variant='link'
              size='default'
              onClick={() => switchAccount.mutate()}
              disabled={busy}
            >
              {t('Switch account')}
            </Button>
          )}
        </div>

        {error ? (
          <Alert variant='destructive'>
            <CircleAlert aria-hidden />
            <AlertDescription>
              {error.message || t('Redemption failed')}
            </AlertDescription>
          </Alert>
        ) : (
          <Alert className='border-success/10 bg-success/5 py-4 text-success'>
            <Check aria-hidden />
            <AlertDescription className='text-success'>
              {complete
                ? t('Your wallet balance has been updated.')
                : t('Redemption details filled in automatically')}
            </AlertDescription>
          </Alert>
        )}

        <div className='flex flex-col gap-3'>
          {complete ? (
            <Button size='xl' className='w-full sm:h-14' onClick={close}>
              {t('Back to wallet')}
            </Button>
          ) : (
            <>
              <Button
                size='xl'
                className='w-full sm:h-14'
                disabled={!ready || busy}
                onClick={() => redeem.mutate()}
              >
                {redeem.isPending && (
                  <Loader2 data-icon='inline-start' className='animate-spin' />
                )}
                {redeem.isPending ? t('Redeeming...') : t('Confirm redemption')}
              </Button>
              <p className='text-muted-foreground text-center text-xs'>
                {t(
                  'Check the account. Redeemed credits cannot be transferred.'
                )}
              </p>
              {error && (
                <Button
                  variant='outline'
                  onClick={() => {
                    redeem.reset()
                    preview.refetch()
                  }}
                  disabled={busy}
                >
                  {t('Refresh redemption details')}
                </Button>
              )}
              <Button variant='ghost' onClick={close} disabled={busy}>
                {t('Redeem later')}
              </Button>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
