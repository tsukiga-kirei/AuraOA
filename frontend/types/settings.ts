// types/settings.ts — 系统设置相关接口类型

/**
 * 后端 /api/system/configs 返回的单条 KV 配置项
 */
export interface ConfigItem {
    key: string
    value: string
    remark: string
}

/**
 * 后端 /api/system/configs 的列表响应
 */
export interface ConfigListResponse {
    code: number
    message: string
    data: ConfigItem[]
}

/**
 * 保存配置时的请求体：key -> 新值字符串
 */
export type ConfigUpdateRequest = Record<string, string>

/**
 * 前端「通用配置」表单模型
 * 字段与后端 system_configs 表 key 的映射关系见下方注释
 */
export interface SystemGeneralConfig {
    // ===== 平台基本信息 =====
    /** system.name */
    platform_name: string
    /** system.version */
    platform_version: string
    /** system.default_language */
    default_language: string

    // ===== 认证 & 安全 =====
    /** auth.login_fail_lock_threshold — 登录失败锁定阈值（次） */
    login_fail_lock_threshold: number
    /** auth.account_lock_minutes — 账户锁定时长（分钟） */
    account_lock_minutes: number
    /** auth.access_token_ttl_hours — Access Token 有效期（小时） */
    access_token_ttl_hours: number
    /** auth.refresh_token_ttl_days — Refresh Token 有效期（天） */
    refresh_token_ttl_days: number
    /** auth.default_password — 新建成员默认密码 */
    default_password: string

    // ===== 配额与策略 =====
    /** tenant.default_token_quota — 租户默认 Token 配额 */
    tenant_default_token_quota: number
    /** tenant.default_max_concurrency — 租户默认最大并发数 */
    tenant_default_max_concurrency: number
    /** tenant.default_log_retention_days — 租户默认日志保留天数（0 表示不保留备份） */
    tenant_default_log_retention_days: number
    /** tenant.default_data_retention_days — 租户默认审核数据保留天数 */
    tenant_default_data_retention_days: number
    /** system.global_log_retention_days — 全局系统日志保留天数（0 表示不保留备份） */
    global_log_retention_days: number

    // ===== 安全开关 =====
    /** system.enable_audit_trail */
    enable_audit_trail: boolean
    /** system.enable_data_encryption */
    enable_data_encryption: boolean

    // ===== 备份 =====
    /** system.backup_enabled */
    backup_enabled: boolean
    /** system.backup_cron */
    backup_cron: string
    /** system.backup_retention_days */
    backup_retention_days: number

    // ===== 邮件通知 =====
    /** system.notification_email */
    notification_email: string
    /** system.smtp_host */
    smtp_host: string
    /** system.smtp_port */
    smtp_port: number
    /** system.smtp_username */
    smtp_username: string
    /** system.smtp_ssl */
    smtp_ssl: boolean
    /** system.smtp_password */
    smtp_password?: string
    /** system.smtp_sender */
    smtp_sender?: string

    // ===== 附件解析（attachment.*） =====
    /** attachment.recognition_enabled — 是否启用附件解析功能 */
    attachment_recognition_enabled: boolean
    /** attachment.mineru_endpoint — MinerU 文档解析服务自建端点 */
    attachment_mineru_endpoint: string
    /** attachment.mineru_api_key — MinerU API Key（可选） */
    attachment_mineru_api_key?: string
    /** attachment.mineru_backend — pipeline / vlm-* / hybrid-* */
    attachment_mineru_backend: string
    /** attachment.mineru_enable_formula — 公式识别 */
    attachment_mineru_enable_formula: boolean
    /** attachment.mineru_enable_table — 表格识别 */
    attachment_mineru_enable_table: boolean
    /** attachment.mineru_parse_method — MinerU 解析方式：auto / txt / ocr */
    attachment_mineru_parse_method: 'auto' | 'txt' | 'ocr'
    /** attachment.mineru_language — 解析语言 */
    attachment_mineru_language: string
    /** attachment.max_file_size_mb — 最大文件大小（MB） */
    attachment_max_file_size_mb: number
    /** attachment.supported_types — 支持的文件类型（逗号分隔） */
    attachment_supported_types: string
    /** attachment.compat_endpoint — 兼容格式解析服务端点 */
    attachment_compat_endpoint: string
    /** attachment.compat_api_key — 兼容格式解析服务 API Key（可选） */
    attachment_compat_api_key?: string
    /** attachment.legacy_office_enabled — 是否解析 DOC / XLS / PPT */
    attachment_legacy_office_enabled: boolean
    /** attachment.ofd_enabled — 是否解析 OFD */
    attachment_ofd_enabled: boolean
    /** attachment.visual_fallback_enabled — 复杂版式是否回退到 MinerU */
    attachment_visual_fallback_enabled: boolean
}



