/**
 * 对话消息中的 Generative UI 结构化块（:::form-confirm / :::chart）解析与归一化。
 *
 * - parseChatChunks：ChatThread 渲染消息时调用。按行解析块标记，容忍模型常见的格式偏差
 *   （代码块围栏包裹、form_confirm 别名、未闭合块、尾随逗号），解析失败时返回 invalid-block，
 *   由调用方显式提示，而不是静默退化成一段 JSON 文本。
 * - normalizeFormConfirmPayload / normalizeChartPayload：卡片组件调用。把模型的“常见写法”
 *   映射到组件真实契约，避免缺 name 导致字段串值、details 明细被丢弃、indicator 写成单数导致雷达图空白等问题。
 *
 * 后端协议正文见 go-service/internal/service/prompt_ui_protocols.go，两边需保持同步。
 */

export type ChatBlockType = 'form-confirm' | 'chart'

export interface FormConfirmOption {
  label: string
  value: string | number
}

export type FormConfirmFieldType = 'text' | 'number' | 'date' | 'select' | 'textarea' | 'browser'

export interface FormConfirmField {
  name: string
  label: string
  type: FormConfirmFieldType
  value?: any
  display_value?: string
  disabled?: boolean
  options?: FormConfirmOption[]
}

export interface FormConfirmTable {
  title?: string
  columns: any[]
  rows: Record<string, any>[]
}

export type FormConfirmVariant = 'summary_confirm' | 'compact_form' | 'data_table'

export interface FormConfirmPayload {
  variant: FormConfirmVariant
  workflow_id?: string
  workflow_name?: string
  title?: string
  description?: string
  fields: FormConfirmField[]
  columns: any[]
  rows: Record<string, any>[]
  details: FormConfirmTable[]
}

export type ChartType = 'bar' | 'line' | 'pie' | 'radar'

export interface ChartSeries {
  name: string
  data: any[]
  color?: string
}

export interface ChartIndicator {
  name: string
  max?: number
}

export interface ChartPayload {
  type: ChartType
  title?: string
  subtitle?: string
  categories: string[]
  series: ChartSeries[]
  indicators: ChartIndicator[]
  columns: any[]
  rows: Record<string, any>[]
}

export interface ChatMessageChunk {
  type: 'markdown' | 'form-confirm' | 'chart' | 'invalid-block' | 'pending-block'
  content: string
  data?: FormConfirmPayload | ChartPayload
  blockType?: ChatBlockType
}

// ─────────────────────────────── 通用工具 ───────────────────────────────

function isPlainObject(value: any): value is Record<string, any> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function asArray(value: any): any[] {
  return Array.isArray(value) ? value : []
}

/** 取第一个有内容的值，兼容模型常用的多种键名 */
function firstDefined<T = any>(source: Record<string, any>, keys: string[]): T | undefined {
  for (const key of keys) {
    const value = source?.[key]
    if (value !== undefined && value !== null && value !== '') return value as T
  }
  return undefined
}

function toText(value: any, fallback = ''): string {
  if (value === undefined || value === null) return fallback
  return String(value)
}

function toNumber(value: any): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() !== '' && !Number.isNaN(Number(value))) return Number(value)
  return undefined
}

// ─────────────────────────── 块解析 ───────────────────────────

const BLOCK_OPEN_RE = /^\s*:::\s*(form[-_]?confirm|chart)\s*$/i
const BLOCK_CLOSE_RE = /^\s*:::\s*$/
const CODE_FENCE_RE = /^\s*```[\w-]*\s*$/

function normalizeBlockType(raw: string): ChatBlockType {
  return /^chart$/i.test(raw) ? 'chart' : 'form-confirm'
}

/** 去掉模型习惯性包上的 ```json 围栏与首尾空行 */
function stripCodeFence(lines: string[]): string[] {
  let start = 0
  let end = lines.length
  while (start < end && !lines[start].trim()) start++
  while (end > start && !lines[end - 1].trim()) end--
  const body = lines.slice(start, end)
  if (body.length && CODE_FENCE_RE.test(body[0])) body.shift()
  if (body.length && CODE_FENCE_RE.test(body[body.length - 1])) body.pop()
  return body
}

function safeParseJson(raw: string): { ok: true; value: any } | { ok: false } {
  if (!raw.trim()) return { ok: false }
  try {
    return { ok: true, value: JSON.parse(raw) }
  } catch {
    // 容忍尾随逗号（模型高频小毛病）
    const cleaned = raw.replace(/,\s*([}\]])/g, '$1')
    try {
      return { ok: true, value: JSON.parse(cleaned) }
    } catch {
      return { ok: false }
    }
  }
}

