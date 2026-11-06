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
import { describe, expect, it, spyOn } from 'bun:test'
import {
  getAnnouncementKey, getNoticeKey, getPublishedNotifications,
  getRecentUnreadNotifications, getSnapshotAnnouncements,
  type NotificationFeed,
} from '../src/lib/notifications'
import {
  createNotificationStore, getNotificationViewerKey, EMPTY_NOTIFICATION_READ_STATE,
} from '../src/stores/notification-store'

const now = Date.parse('2026-11-06T12:00:00Z')
const hour = 3_600_000
const ann = (id: number, age = hour, content = `Notification ${id}`) => ({
  id, content, publishDate: new Date(now - age).toISOString(),
})
function feed(announcements = [ann(1)], notice = '', publishedAt = (now - hour) / 1000): NotificationFeed {
  return { server_time: now, notice: { content: notice, published_at: publishedAt }, announcements }
}
function memoryStorage() {
  const values = new Map<string, string>()
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => { values.set(key, value) },
    removeItem: (key: string) => { values.delete(key) },
  }
}
const empty = EMPTY_NOTIFICATION_READ_STATE

describe('notification publication window and snapshots', () => {
  it('groups both sources into one recent batch, ordered by publication time', () => {
    const result = getRecentUnreadNotifications(feed([ann(1), ann(2, 60_000)], 'Site notice'), empty)!
    expect(result.announcements).toHaveLength(2)
    expect(result.notice.content).toBe('Site notice')
    expect(getSnapshotAnnouncements(result)[0].content).toBe('Notification 2')
  })
  it('uses the server clock even when a client clock is wrong', () => {
    const clock = spyOn(Date, 'now').mockReturnValue(now + 365 * 24 * hour)
    try {
      expect(getRecentUnreadNotifications(feed(), empty)).not.toBeNull()
    } finally {
      clock.mockRestore()
    }
  })
  it('accepts a publication just under 24 hours old and expires exactly at 24 hours', () => {
    expect(getRecentUnreadNotifications(feed([ann(1, 24 * hour - 1)]), empty)).not.toBeNull()
    expect(getRecentUnreadNotifications(feed([ann(1, 24 * hour)]), empty)).toBeNull()
    expect(getRecentUnreadNotifications(feed([], 'Site notice', (now - 24 * hour) / 1000), empty)).toBeNull()
  })
  it('does not auto-open legacy notices without a publication timestamp', () => {
    expect(getRecentUnreadNotifications(feed([], 'Legacy notice', 0), empty)).toBeNull()
    expect(getPublishedNotifications(feed([], 'Legacy notice', 0)).notice.content).toBe('Legacy notice')
  })
  it('does not expose scheduled announcements before publication, then opens them when due', () => {
    const scheduled = feed([ann(1, -60_000)], 'Future notice', (now + 60_000) / 1000)
    expect(getPublishedNotifications(scheduled).announcements).toHaveLength(0)
    expect(getPublishedNotifications(scheduled).notice.content).toBe('')
    expect(getRecentUnreadNotifications(scheduled, empty)).toBeNull()
    expect(getRecentUnreadNotifications({ ...scheduled, server_time: now + 60_000 }, empty)).not.toBeNull()
  })
  it('keeps malformed dates out of automatic reminders', () => {
    const invalid = { ...ann(1), publishDate: 'invalid' }
    expect(getRecentUnreadNotifications(feed([invalid]), empty)).toBeNull()
  })
  it('recognizes edited content and reused IDs as different versions', () => {
    const original = ann(1)
    const read = { ...empty, readAnnouncementKeys: [getAnnouncementKey(original)] }
    expect(getRecentUnreadNotifications(feed([original]), read)).toBeNull()
    expect(getRecentUnreadNotifications(feed([ann(1, hour, 'Updated')]), read)).not.toBeNull()
    expect(getRecentUnreadNotifications(feed([ann(1, 60_000)]), read)).not.toBeNull()
  })
  it('distinguishes a republished identical site notice by publication time', () => {
    const old = feed([], 'Same text').notice
    const read = { ...empty, readNoticeKeys: [getNoticeKey(old)] }
    expect(getRecentUnreadNotifications(feed([], 'Same text'), read)).toBeNull()
    expect(getRecentUnreadNotifications(feed([], 'Same text', now / 1000), read)).not.toBeNull()
  })
  it('never truncates new notifications behind a twenty-item history limit', () => {
    expect(getRecentUnreadNotifications(feed(Array.from({ length: 35 }, (_, i) => ann(i))), empty)!.announcements).toHaveLength(35)
  })
  it('keeps open snapshots unchanged when the underlying feed changes', () => {
    const source = feed()
    const snapshot = getRecentUnreadNotifications(source, empty)!
    source.announcements[0].content = 'Changed while open'
    source.announcements.push(ann(2))
    expect(snapshot.announcements).toHaveLength(1)
    expect(snapshot.announcements[0].content).toBe('Notification 1')
  })
})

