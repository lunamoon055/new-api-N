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
import {
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
} from '../src/i18n/languages'
import { resources } from '../src/i18n/config'

describe('interface languages', () => {
  it('only exposes Chinese and English in the interface', () => {
    expect(INTERFACE_LANGUAGE_OPTIONS.map((language) => language.code)).toEqual([
      'zh',
      'en',
    ])
    expect(Object.keys(resources).sort()).toEqual(['en', 'zh'])
  })

  it('falls back removed languages to English', () => {
    expect(normalizeInterfaceLanguage('fr')).toBe('en')
    expect(normalizeInterfaceLanguage('ru')).toBe('en')
    expect(normalizeInterfaceLanguage('ja')).toBe('en')
    expect(normalizeInterfaceLanguage('vi')).toBe('en')
    expect(normalizeInterfaceLanguage('zh-CN')).toBe('zh')
  })
})