/**
 * 将后端 ConfigItem[] 转换为 SystemGeneralConfig
 * int/bool 字段统一做类型转换
 */
export function mapConfigItems(items: ConfigItem[]): Partial<SystemGeneralConfig> {
    const kv: Record<string, string> = {}
    items.forEach(item => { kv[item.key] = item.value })

    const int = (k: string) => parseInt(kv[k] ?? '', 10)
    const bool = (k: string) => kv[k] === 'true'
    const str = (k: string) => kv[k] ?? undefined

    return {
        ...(str('system.name') !== undefined && { platform_name: kv['system.name'] }),
        ...(str('system.version') !== undefined && { platform_version: kv['system.version'] }),
        ...(str('system.default_language') !== undefined && { default_language: kv['system.default_language'] }),

        ...(!isNaN(int('auth.login_fail_lock_threshold')) && { login_fail_lock_threshold: int('auth.login_fail_lock_threshold') }),
        ...(!isNaN(int('auth.account_lock_minutes')) && { account_lock_minutes: int('auth.account_lock_minutes') }),
        ...(!isNaN(int('auth.access_token_ttl_hours')) && { access_token_ttl_hours: int('auth.access_token_ttl_hours') }),
        ...(!isNaN(int('auth.refresh_token_ttl_days')) && { refresh_token_ttl_days: int('auth.refresh_token_ttl_days') }),
        ...(str('auth.default_password') !== undefined && { default_password: kv['auth.default_password'] }),

        ...(!isNaN(int('tenant.default_token_quota')) && { tenant_default_token_quota: int('tenant.default_token_quota') }),
        ...(!isNaN(int('tenant.default_max_concurrency')) && { tenant_default_max_concurrency: int('tenant.default_max_concurrency') }),
        ...(!isNaN(int('tenant.default_log_retention_days')) && { tenant_default_log_retention_days: int('tenant.default_log_retention_days') }),
        ...(!isNaN(int('tenant.default_data_retention_days')) && { tenant_default_data_retention_days: int('tenant.default_data_retention_days') }),
        ...(!isNaN(int('system.global_log_retention_days')) && { global_log_retention_days: int('system.global_log_retention_days') }),

        ...(str('system.enable_audit_trail') !== undefined && { enable_audit_trail: bool('system.enable_audit_trail') }),
        ...(str('system.enable_data_encryption') !== undefined && { enable_data_encryption: bool('system.enable_data_encryption') }),
        ...(str('system.backup_enabled') !== undefined && { backup_enabled: bool('system.backup_enabled') }),
        ...(str('system.backup_cron') !== undefined && { backup_cron: kv['system.backup_cron'] }),
        ...(!isNaN(int('system.backup_retention_days')) && { backup_retention_days: int('system.backup_retention_days') }),

        ...(str('system.notification_email') !== undefined && { notification_email: kv['system.notification_email'] }),
        ...(str('system.smtp_host') !== undefined && { smtp_host: kv['system.smtp_host'] }),
        ...(!isNaN(int('system.smtp_port')) && { smtp_port: int('system.smtp_port') }),
        ...(str('system.smtp_username') !== undefined && { smtp_username: kv['system.smtp_username'] }),
        ...(str('system.smtp_ssl') !== undefined && { smtp_ssl: bool('system.smtp_ssl') }),
        ...(str('system.smtp_password') !== undefined && { smtp_password: kv['system.smtp_password'] }),
        ...(str('system.smtp_sender') !== undefined && { smtp_sender: kv['system.smtp_sender'] }),

        ...(str('attachment.recognition_enabled') !== undefined && { attachment_recognition_enabled: bool('attachment.recognition_enabled') }),
        ...(str('attachment.mineru_endpoint') !== undefined && { attachment_mineru_endpoint: kv['attachment.mineru_endpoint'] }),
        ...(str('attachment.mineru_api_key') !== undefined && { attachment_mineru_api_key: kv['attachment.mineru_api_key'] }),
        ...(str('attachment.mineru_backend') !== undefined && { attachment_mineru_backend: kv['attachment.mineru_backend'] }),
        ...(str('attachment.mineru_enable_formula') !== undefined && { attachment_mineru_enable_formula: bool('attachment.mineru_enable_formula') }),
        ...(str('attachment.mineru_enable_table') !== undefined && { attachment_mineru_enable_table: bool('attachment.mineru_enable_table') }),
        ...(str('attachment.mineru_parse_method') !== undefined
            ? { attachment_mineru_parse_method: kv['attachment.mineru_parse_method'] as 'auto' | 'txt' | 'ocr' }
            : str('attachment.mineru_enable_ocr') !== undefined
                ? { attachment_mineru_parse_method: bool('attachment.mineru_enable_ocr') ? 'ocr' : 'txt' }
                : {}),
        ...(str('attachment.mineru_language') !== undefined && { attachment_mineru_language: kv['attachment.mineru_language'] }),
        ...(!isNaN(int('attachment.max_file_size_mb')) && { attachment_max_file_size_mb: int('attachment.max_file_size_mb') }),
        ...(str('attachment.supported_types') !== undefined && { attachment_supported_types: kv['attachment.supported_types'] }),
        ...(str('attachment.compat_endpoint') !== undefined && { attachment_compat_endpoint: kv['attachment.compat_endpoint'] }),
        ...(str('attachment.compat_api_key') !== undefined && { attachment_compat_api_key: kv['attachment.compat_api_key'] }),
        ...(str('attachment.legacy_office_enabled') !== undefined && { attachment_legacy_office_enabled: bool('attachment.legacy_office_enabled') }),
        ...(str('attachment.ofd_enabled') !== undefined && { attachment_ofd_enabled: bool('attachment.ofd_enabled') }),
        ...(str('attachment.visual_fallback_enabled') !== undefined && { attachment_visual_fallback_enabled: bool('attachment.visual_fallback_enabled') }),
    }
}