describe('close, new publication, account and storage regressions', () => {
  it.each([false, true])('closing a batch (today=%s) never suppresses a later publication', today => {
    const state = createNotificationStore(memoryStorage()).getState()
    const snapshot = getRecentUnreadNotifications(feed([ann(1)], 'First notice'), empty)!
    state.acknowledgeSnapshot('user:2', snapshot, today, now)
    const read = state.getReadState('user:2')
    expect(getRecentUnreadNotifications(feed([ann(1)], 'First notice'), read)).toBeNull()
    const next = getRecentUnreadNotifications(feed([ann(1), ann(2, 1000)], 'Second notice', now / 1000), read)!
    expect(next.announcements.map(item => item.id)).toEqual([2])
    expect(next.notice.content).toBe('Second notice')
  })
  it('a publication arriving while a dialog is open is not acknowledged with the old batch', () => {
    const state = createNotificationStore(memoryStorage()).getState()
    const original = getRecentUnreadNotifications(feed(), empty)!
    const latest = feed([ann(1), ann(2, 1000)])
    state.acknowledgeSnapshot('user:2', original)
    expect(getRecentUnreadNotifications(latest, state.getReadState('user:2'))!.announcements[0].id).toBe(2)
  })
  it('remains closed after refresh and across midnight, while a new batch still opens', () => {
    const storage = memoryStorage()
    const first = createNotificationStore(storage).getState()
    first.acknowledgeSnapshot('user:2', getRecentUnreadNotifications(feed(), empty)!, true, now)
    const restored = createNotificationStore(storage).getState()
    const nextDay = { ...feed([ann(2, -12 * hour)]), server_time: now + 13 * hour }
    expect(getRecentUnreadNotifications(feed(), restored.getReadState('user:2'))).toBeNull()
    expect(getRecentUnreadNotifications(nextDay, restored.getReadState('user:2'))).not.toBeNull()
  })
  it('isolates administrator, ordinary accounts and guest acknowledgements', () => {
    const state = createNotificationStore(memoryStorage()).getState()
    const snapshot = getRecentUnreadNotifications(feed(), empty)!
    state.acknowledgeSnapshot(getNotificationViewerKey(1), snapshot, true, now)
    for (const viewer of [getNotificationViewerKey(2), getNotificationViewerKey(3), getNotificationViewerKey(null)]) {
      expect(getRecentUnreadNotifications(feed(), state.getReadState(viewer))).not.toBeNull()
    }
  })
  it('merges writes from two tabs instead of overwriting the other tabs read versions', () => {
    const storage = memoryStorage()
    const a = createNotificationStore(storage)
    const b = createNotificationStore(storage)
    a.getState().acknowledgeSnapshot('user:2', getRecentUnreadNotifications(feed([ann(1)]), empty)!)
    b.getState().acknowledgeSnapshot('user:2', getRecentUnreadNotifications(feed([ann(2)]), empty)!)
    const restored = createNotificationStore(storage)
    expect(getRecentUnreadNotifications(feed([ann(1), ann(2)]), restored.getState().getReadState('user:2'))).toBeNull()
  })
  it('still works when persistent storage is unavailable or full', () => {
    const broken = { getItem() { throw new Error('blocked') }, setItem() { throw new Error('full') }, removeItem() {} }
    const state = createNotificationStore(broken).getState()
    state.acknowledgeSnapshot('user:2', getRecentUnreadNotifications(feed(), empty)!)
    expect(getRecentUnreadNotifications(feed(), state.getReadState('user:2'))).toBeNull()
  })
})
