/**
 * usePagination — generic client-side pagination for any list.
 *
 * Usage:
 *   const { paged, current, pageSize, total, onChange } = usePagination(filteredList, 10)
 */
export const usePagination = <T>(
  source: Ref<T[]> | ComputedRef<T[]>,
  defaultPageSize = 10,
) => {
  const current = ref(1)
  const pageSize = ref(defaultPageSize)

  const total = computed(() => unref(source).length)

  // Reset to page 1 when source data changes
  watch(source, () => {
    if (current.value > Math.ceil(total.value / pageSize.value)) {
      current.value = 1
    }
  })

  const paged = computed(() => {
    const start = (current.value - 1) * pageSize.value
    return unref(source).slice(start, start + pageSize.value)
  })

  const onChange = (page: number, size: number) => {
    current.value = page
    pageSize.value = size
  }

  return { paged, current, pageSize, total, onChange }
}
