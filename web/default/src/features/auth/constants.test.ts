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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  REGISTRATION_USERNAME_ERROR_MESSAGE,
  registerFormSchema,
  sanitizeRegistrationUsername,
} from './constants'

const validRegistration = {
  email: '',
  password: 'password123',
  confirmPassword: 'password123',
}

describe('registration username validation', () => {
  test('accepts only English letters and Arabic numerals', () => {
    for (const username of ['Alice', 'alice123', '123456']) {
      const result = registerFormSchema.safeParse({
        ...validRegistration,
        username,
      })

      assert.equal(result.success, true, username)
    }
  })

  test('rejects Chinese characters, spaces, and special characters', () => {
    for (const username of ['用户名', 'alice user', 'alice_123', 'alice@123']) {
      const result = registerFormSchema.safeParse({
        ...validRegistration,
        username,
      })

      assert.equal(result.success, false, username)
      if (!result.success) {
        assert.equal(
          result.error.issues.find((issue) => issue.path[0] === 'username')
            ?.message,
          REGISTRATION_USERNAME_ERROR_MESSAGE
        )
      }
    }
  })

  test('removes unsupported characters before updating the input', () => {
    assert.equal(sanitizeRegistrationUsername('A中_1-2'), 'A12')
  })
})