/**
 * 把消息正文切分为 Markdown 片段与结构化卡片块。
 * 未闭合的块（流式输出中）返回 pending-block，避免每帧闪一次解析失败提示。
 */
export function parseChatChunks(content: string): ChatMessageChunk[] {
  const text = typeof content === 'string' ? content : ''
  if (!text.trim()) return []

  const lines = text.split(/\r?\n/)
  const chunks: ChatMessageChunk[] = []
  let buffer: string[] = []

  const flushMarkdown = () => {
    const markdown = buffer.join('\n').trim()
    if (markdown) chunks.push({ type: 'markdown', content: markdown })
    buffer = []
  }

  for (let i = 0; i < lines.length; i++) {
    const opened = lines[i].match(BLOCK_OPEN_RE)
    if (!opened) {
      buffer.push(lines[i])
      continue
    }

    let closeIndex = -1
    for (let j = i + 1; j < lines.length; j++) {
      if (BLOCK_CLOSE_RE.test(lines[j])) {
        closeIndex = j
        break
      }
      // 缺少结束标记又遇到下一个开始标记：截断当前块，保持整洁
      if (BLOCK_OPEN_RE.test(lines[j])) break
    }

    const blockType = normalizeBlockType(opened[1])
    const bodyLines = lines.slice(i + 1, closeIndex === -1 ? lines.length : closeIndex)
    flushMarkdown()

    if (closeIndex === -1) {
      chunks.push({ type: 'pending-block', content: stripCodeFence(bodyLines).join('\n').trim(), blockType })
      break
    }

    const jsonText = stripCodeFence(bodyLines).join('\n').trim()
    const parsed = safeParseJson(jsonText)
    if (parsed.ok) {
      chunks.push({
        type: blockType,
        content: jsonText,
        blockType,
        data: blockType === 'chart' ? normalizeChartPayload(parsed.value) : normalizeFormConfirmPayload(parsed.value),
      })
    } else {
      chunks.push({ type: 'invalid-block', content: jsonText, blockType })
    }
    i = closeIndex
  }

  flushMarkdown()
  return chunks
}

// ─────────────────────── 表单确认卡片归一化 ───────────────────────

const FIELD_TYPE_ALIASES: Record<string, FormConfirmFieldType> = {
  text: 'text',
  input: 'text',
  string: 'text',
  varchar: 'text',
  singleline: 'text',
  'single-line': 'text',
  textarea: 'textarea',
  'text-area': 'textarea',
  multiline: 'textarea',
  memo: 'textarea',
  remark: 'textarea',
  longtext: 'textarea',
  number: 'number',
  numeric: 'number',
  int: 'number',
  integer: 'number',
  float: 'number',
  decimal: 'number',
  amount: 'number',
  money: 'number',
  date: 'date',
  datetime: 'date',
  'date-time': 'date',
  datepicker: 'date',
  select: 'select',
  dropdown: 'select',
  combo: 'select',
  radio: 'select',
  checkbox: 'select',
  enum: 'select',
  browser: 'browser',
  browse: 'browser',
  person: 'browser',
  user: 'browser',
  department: 'browser',
  org: 'browser',
  reference: 'browser',
}

function normalizeFieldType(raw: any): FormConfirmFieldType {
  const key = toText(raw, '').trim().toLowerCase().replace(/\s+/g, '')
  return FIELD_TYPE_ALIASES[key] || 'text'
}

function normalizeFieldOptions(raw: any): FormConfirmOption[] | undefined {
  const items = asArray(raw)
  if (!items.length) return undefined
  return items.map(item => {
    if (isPlainObject(item)) {
      const value = firstDefined(item, ['value', 'id', 'key', 'code', 'val', 'label', 'name', 'text'])
      const label = firstDefined(item, ['label', 'name', 'text', 'title', 'value', 'id'])
      return {
        label: toText(label ?? value),
        value: typeof value === 'number' ? value : toText(value ?? label),
      }
    }
    return { label: toText(item), value: typeof item === 'number' ? item : toText(item) }
  })
}

/** 日期字段统一成 YYYY-MM-DD，避免日期选择器因带时间/斜杠而显示为空 */
function normalizeDateValue(value: any): any {
  if (typeof value !== 'string') return value
  const trimmed = value.trim()
  const matched = trimmed.match(/^(\d{4})[-/](\d{1,2})[-/](\d{1,2})/)
  if (!matched) return trimmed
  const [, year, month, day] = matched
  return `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`
}

