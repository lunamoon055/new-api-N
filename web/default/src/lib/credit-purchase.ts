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
export const CREDIT_PURCHASE_TIERS = [
  '10',
  '20',
  '50',
  '100',
  '200',
  'other',
] as const
export type CreditPurchaseTier = (typeof CREDIT_PURCHASE_TIERS)[number]
export type CreditPurchaseLinks = Partial<Record<CreditPurchaseTier, string>>

export const CREDIT_PURCHASE_EVENT = 'credit-purchase-changed'
export const WALLET_BALANCE_EVENT = 'wallet-balance-updated'
const purchaseStorageKey = 'credit-purchase-pending'
const purchaseLifetime = 24 * 60 * 60 * 1000
let fallbackSnapshot: string | null = null

export interface CreditPurchaseIntent {
  id: string
  tier: CreditPurchaseTier
  userId: number
  createdAt: number
}

export function parseCreditPurchaseIntent(
  raw: string | null,
  now = Date.now()
): CreditPurchaseIntent | null {
  if (!raw) return null
  try {
    const value = JSON.parse(raw) as Partial<CreditPurchaseIntent>
    if (
      typeof value.id !== 'string' ||
      !value.id ||
      typeof value.tier !== 'string' ||
      !CREDIT_PURCHASE_TIERS.includes(value.tier as CreditPurchaseTier) ||
      typeof value.userId !== 'number' ||
      !Number.isInteger(value.userId) ||
      value.userId <= 0 ||
      typeof value.createdAt !== 'number' ||
      !Number.isFinite(value.createdAt) ||
      value.createdAt > now ||
      now - value.createdAt >= purchaseLifetime
    )
      return null
    return value as CreditPurchaseIntent
  } catch {
    return null
  }
}

export function readCreditPurchaseSnapshot(): string | null {
  try {
    return window.localStorage.getItem(purchaseStorageKey)
  } catch {
    return fallbackSnapshot
  }
}

function writeCreditPurchaseSnapshot(value: string | null) {
  fallbackSnapshot = value
  try {
    if (value) window.localStorage.setItem(purchaseStorageKey, value)
    else window.localStorage.removeItem(purchaseStorageKey)
  } catch {
    // The current page still works when browser storage is unavailable.
  }
  window.dispatchEvent(new Event(CREDIT_PURCHASE_EVENT))
}

export function beginCreditPurchase(tier: CreditPurchaseTier, userId: number) {
  const intent: CreditPurchaseIntent = {
    id: crypto.randomUUID(),
    tier,
    userId,
    createdAt: Date.now(),
  }
  // Prepare the original tab before the browser opens the external store.
  // No payment status or redemption code is stored here.
  writeCreditPurchaseSnapshot(JSON.stringify(intent))
}

export function clearCreditPurchase(id?: string) {
  if (id && parseCreditPurchaseIntent(readCreditPurchaseSnapshot())?.id !== id)
    return
  writeCreditPurchaseSnapshot(null)
}

export function subscribeCreditPurchase(listener: () => void) {
  const onStorage = (event: StorageEvent) => {
    if (event.key === purchaseStorageKey || event.key === null) listener()
  }
  window.addEventListener('storage', onStorage)
  window.addEventListener(CREDIT_PURCHASE_EVENT, listener)
  window.addEventListener('focus', listener)
  window.addEventListener('pageshow', listener)
  return () => {
    window.removeEventListener('storage', onStorage)
    window.removeEventListener(CREDIT_PURCHASE_EVENT, listener)
    window.removeEventListener('focus', listener)
    window.removeEventListener('pageshow', listener)
  }
}

export function normalizeCreditPurchaseLinks(
  value: unknown,
  legacyLink = ''
): CreditPurchaseLinks {
  let parsed = value
  if (typeof parsed === 'string') {
    try {
      parsed = JSON.parse(parsed)
    } catch {
      parsed = null
    }
  }
  const links: CreditPurchaseLinks = {}
  const source =
    parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {}
  for (const tier of CREDIT_PURCHASE_TIERS) {
    const candidate = source[tier]
    let link = typeof candidate === 'string' ? candidate.trim() : ''
    if (tier === 'other' && !(tier in source)) link = legacyLink.trim()
    try {
      const url = new URL(link)
      if (
        !['http:', 'https:'].includes(url.protocol) ||
        url.username ||
        url.password
      )
        link = ''
    } catch {
      link = ''
    }
    links[tier] = link
  }
  return links
}
