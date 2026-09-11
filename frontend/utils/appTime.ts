import dayjs, { type Dayjs } from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'

dayjs.extend(utc)
dayjs.extend(timezone)

const DEFAULT_TIME_ZONE = 'Asia/Shanghai'

export function useAppTimeZone(): string {
  const config = useRuntimeConfig()
  const timeZone = String(config.public.timeZone || '').trim()
  return timeZone || DEFAULT_TIME_ZONE
}

export function appDayjs(value?: string | number | Date | Dayjs | null): Dayjs {
  const timeZone = useAppTimeZone()
  if (value == null) return dayjs().tz(timeZone)
  // OA 无时区日期按应用时区解释；带偏移的时间戳保留其代表的实际时刻。
  if (typeof value === 'string' && /^\d{4}[-/]\d{1,2}[-/]\d{1,2}(?:[ T]\d{1,2}:\d{2}(?::\d{2}(?:\.\d+)?)?)?$/.test(value.trim())) {
    try {
      return dayjs.tz(value.trim().replace(/\//g, '-'), timeZone)
    } catch {
      return dayjs(Number.NaN)
    }
  }
  return dayjs(value).tz(timeZone)
}

export function formatDateTimeInAppZone(
  value: string | number | Date | null | undefined,
  locale = 'zh-CN',
  options: Intl.DateTimeFormatOptions = {},
) {
  if (value == null || value === '') return '-'
  const parsed = appDayjs(value)
  if (!parsed.isValid()) return String(value)
  const d = parsed.toDate()
  const defaultOptions: Intl.DateTimeFormatOptions = options.dateStyle || options.timeStyle
    ? {}
    : {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }
  return new Intl.DateTimeFormat(locale, {
    timeZone: useAppTimeZone(),
    ...defaultOptions,
    ...options,
  }).format(d)
}

export function formatDateTimeTextInAppZone(value: string | number | Date | null | undefined, locale = 'zh-CN') {
  return formatDateTimeInAppZone(value, locale).replace(/\//g, '-')
}
