/**
 * 将文本写入系统剪贴板；在非安全上下文中回退到浏览器兼容复制方案。
 */
export async function writeClipboardText(text: string): Promise<void> {
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // 权限被拒绝时继续尝试兼容方案。
    }
  }

  if (typeof document === 'undefined' || !document.body) {
    throw new Error('Clipboard is unavailable')
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.readOnly = true
  textarea.setAttribute('aria-hidden', 'true')
  textarea.style.position = 'fixed'
  textarea.style.top = '0'
  textarea.style.left = '-9999px'
  textarea.style.opacity = '0'
  textarea.style.pointerEvents = 'none'

  const activeElement = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null

  document.body.appendChild(textarea)
  textarea.focus({ preventScroll: true })
  textarea.select()
  textarea.setSelectionRange(0, textarea.value.length)

  try {
    if (!document.execCommand('copy')) {
      throw new Error('Clipboard copy command failed')
    }
  } finally {
    textarea.remove()
    activeElement?.focus({ preventScroll: true })
  }
}

/**
 * 安全复制文本，返回是否成功，不会抛出异常。
 */
export async function safeCopyText(text: string): Promise<boolean> {
  try {
    await writeClipboardText(text)
    return true
  } catch {
    return false
  }
}