/**
 * 将 SystemGeneralConfig 转回后端需要的 key-value Record
 */
export function configToUpdateRequest(cfg: SystemGeneralConfig): ConfigUpdateRequest {
    return {
        'system.name': cfg.platform_name,
        'system.version': cfg.platform_version,
        'system.default_language': cfg.default_language,

        'auth.login_fail_lock_threshold': String(cfg.login_fail_lock_threshold),
        'auth.account_lock_minutes': String(cfg.account_lock_minutes),
        'auth.access_token_ttl_hours': String(cfg.access_token_ttl_hours),
        'auth.refresh_token_ttl_days': String(cfg.refresh_token_ttl_days),
        'auth.default_password': cfg.default_password,

        'tenant.default_token_quota': String(cfg.tenant_default_token_quota),
        'tenant.default_max_concurrency': String(cfg.tenant_default_max_concurrency),
        'tenant.default_log_retention_days': String(cfg.tenant_default_log_retention_days),
        'tenant.default_data_retention_days': String(cfg.tenant_default_data_retention_days),
        'system.global_log_retention_days': String(cfg.global_log_retention_days),

        'system.enable_audit_trail': String(cfg.enable_audit_trail),
        'system.enable_data_encryption': String(cfg.enable_data_encryption),
        'system.backup_enabled': String(cfg.backup_enabled),
        'system.backup_cron': cfg.backup_cron,
        'system.backup_retention_days': String(cfg.backup_retention_days),

        'system.notification_email': cfg.notification_email,
        'system.smtp_host': cfg.smtp_host,
        'system.smtp_port': String(cfg.smtp_port),
        'system.smtp_username': cfg.smtp_username,
        'system.smtp_ssl': String(cfg.smtp_ssl),
        'system.smtp_password': cfg.smtp_password || '',
        'system.smtp_sender': cfg.smtp_sender || '',

        'attachment.recognition_enabled': String(cfg.attachment_recognition_enabled ?? false),
        'attachment.mineru_endpoint': cfg.attachment_mineru_endpoint ?? '',
        'attachment.mineru_api_key': cfg.attachment_mineru_api_key ?? '',
        'attachment.mineru_backend': cfg.attachment_mineru_backend ?? 'pipeline',
        'attachment.mineru_enable_formula': String(cfg.attachment_mineru_enable_formula ?? true),
        'attachment.mineru_enable_table': String(cfg.attachment_mineru_enable_table ?? true),
        'attachment.mineru_parse_method': cfg.attachment_mineru_parse_method ?? 'ocr',
        'attachment.mineru_language': cfg.attachment_mineru_language ?? 'ch',
        'attachment.max_file_size_mb': String(cfg.attachment_max_file_size_mb ?? 10),
        'attachment.supported_types': cfg.attachment_supported_types ?? 'pdf,png,jpg,jpeg,bmp,gif,tiff,webp,txt,csv,md,docx,xlsx,pptx,doc,xls,ppt,ofd',
        'attachment.compat_endpoint': cfg.attachment_compat_endpoint ?? 'http://document-parser:8090',
        'attachment.compat_api_key': cfg.attachment_compat_api_key ?? '',
        'attachment.legacy_office_enabled': String(cfg.attachment_legacy_office_enabled ?? false),
        'attachment.ofd_enabled': String(cfg.attachment_ofd_enabled ?? false),
        'attachment.visual_fallback_enabled': String(cfg.attachment_visual_fallback_enabled ?? true),
    }
}
