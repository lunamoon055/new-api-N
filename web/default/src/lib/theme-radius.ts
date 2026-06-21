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
import { useMemo } from 'react'

function getRootFontSize(styles: CSSStyleDeclaration): number {
  const parsedSize = Number.parseFloat(styles.fontSize)
  return Number.isFinite(parsedSize) ? parsedSize : 16
}

function parseCssLengthPx(
  value: string,
  styles: CSSStyleDeclaration,
  seenVariables = new Set<string>()
): number | undefined {
  const trimmedValue = value.trim()
  if (!trimmedValue) return undefined

  if (trimmedValue.startsWith('var(') && trimmedValue.endsWith(')')) {
    const variableName = trimmedValue.slice(4, -1).trim()
    if (!variableName || seenVariables.has(variableName)) return undefined

    const nextSeenVariables = new Set(seenVariables)
    nextSeenVariables.add(variableName)
    return parseCssLengthPx(
      styles.getPropertyValue(variableName),
      styles,
      nextSeenVariables
    )
  }

  const calcMatch = trimmedValue.match(/^calc\((.+)\s*\*\s*([\d.]+)\)$/)
  if (calcMatch) {
    const base = parseCssLengthPx(calcMatch[1], styles, seenVariables)
    const multiplier = Number(calcMatch[2])
    return base !== undefined && Number.isFinite(multiplier)
      ? base * multiplier
      : undefined
  }

  const lengthMatch = trimmedValue.match(/^([\d.]+)(px|rem|em)$/)
  if (!lengthMatch) return undefined

  const numericValue = Number(lengthMatch[1])
  if (!Number.isFinite(numericValue)) return undefined

  if (lengthMatch[2] === 'px') return numericValue
  return numericValue * getRootFontSize(styles)
}

export function resolveThemeRadiusPx(
  cssVariable = '--radius-md'
): number | undefined {
  if (typeof document === 'undefined') return undefined

  const styles = getComputedStyle(document.documentElement)
  return parseCssLengthPx(styles.getPropertyValue(cssVariable), styles)
}

export function useThemeRadiusPx(
  cssVariable = '--radius-md',
  refreshKey?: string
): number | undefined {
  return useMemo(() => {
    void refreshKey
    return resolveThemeRadiusPx(cssVariable)
  }, [cssVariable, refreshKey])
}
