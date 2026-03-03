/**
 * usePagination — 任何列表的通用客户端分页。
 *
 * 用法：
 * const { 分页，当前，pageSize，总计，onChange } = usePagination(filteredList, 10)*/
export const usePagination = <T>(
  source: Ref<T[]> | ComputedRef<T[]>,
  defaultPageSize = 10,
) => {
  const current = ref(1)
  const pageSize = ref(defaultPageSize)

  const total = computed(() => unref(source).length)

  //源数据更改时重置为第 1 页
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
