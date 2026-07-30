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

import { safeJsonParse } from '../utils/json-parser'

export type GroupPricingValues = {
  GroupRatio: string
  UserUsableGroups: string
}

export type GroupPricingEchoGuard = {
  expectedSignatures: ReadonlySet<string>
  finalSignature: string
}

export function getGroupPricingSignature(
  groupRatio: string,
  userUsableGroups: string
): string {
  return JSON.stringify({
    groupRatio: safeJsonParse(groupRatio, { fallback: {}, silent: true }),
    userUsableGroups: safeJsonParse(userUsableGroups, {
      fallback: {},
      silent: true,
    }),
  })
}

export function createGroupPricingEchoGuard(
  currentGroupRatio: string,
  currentUserUsableGroups: string,
  nextValues: GroupPricingValues
): GroupPricingEchoGuard {
  const finalSignature = getGroupPricingSignature(
    nextValues.GroupRatio,
    nextValues.UserUsableGroups
  )

  return {
    expectedSignatures: new Set([
      getGroupPricingSignature(nextValues.GroupRatio, currentUserUsableGroups),
      getGroupPricingSignature(currentGroupRatio, nextValues.UserUsableGroups),
      finalSignature,
    ]),
    finalSignature,
  }
}

export function classifyGroupPricingEcho(
  guard: GroupPricingEchoGuard | null,
  groupRatio: string,
  userUsableGroups: string
): { expected: boolean; complete: boolean } {
  if (!guard) {
    return { expected: false, complete: false }
  }

  const signature = getGroupPricingSignature(groupRatio, userUsableGroups)
  return {
    expected: guard.expectedSignatures.has(signature),
    complete: signature === guard.finalSignature,
  }
}