function normalizeField(raw: any, index: number): FormConfirmField {
  const source = isPlainObject(raw) ? raw : { value: raw }
  const name = toText(firstDefined(source, ['name', 'field', 'field_name', 'fieldName', 'code', 'key']), '')
  const label = toText(firstDefined(source, ['label', 'title', 'display_name', 'displayName', 'name_cn', 'cn_name']), '')
  const type = normalizeFieldType(firstDefined(source, ['type', 'field_type', 'fieldType', 'input_type']))
  const options = normalizeFieldOptions(firstDefined(source, ['options', 'choices', 'enum', 'candidates', 'items']))

  let value = firstDefined(source, ['value', 'default', 'default_value', 'defaultValue', 'val'])
  if (type === 'date') value = normalizeDateValue(value)
  // 模型常把下拉选项的显示名写进 value，这里按 label 反查回真实取值
  if (type === 'select' && options?.length && value !== undefined && value !== null) {
    const matchedOption = options.find(option => String(option.label) === String(value))
    const alreadyValid = options.some(option => String(option.value) === String(value))
    if (matchedOption && !alreadyValid) value = matchedOption.value
  }

  return {
    // 兜底 name：缺失时用位置生成稳定键，避免所有字段共用 undefined 键导致串值
    name: name || `field_${index}`,
    label: label || name || `字段${index + 1}`,
    type,
    value: value ?? '',
    display_value: toText(firstDefined(source, ['display_value', 'displayValue', 'display_name', 'displayName', 'value_text']), '') || undefined,
    disabled: Boolean(firstDefined(source, ['disabled', 'readonly', 'read_only', 'readOnly'])),
    options,
  }
}

function normalizeColumns(raw: any): any[] {
  return asArray(raw).map((column, index) => {
    if (typeof column === 'string') {
      return {
        title: column,
        dataIndex: column,
        key: column,
        sorter: (a: any, b: any) => {
          const left = a?.[column] ?? ''
          const right = b?.[column] ?? ''
          if (typeof left === 'number' && typeof right === 'number') return left - right
          return String(left).localeCompare(String(right))
        },
      }
    }
    if (isPlainObject(column)) {
      const dataIndex = toText(firstDefined(column, ['dataIndex', 'data_index', 'key', 'field', 'name', 'prop']), `col_${index}`)
      return {
        ...column,
        title: toText(firstDefined(column, ['title', 'label', 'name', 'header']), dataIndex),
        dataIndex,
        key: toText(firstDefined(column, ['key', 'dataIndex', 'data_index']), dataIndex),
      }
    }
    return { title: `列${index + 1}`, dataIndex: `col_${index}`, key: `col_${index}` }
  })
}

function normalizeRows(raw: any): Record<string, any>[] {
  return asArray(raw).filter(isPlainObject)
}

function normalizeTable(raw: any, fallbackTitle = ''): FormConfirmTable | null {
  if (!isPlainObject(raw)) return null
  const columns = normalizeColumns(firstDefined(raw, ['columns', 'cols', 'header', 'headers', 'fields']))
  const rows = normalizeRows(firstDefined(raw, ['rows', 'data', 'list', 'records', 'items']))
  if (!columns.length && !rows.length) return null
  const title = toText(firstDefined(raw, ['title', 'name', 'label', 'caption']), fallbackTitle) || undefined
  return { title, columns, rows }
}

/** details 支持单对象、对象数组、或 {表名: {columns, rows}} 映射 */
function normalizeDetails(raw: any): FormConfirmTable[] {
  if (!raw) return []
  const candidates = Array.isArray(raw) ? raw : [raw]
  const tables: FormConfirmTable[] = []
  for (const candidate of candidates) {
    if (!isPlainObject(candidate)) continue
    const single = normalizeTable(candidate)
    if (single) {
      tables.push(single)
      continue
    }
    // { "费用明细": { columns, rows } } 形式
    Object.entries(candidate).forEach(([name, value]) => {
      if (!isPlainObject(value)) return
      const table = normalizeTable(value, name)
      if (table) tables.push({ ...table, title: table.title || name })
    })
  }
  return tables
}

