/** 比较页面可保存内容，忽略对象键序和服务端只读元数据。 */
export function configFormFingerprint(module: string, config: Record<string, any>): string {
  const fields = ['process_type', 'process_type_label', 'main_table_name', 'main_fields', 'detail_tables', 'status']
  if (module === 'summary') fields.push('summary_blocks', 'embed_enabled', 'embed_config')
  else fields.push('field_mode', 'kb_mode', 'ai_config', 'user_permissions', 'access_control')
  if (module === 'audit') fields.push('embed_enabled', 'embed_config')
  const normalize = (value: any): any => Array.isArray(value) ? value.map(normalize) : value && typeof value === 'object'
    ? Object.fromEntries(Object.keys(value).sort().filter(key => value[key] !== undefined).map(key => [key, normalize(value[key])])) : value
  return JSON.stringify(normalize(Object.fromEntries(fields.map(key => [key, config[key]]))))
}

/** 保存成功后才发布；未修改时跳过保存，由发布接口去重已有版本。 */
export async function publishAfterSavingChanges<T>(changed: boolean, save: () => Promise<boolean>, publish: () => Promise<T>): Promise<T | undefined> {
  if (changed && !await save()) return undefined
  return await publish()
}
