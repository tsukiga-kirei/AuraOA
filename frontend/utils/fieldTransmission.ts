import type { FieldTransmissionConfig } from '~/types/common'

interface FieldReference {
  source: string
  field_key: string
}

const fieldReferenceKey = (field: FieldReference) => `${field.source}:${field.field_key}`.toLowerCase()

/** 查询单字段空值设置；统一开关只覆盖执行效果，不改写单字段设置。 */
export function isEmptyFieldSkipped(config: FieldTransmissionConfig, field: FieldReference): boolean {
  return (config.skip_empty_field_keys || []).some(key => key.toLowerCase() === fieldReferenceKey(field))
}

/** 关闭时省略缺省值，避免旧配置仅因补充 false 而显示草稿已修改。 */
export function setSkipEmptyFields(config: FieldTransmissionConfig, enabled: boolean): void {
  if (enabled) config.skip_empty_fields = true
  else delete config.skip_empty_fields
}

/** 按表与字段共同定位，保留其他字段的设置，稳定排序以减少无意义的内容变化。 */
export function setEmptyFieldSkipped(config: FieldTransmissionConfig, field: FieldReference, enabled: boolean): void {
  const keys = new Set((config.skip_empty_field_keys || []).map(key => key.toLowerCase()))
  if (enabled) keys.add(fieldReferenceKey(field))
  else keys.delete(fieldReferenceKey(field))
  if (keys.size) config.skip_empty_field_keys = [...keys].sort()
  else delete config.skip_empty_field_keys
}
