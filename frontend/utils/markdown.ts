import { marked } from 'marked'
import sanitizeHtml from 'sanitize-html'

/** 模型回复及外部工具内容只渲染安全的阅读型 Markdown。 */
export function renderSafeMarkdown(content: string): string {
  return sanitizeHtml(marked.parse(content || '', { async: false, gfm: true }) as string, {
    allowedTags: sanitizeHtml.defaults.allowedTags.concat(['del', 'input']),
    allowedAttributes: { ...sanitizeHtml.defaults.allowedAttributes, a: ['href', 'name', 'target', 'rel'], code: ['class'], input: ['type', 'checked', 'disabled'] },
    allowedSchemes: ['http', 'https', 'mailto'], allowProtocolRelative: false,
    transformTags: {
      a: sanitizeHtml.simpleTransform('a', { target: '_blank', rel: 'noopener noreferrer' }),
      input: (_tag, attrs) => ({ tagName: 'input', attribs: { type: 'checkbox', disabled: '', ...(attrs.checked !== undefined ? { checked: '' } : {}) } }),
    },
  })
}
