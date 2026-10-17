/** 只接管嵌入审核文档；主页面及正文横向滚动条均在滚动停止后隐藏。 */
export function useEmbedAuditScrollbars() {
  const timers = new Map<HTMLElement, ReturnType<typeof setTimeout>>()
  const onScroll = (event: Event) => {
    const target = event.target === document ? document.documentElement : event.target
    if (!(target instanceof HTMLElement)) return
    target.setAttribute('data-audit-scrolling', '')
    const timer = timers.get(target)
    if (timer) clearTimeout(timer)
    timers.set(target, setTimeout(() => {
      target.removeAttribute('data-audit-scrolling')
      timers.delete(target)
    }, 800))
  }
  onMounted(() => {
    document.documentElement.classList.add('embed-audit-document')
    document.addEventListener('scroll', onScroll, { passive: true, capture: true })
  })
  onBeforeUnmount(() => {
    document.removeEventListener('scroll', onScroll, true)
    for (const [element, timer] of timers) {
      clearTimeout(timer)
      element.removeAttribute('data-audit-scrolling')
    }
    timers.clear()
    document.documentElement.classList.remove('embed-audit-document')
  })
}