function normalizeVariant(raw: any, hasFields: boolean, hasRows: boolean): FormConfirmVariant {
  const key = toText(raw, '').trim().toLowerCase().replace(/\s+/g, '_')
  if (key === 'data_table' || key === 'datatable' || key === 'table' || key === 'grid') return 'data_table'
  if (key === 'compact_form' || key === 'compact' || key === 'form') return 'compact_form'
  if (key === 'summary_confirm' || key === 'summary' || key === 'confirm') return 'summary_confirm'
  // 没有 fields 但有表格数据：按查数表格处理
  if (!hasFields && hasRows) return 'data_table'
  return 'summary_confirm'
}

export function normalizeFormConfirmPayload(raw: any): FormConfirmPayload {
  const source = isPlainObject(raw) ? raw : {}
  const fields = asArray(firstDefined(source, ['fields', 'form_fields', 'formFields', 'items', 'props', 'columns_data'])).map(
    (field, index) => normalizeField(field, index),
  )
  // data_table 的列/行也可能写在 details 里，兼容旧写法的顶层 columns/rows
  const topTable = normalizeTable({
    columns: firstDefined(source, ['columns', 'cols']),
    rows: firstDefined(source, ['rows', 'data', 'list']),
    title: firstDefined(source, ['title']),
  })
  const details = normalizeDetails(firstDefined(source, ['details', 'detail', 'detail_table', 'detailTable', 'sub_tables', 'subTables']))

  const variant = normalizeVariant(firstDefined(source, ['variant', 'card_type', 'cardType']), fields.length > 0, !!topTable?.rows.length)
  const title = toText(firstDefined(source, ['title', 'card_title', 'heading']), '')
  const workflowName = toText(firstDefined(source, ['workflow_name', 'workflowName', 'process_name', 'processName']), '')

  return {
    variant,
    workflow_id: toText(firstDefined(source, ['workflow_id', 'workflowId', 'process_id', 'processId', 'wf_id']), '') || undefined,
    workflow_name: workflowName || title || undefined,
    title: title || workflowName || undefined,
    description: toText(firstDefined(source, ['description', 'desc', 'summary', 'remark', 'intro']), '') || undefined,
    fields,
    columns: topTable?.columns || [],
    rows: topTable?.rows || [],
    details,
  }
}

// ───────────────────────── 图表卡片归一化 ─────────────────────────

const CHART_TYPE_ALIASES: Record<string, ChartType> = {
  bar: 'bar',
  column: 'bar',
  histogram: 'bar',
  柱状图: 'bar',
  条形图: 'bar',
  line: 'line',
  area: 'line',
  trend: 'line',
  折线图: 'line',
  曲线图: 'line',
  pie: 'pie',
  donut: 'pie',
  doughnut: 'pie',
  ring: 'pie',
  饼图: 'pie',
  环形图: 'pie',
  radar: 'radar',
  雷达图: 'radar',
}

function normalizeChartType(raw: any): ChartType | undefined {
  const key = toText(raw, '').trim().toLowerCase()
  return CHART_TYPE_ALIASES[key]
}

function normalizeCategories(raw: any): string[] {
  return asArray(raw).map((item, index) => {
    if (isPlainObject(item)) return toText(firstDefined(item, ['name', 'label', 'title', 'category', 'x']), `维度${index + 1}`)
    return toText(item, `维度${index + 1}`)
  })
}

function normalizePoint(value: any): any {
  if (isPlainObject(value)) {
    const numeric = toNumber(firstDefined(value, ['value', 'count', 'total', 'y', 'num']))
    if (numeric !== undefined) return numeric
    return value
  }
  const numeric = toNumber(value)
  return numeric !== undefined ? numeric : value
}

function normalizeSeriesItem(raw: any, index: number): ChartSeries {
  if (isPlainObject(raw)) {
    const name = toText(firstDefined(raw, ['name', 'label', 'title', 'series_name', 'seriesName']), `系列${index + 1}`)
    // 保留原始点集：饼图的 [{name,value}] 需要先拆出分类再转数值
    const data = asArray(firstDefined(raw, ['data', 'values', 'value', 'points', 'list', 'y']))
    return { name, data, color: firstDefined<string>(raw, ['color']) }
  }
  return { name: `系列${index + 1}`, data: asArray(raw) }
}

function normalizeSeries(raw: any): ChartSeries[] {
  if (!raw) return []
  if (Array.isArray(raw)) {
    // 直接给了数值数组
    if (raw.every(item => typeof item === 'number' || typeof item === 'string')) {
      return [{ name: '数值', data: raw }]
    }
    return raw.map((item, index) => normalizeSeriesItem(item, index))
  }
  if (isPlainObject(raw)) {
    if (['data', 'values', 'value', 'points'].some(key => key in raw)) {
      return [normalizeSeriesItem(raw, 0)]
    }
    // { "市场部": 142, "研发部": 98 } 形式
    const entries = Object.entries(raw).filter(([, value]) => typeof value === 'number' || Array.isArray(value))
    if (entries.length) {
      return entries.map(([name, value], index) => normalizeSeriesItem({ name, data: Array.isArray(value) ? value : [value] }, index))
    }
  }
  return []
}

