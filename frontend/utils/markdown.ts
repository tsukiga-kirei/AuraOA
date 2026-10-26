import { marked } from 'marked'
import sanitizeHtml from 'sanitize-html'

/** 模型回复及外部工具内容只渲染安全的阅读型 Markdown。 */
export function renderSafeMarkdown(content: string): string {
  try {
    return sanitizeHtml(marked.parse(normalizeModelMarkdown(content || ''), { async: false, gfm: true, breaks: true }) as string, {
      allowedTags: sanitizeHtml.defaults.allowedTags.concat(['del', 'input']),
      allowedAttributes: { ...sanitizeHtml.defaults.allowedAttributes, a: ['href', 'name', 'target', 'rel'], code: ['class'], input: ['type', 'checked', 'disabled'] },
      allowedSchemes: ['http', 'https', 'mailto'], allowProtocolRelative: false,
      transformTags: {
        a: sanitizeHtml.simpleTransform('a', { target: '_blank', rel: 'noopener noreferrer' }),
        input: (_tag, attrs) => ({ tagName: 'input', attribs: { type: 'checkbox', disabled: '', ...(attrs.checked !== undefined ? { checked: '' } : {}) } }),
      },
    })
  } catch {
    return sanitizeHtml(content || '', { allowedTags: [], allowedAttributes: {} })
  }
}

/** 兼容模型流式输出时省略标题空格的写法，不改动代码围栏中的内容。 */
export function normalizeModelMarkdown(content: string): string {
  let fence = ''
  return content.split('\n').map(line => {
    const marker = line.match(/^\s*(`{3,}|~{3,})/)
    if (marker) { if (!fence) fence = marker[1][0]; else if (fence === marker[1][0]) fence = ''; return line }
    if (fence) return line
    return line.replace(/^( {0,3}#{1,6})(?=[^#\s])/, '$1 ')
  }).join('\n')
}
