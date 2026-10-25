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
import { ExternalLink, Gift, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import {
  beginCreditPurchase,
  CREDIT_PURCHASE_TIERS,
  type CreditPurchaseLinks,
} from '@/lib/credit-purchase'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface RedemptionCodeSectionProps {
  purchaseLinks: CreditPurchaseLinks
  redemptionCode: string
  onRedemptionCodeChange: (code: string) => void
  onRedeem: () => void
  redeeming: boolean
}

export function RedemptionCodeSection(props: RedemptionCodeSectionProps) {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)

  return (
    <section
      className='flex flex-col gap-5'
      aria-label={t('Buy and redeem credits')}
    >
      <div className='bg-muted/40 flex flex-col gap-4 rounded-xl border p-4 sm:p-5'>
        <div className='flex flex-col gap-1.5'>
          <h3 className='text-base font-semibold'>
            {t('Choose a recharge tier')}
          </h3>
          <p className='text-muted-foreground text-sm leading-relaxed'>
            {t('The store opens in a new tab. Keep this wallet page open.')}
          </p>
        </div>
        <div className='grid grid-cols-3 gap-2 sm:grid-cols-6 sm:gap-3'>
          {CREDIT_PURCHASE_TIERS.map((tier) => (
            <Button
              key={tier}
              variant='outline'
              className='h-20 flex-col gap-1'
              disabled={!props.purchaseLinks[tier] || !user}
              onClick={() => {
                if (user) beginCreditPurchase(tier, user.id)
              }}
              render={
                <a
                  href={props.purchaseLinks[tier] || undefined}
                  target='_blank'
                  rel='noopener noreferrer'
                />
              }
            >
              <span className='text-xl font-semibold tabular-nums'>
                {tier === 'other' ? t('Other') : tier}
              </span>
              <span className='text-muted-foreground inline-flex items-center gap-1 text-xs'>
                {props.purchaseLinks[tier]
                  ? t('Buy credits')
                  : t('Not available')}
                {props.purchaseLinks[tier] && (
                  <ExternalLink data-icon='inline-end' aria-hidden />
                )}
              </span>
            </Button>
          ))}
        </div>
        <Button
          size='xl'
          variant='outline'
          onClick={() => {
            if (user) beginCreditPurchase('other', user.id)
          }}
          disabled={!user}
          className='sm:self-start'
        >
          {t('Already purchased? Redeem here')}
        </Button>
        <p className='text-muted-foreground text-sm leading-relaxed'>
          {t(
            'After purchase, return to this site. The redemption dialog will be ready for you to paste the code from your delivery details.'
          )}
        </p>
      </div>

      <div className='space-y-3'>
        <div className='flex items-center gap-2'>
          <Gift className='text-muted-foreground size-4' aria-hidden />
          <Label htmlFor='redemption-code' className='text-sm font-medium'>
            {t('Already have a code?')}
          </Label>
        </div>
        <div className='grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]'>
          <Input
            id='redemption-code'
            value={props.redemptionCode}
            onChange={(e) => props.onRedemptionCodeChange(e.target.value)}
            placeholder={t('Paste the redemption code you received')}
            autoComplete='off'
            className='h-12 min-w-0'
          />
          <Button
            onClick={props.onRedeem}
            disabled={props.redeeming || !props.redemptionCode.trim()}
            variant='outline'
            size='xl'
          >
            {props.redeeming && (
              <Loader2 className='animate-spin' aria-hidden />
            )}
            {t('Redeem credits')}
          </Button>
        </div>
      </div>
    </section>
  )
}
