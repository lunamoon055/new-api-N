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
import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useMutation } from '@tanstack/react-query'
import { Gift, Loader2, CircleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { readRedemptionLinkCode } from '@/lib/redemption-link'
import { clearCreditPurchase } from '@/lib/credit-purchase'
import { Button } from '@/components/ui/button'
import { createRedemptionLinkIntent } from './api'

export function RedemptionLink() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const started = useRef(false)
  const [code] = useState(() => readRedemptionLinkCode(window.location.hash))
  const initialize = useMutation({
    mutationFn: createRedemptionLinkIntent,
    onSuccess: () => {
      clearCreditPurchase()
      return navigate({ to: '/wallet', replace: true })
    },
  })
  const { mutate } = initialize

  useEffect(() => {
    if (!code || started.current) return
    started.current = true
    mutate(code)
  }, [code, mutate])

  const failed = !code || initialize.isError
  const error =
    initialize.error?.message ||
    t('Open the complete redemption link you received after purchase.')

  return (
    <main className='bg-background flex min-h-svh items-center justify-center px-6 py-12'>
      <div className='flex w-full max-w-sm flex-col items-center gap-6 text-center'>
        <div className='bg-muted text-muted-foreground flex size-16 items-center justify-center rounded-full'>
          {failed ? (
            <CircleAlert className='size-7' aria-hidden />
          ) : (
            <Gift className='size-7' aria-hidden />
          )}
        </div>
        <h1 className='text-2xl font-semibold tracking-tight'>
          {failed
            ? t('Unable to open redemption link')
            : t('Opening your redemption link')}
        </h1>
        <p className='text-muted-foreground text-sm'>
          {failed
            ? error
            : t(
                'Your code is being recognized. No credits will be redeemed until you confirm.'
              )}
        </p>
        {initialize.isPending && (
          <Loader2
            className='text-muted-foreground size-5 animate-spin'
            aria-label={t('Loading...')}
          />
        )}
        {failed && code && (
          <Button onClick={() => mutate(code)} disabled={initialize.isPending}>
            {t('Retry')}
          </Button>
        )}
        {failed && (
          <Button variant='link' render={<Link to='/wallet' />}>
            {t('Back to wallet')}
          </Button>
        )}
      </div>
    </main>
  )
}
