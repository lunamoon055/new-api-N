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
import type { Channel } from '../types'

export function supportsChannelCapabilitySync(
  channel: Pick<Channel, 'type' | 'base_url'>
): boolean {
  if ([59, 60].includes(channel.type)) return true
  if (![1, 55].includes(channel.type) || !channel.base_url) return false
  try {
    return ['suanliai.top', 'api.meaicc.com'].includes(
      new URL(channel.base_url).hostname.toLowerCase()
    )
  } catch {
    return false
  }
}
