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
import { useMemo, useRef, useState, useSyncExternalStore } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Check, Gift, Loader2 } from 'lucide-react'
import { useAuthStore } from '@/stores/auth-store'
import { getSelf } from '@/lib/api'
import {
  clearCreditPurchase,
  parseCreditPurchaseIntent,
  readCreditPurchaseSnapshot,
  subscribeCreditPurchase,
  WALLET_BALANCE_EVENT,
  type CreditPurchaseIntent,
} from '@/lib/credit-purchase'
import { formatQuota } from '@/lib/format'
import { hasPendingRedemptionLink } from '@/lib/redemption-link'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { redeemTopupCode } from '../../api'

const emptySnapshot = () => null

export function PurchaseRedemptionDialog() {
  const user = useAuthStore((state) => state.auth.user)
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const snapshot = useSyncExternalStore(
    subscribeCreditPurchase,
    readCreditPurchaseSnapshot,
    emptySnapshot
  )
  const intent = useMemo(() => parseCreditPurchaseIntent(snapshot), [snapshot])
  if (
    !intent ||
    !user ||
    intent.userId !== user.id ||
    /^\/(redeem|sign-in|sign-up|setup|oauth)(\/|$)/.test(pathname) ||
    hasPendingRedemptionLink()
  )
    return null
  return (
    <PurchaseRedemptionContent
      key={`${intent.id}:${user.id}`}
      intent={intent}
    />
  )
}

function PurchaseRedemptionContent(props: { intent: CreditPurchaseIntent }) {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const [code, setCode] = useState('')
  const [redeemedQuota, setRedeemedQuota] = useState<number | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const complete = redeemedQuota !== null
  const redeem = useMutation({
    mutationFn: async () => {
      const response = await redeemTopupCode(
        { key: code.trim(), expected_user_id: props.intent.userId },
        {
          skipBusinessError: true,
        } as Record<string, unknown>
      )
      if (!response.success || typeof response.data !== 'number')
        throw new Error(response.message || t('Redemption failed'))
      return response.data
    },
    onSuccess: async (quota) => {
      // A refresh failure must not make an already-consumed code retryable.
      setRedeemedQuota(quota)
      setCode('')
      window.dispatchEvent(new Event(WALLET_BALANCE_EVENT))
      const response = await getSelf().catch(() => null)
      if (
        response?.success &&
        response.data &&
        useAuthStore.getState().auth.user?.id === props.intent.userId
      ) {
        useAuthStore.getState().auth.setUser(response.data)
      }
    },
  })
  const close = () => {
    if (!redeem.isPending) clearCreditPurchase(props.intent.id)
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) close()
      }}
    >
      <DialogContent
        className='max-h-[calc(100dvh-2rem)] gap-6 overflow-y-auto p-6 sm:max-w-[520px] sm:p-8'
        initialFocus={inputRef}
        closeLabel={t('Close')}
        showCloseButton={!redeem.isPending}
      >
        <DialogHeader className='items-center gap-3 text-center'>
          <div className='bg-muted text-muted-foreground flex size-14 items-center justify-center rounded-full'>
            {complete ? (
              <Check className='text-success size-7' aria-hidden />
            ) : (
              <Gift className='size-7' aria-hidden />
            )}
          </div>
          <DialogTitle className='text-2xl'>
            {complete
              ? t('Credits redeemed successfully')
              : t('Redeem credits')}
          </DialogTitle>
          <DialogDescription>
            {complete
              ? t('Your credits have been added to this account.')
              : t(
                  'Paste the code from your purchase delivery details to redeem your credits.'
                )}
          </DialogDescription>
        </DialogHeader>
        {complete ? (
          <div
            className='flex flex-col items-center gap-2 py-3'
            aria-live='polite'
          >
            <p className='text-4xl font-semibold tabular-nums'>
              +{formatQuota(redeemedQuota)}
            </p>
            <p className='text-muted-foreground text-sm'>{t('Credits')}</p>
          </div>
        ) : (
          <form
            className='flex flex-col gap-5'
            onSubmit={(event) => {
              event.preventDefault()
              if (code.trim() && !redeem.isPending) redeem.mutate()
            }}
          >
            <div className='bg-muted/50 flex flex-col gap-1 rounded-lg p-4'>
              <p className='text-muted-foreground text-xs'>
                {t('Current account')}
              </p>
              <p className='truncate font-medium'>
                {user?.display_name || user?.username}{' '}
                <span className='text-muted-foreground text-xs'>
                  @{user?.username}
                </span>
              </p>
              <p className='text-muted-foreground text-xs'>
                {t('Selected recharge tier: {{tier}}', {
                  tier:
                    props.intent.tier === 'other'
                      ? t('Other')
                      : props.intent.tier,
                })}
              </p>
            </div>
            <Field data-invalid={redeem.isError}>
              <FieldLabel htmlFor='purchase-redemption-code'>
                {t('Redemption code')}
              </FieldLabel>
              <Input
                ref={inputRef}
                id='purchase-redemption-code'
                value={code}
                onChange={(event) => {
                  setCode(event.target.value)
                  if (redeem.isError) redeem.reset()
                }}
                placeholder={t('Paste the redemption code you received')}
                autoComplete='off'
                maxLength={128}
                disabled={redeem.isPending}
                aria-invalid={redeem.isError}
                className='h-12'
              />
              <FieldDescription>
                {t(
                  'The credited amount is determined by the redemption code, not the selected tier.'
                )}
              </FieldDescription>
            </Field>
            {redeem.error && (
              <Alert variant='destructive' role='alert'>
                <AlertDescription>{redeem.error.message}</AlertDescription>
              </Alert>
            )}
            <Button
              type='submit'
              size='xl'
              disabled={!code.trim() || redeem.isPending}
            >
              {redeem.isPending && (
                <Loader2 data-icon='inline-start' className='animate-spin' />
              )}
              {redeem.isPending ? t('Redeeming...') : t('Redeem credits')}
            </Button>
          </form>
        )}
        <Button
          size={complete ? 'xl' : 'default'}
          variant={complete ? 'default' : 'ghost'}
          disabled={redeem.isPending}
          onClick={close}
        >
          {complete ? t('Done') : t('Redeem later')}
        </Button>
      </DialogContent>
    </Dialog>
  )
}
