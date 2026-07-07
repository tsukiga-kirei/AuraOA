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
  return value == null ? dayjs().tz(timeZone) : dayjs(value).tz(timeZone)
}

export function formatDateTimeInAppZone(
  value: string | number | Date | null | undefined,
  locale = 'zh-CN',
  options: Intl.DateTimeFormatOptions = {},
) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
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
