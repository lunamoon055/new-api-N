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
import { describe, expect, it } from 'bun:test'
import { createNotificationStore } from '../src/stores/notification-store'
import { getRecentUnreadNotifications } from '../src/lib/notifications'
function storageWith(state: unknown, version: number) {
  let value = JSON.stringify({ state, version })
  return {
    getItem: () => value,
    setItem: (_key: string, next: string) => { value = next },
    removeItem: () => { value = '' },
  }
}
describe('notification state migration and synchronization', () => {
  it('does not assign old globally shared flags to any account', () => {
    const store = createNotificationStore(storageWith({ lastReadNotice: 'old', closedUntilDate: new Date().toDateString() }, 0))
    for (const key of ['user:1', 'user:2', 'guest']) {
      expect(store.getState().getReadState(key).readNoticeKeys).toEqual([])
      expect(store.getState().getReadState(key).closedNotificationKeys).toEqual([])
    }
  })
  it('migrates per-user read versions but drops the old blanket close-today effect', () => {
    const store = createNotificationStore(storageWith({ readByViewer: {
      'user:2': { lastReadNotice: 'old', readAnnouncementKeys: ['read'], closedUntilDate: new Date().toDateString() },
    } }, 1))
    const state = store.getState().getReadState('user:2')
    expect(state.readAnnouncementKeys).toEqual(['read'])
    expect(getRecentUnreadNotifications({ server_time: Date.now(), notice: { content: 'new', published_at: Math.floor(Date.now()/1000) }, announcements: [] }, state)).not.toBeNull()
  })
  it('rehydrates another tab read state without losing local account state', async () => {
    const storage = storageWith({}, 2)
    const first = createNotificationStore(storage)
    const second = createNotificationStore(storage)
    first.getState().markAnnouncementsRead('user:1', ['a'])
    second.getState().markAnnouncementsRead('user:2', ['b'])
    await first.persist.rehydrate()
    expect(first.getState().getReadState('user:1').readAnnouncementKeys).toEqual(['a'])
    expect(first.getState().getReadState('user:2').readAnnouncementKeys).toEqual(['b'])
  })
  it('ignores corrupted persisted input rather than breaking the page', () => {
    const store = createNotificationStore(storageWith({ readByViewer: { 'user:2': { readNoticeKeys: [null, 1, 'valid'], readAnnouncementKeys: 'invalid' } } }, 2))
    expect(store.getState().getReadState('user:2').readNoticeKeys).toEqual(['valid'])
    expect(store.getState().getReadState('user:2').readAnnouncementKeys).toEqual([])
  })
})