function normalizeIndicators(raw: any): ChartIndicator[] {
  return asArray(raw).map((item, index) => {
    if (isPlainObject(item)) {
      const name = toText(firstDefined(item, ['name', 'label', 'title', 'indicator']), `维度${index + 1}`)
      const max = toNumber(firstDefined(item, ['max', 'limit', 'maximum', 'value']))
      return { name, max: max && max > 0 ? max : undefined }
    }
    return { name: toText(item, `维度${index + 1}`), max: undefined }
  })
}

/** 从对象点集 [{name,value}] 中拆出分类与数值（饼图常见写法） */
function splitPiePoints(points: any[]): { names: string[]; values: number[] } | null {
  if (!points.length || !points.every(isPlainObject)) return null
  const names = points.map((point, index) => toText(firstDefined(point, ['name', 'label', 'title', 'category']), `分类${index + 1}`))
  const values = points.map(point => toNumber(firstDefined(point, ['value', 'count', 'total', 'num'])) ?? 0)
  return { names, values }
}

export function normalizeChartPayload(raw: any): ChartPayload {
  const source = isPlainObject(raw) ? raw : {}

  // xAxis 可能是数组，也可能是 [{data:[...]}] / {data:[...]}
  const rawXAxis = firstDefined(source, ['xAxis', 'xaxis', 'x_axis', 'labels', 'dimensions', 'dimension'])
  const xAxisArray = Array.isArray(rawXAxis)
    ? (isPlainObject(rawXAxis[0]) && 'data' in rawXAxis[0] ? rawXAxis[0].data : rawXAxis)
    : (isPlainObject(rawXAxis) ? rawXAxis.data : undefined)

  let categories = normalizeCategories(
    firstDefined(source, ['categories', 'category', 'cats']) ?? xAxisArray,
  )
  let series = normalizeSeries(firstDefined(source, ['series', 'data', 'dataset', 'datasets']))
  let indicators = normalizeIndicators(firstDefined(source, ['indicators', 'indicator', 'axes', 'metrics']))

  let type = normalizeChartType(firstDefined(source, ['type', 'chart_type', 'chartType', 'kind']))
  if (!type) {
    if (indicators.length) type = 'radar'
    else if (series.some(item => item.data.some(isPlainObject))) type = 'pie'
    else type = 'bar'
  }

  // 饼图：只给了 [{name,value}] 对象数组时，拆成 categories + 数值数组
  if (type === 'pie' && !categories.length && series.length) {
    const split = splitPiePoints(series[0].data)
    if (split) {
      categories = split.names
      series = [{ ...series[0], data: split.values }, ...series.slice(1)]
    }
  }

  // 统一把点集转成 ECharts 可直接消费的数值（对象点取 value）
  series = series.map(item => ({ ...item, data: item.data.map(normalizePoint) }))

  // 雷达图：指标缺 max 时按数据量级补一个上限；indicators 与 categories 互为兜底
  if (type === 'radar') {
    if (!indicators.length && categories.length) {
      indicators = categories.map(name => ({ name, max: undefined }))
    }
    if (!categories.length && indicators.length) {
      categories = indicators.map(item => item.name)
    }
    const dataMax = Math.max(
      0,
      ...series.flatMap(item => item.data.map(value => toNumber(value) ?? 0)),
    )
    if (dataMax > 0) {
      indicators = indicators.map(item => ({
        name: item.name,
        max: item.max && item.max > 0 ? item.max : Math.ceil(dataMax * 1.2),
      }))
    }
  }

  const table = normalizeTable({
    columns: firstDefined(source, ['columns', 'cols']),
    rows: firstDefined(source, ['rows', 'records', 'list']),
  })

  return {
    type,
    title: toText(firstDefined(source, ['title', 'name', 'chart_title', 'chartTitle']), '') || undefined,
    subtitle: toText(firstDefined(source, ['subtitle', 'sub_title', 'desc', 'description']), '') || undefined,
    categories,
    series,
    indicators,
    columns: table?.columns || [],
    rows: table?.rows || [],
  }
}
