/**
 * usePagination — 通用客户端分页工具。
 *
 * 接收响应式数据源，返回当前页数据切片及分页控制方法。
 *
 * 用法示例：
 *   const { paged, current, pageSize, total, onChange } = usePagination(filteredList, 10)
 */
export const usePagination = <T>(
  source: Ref<T[]> | ComputedRef<T[]>,
  defaultPageSize = 10,
) => {
  /** 当前页码（从 1 开始） */
  const current = ref(1)
  /** 每页条数 */
  const pageSize = ref(defaultPageSize)

  /** 数据总条数（由数据源长度计算） */
  const total = computed(() => unref(source).length)

  // 数据源变化时，若当前页超出范围则重置到第 1 页
  watch(source, () => {
    if (current.value > Math.ceil(total.value / pageSize.value)) {
      current.value = 1
    }
  })

  /** 当前页的数据切片 */
  const paged = computed(() => {
    const start = (current.value - 1) * pageSize.value
    return unref(source).slice(start, start + pageSize.value)
  })

  /**
   * 分页变化回调，供分页组件的 onChange 事件绑定。
   * @param page 目标页码
   * @param size 目标每页条数
   */
  const onChange = (page: number, size: number) => {
    current.value = page
    pageSize.value = size
  }

  return { paged, current, pageSize, total, onChange }
}
