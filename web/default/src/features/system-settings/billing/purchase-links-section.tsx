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
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import {
  CREDIT_PURCHASE_TIERS,
  normalizeCreditPurchaseLinks,
  type CreditPurchaseTier,
} from '@/lib/credit-purchase'
import { Button } from '@/components/ui/button'
import { FieldGroup } from '@/components/ui/field'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

function isPurchaseLink(value: string) {
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
}

const link = z
  .string()
  .trim()
  .max(2048)
  .refine(isPurchaseLink, 'Enter a valid http or https purchase link')
const schema = z.object({
  tier10: link,
  tier20: link,
  tier50: link,
  tier100: link,
  tier200: link,
  other: link,
})
type Values = z.infer<typeof schema>
const fieldName = (tier: CreditPurchaseTier) =>
  (tier === 'other' ? 'other' : `tier${tier}`) as keyof Values

interface PurchaseLinksSectionProps {
  value: string
  legacyLink: string
}

export function PurchaseLinksSection(props: PurchaseLinksSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const links = normalizeCreditPurchaseLinks(props.value, props.legacyLink)
  const { form, handleSubmit, isDirty, isSubmitting } = useSettingsForm<Values>(
    {
      resolver: zodResolver(schema),
      defaultValues: {
        tier10: links['10'] || '',
        tier20: links['20'] || '',
        tier50: links['50'] || '',
        tier100: links['100'] || '',
        tier200: links['200'] || '',
        other: links.other || '',
      },
      onSubmit: async (values) => {
        const purchaseLinks = Object.fromEntries(
          CREDIT_PURCHASE_TIERS.map((tier) => [tier, values[fieldName(tier)]])
        )
        await updateOption.mutateAsync({
          key: 'payment_setting.purchase_links',
          value: JSON.stringify(purchaseLinks),
        })
      },
    }
  )

  return (
    <SettingsSection
      title={t('Recharge tier links')}
      description={t(
        'Set a separate purchase link for each of the six recharge tiers.'
      )}
    >
      <FormNavigationGuard when={isDirty} />
      <Form {...form}>
        <form
          onSubmit={handleSubmit}
          autoComplete='off'
          className='flex flex-col gap-6'
        >
          <FieldGroup className='grid gap-5 lg:grid-cols-2'>
            {CREDIT_PURCHASE_TIERS.map((tier) => (
              <FormField
                key={tier}
                control={form.control}
                name={fieldName(tier)}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('{{tier}} tier purchase link', {
                        tier: tier === 'other' ? t('Other') : tier,
                      })}
                    </FormLabel>
                    <FormControl>
                      <Input type='url' placeholder='https://...' {...field} />
                    </FormControl>
                    <FormDescription>
                      {t('Leave blank to disable purchases for this tier.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ))}
          </FieldGroup>
          <p className='text-muted-foreground text-sm leading-relaxed'>
            {t(
              'Selecting a tier opens its link in a new tab. The original page shows the redemption dialog when the user returns. Credits are added only after a valid code is redeemed.'
            )}
          </p>
          <Button
            type='submit'
            disabled={!isDirty || isSubmitting}
            className='self-start'
          >
            {isSubmitting ? t('Saving...') : t('Save Changes')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
