const PASSWORD_GROUPS = [
  'ABCDEFGHJKLMNPQRSTUVWXYZ',
  'abcdefghijkmnopqrstuvwxyz',
  '23456789',
  '!@#$%^&*_-+=?',
]

const PASSWORD_CHARACTERS = PASSWORD_GROUPS.join('')

function secureRandomIndex(maxExclusive: number): number {
  if (!globalThis.crypto?.getRandomValues) {
    throw new Error('Secure random generator is unavailable')
  }

  const randomValue = new Uint32Array(1)
  const range = 0x100000000
  const limit = range - (range % maxExclusive)

  do {
    globalThis.crypto.getRandomValues(randomValue)
  } while (randomValue[0] >= limit)

  return randomValue[0] % maxExclusive
}

function randomCharacter(characters: string): string {
  return characters[secureRandomIndex(characters.length)]
}

/**
 * 生成 32 位高强度密码，并保证包含大写字母、小写字母、数字和特殊字符。
 */
export function generateStrongPassword(): string {
  const password = PASSWORD_GROUPS.map(randomCharacter)

  while (password.length < 32) {
    password.push(randomCharacter(PASSWORD_CHARACTERS))
  }

  for (let index = password.length - 1; index > 0; index -= 1) {
    const targetIndex = secureRandomIndex(index + 1)
    const currentCharacter = password[index]
    password[index] = password[targetIndex]
    password[targetIndex] = currentCharacter
  }

  return password.join('')
}
