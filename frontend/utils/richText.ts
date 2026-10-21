import DOMPurify from 'dompurify'

/**
 * Rich-text descriptions are a small HTML subset. The API sanitizes on save; we sanitize again
 * before rendering (v-html) so nothing unexpected can ever run in the page.
 */
const ALLOWED_TAGS = ['b', 'strong', 'i', 'em', 'u', 's', 'p', 'div', 'br', 'ul', 'ol', 'li', 'a']

let hookInstalled = false
function installLinkHook() {
  if (hookInstalled) return
  // Every link opens in a new tab without access to this page.
  DOMPurify.addHook('afterSanitizeAttributes', (node) => {
    if (node.tagName === 'A') {
      node.setAttribute('target', '_blank')
      node.setAttribute('rel', 'noopener noreferrer nofollow')
    }
  })
  hookInstalled = true
}

export function sanitizeRichText(html: string | null | undefined): string {
  if (!html) return ''
  installLinkHook()
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS,
    ALLOWED_ATTR: ['href'],
    ALLOWED_URI_REGEXP: /^(?:https?:|mailto:)/i,
  })
}

/** Plain text for one-line labels (list titles, search). Old plain descriptions pass through. */
export function richTextToPlain(html: string | null | undefined): string {
  if (!html) return ''
  if (!/[<&]/.test(html)) return html.trim()
  const doc = new DOMParser().parseFromString(sanitizeRichText(html), 'text/html')
  doc.querySelectorAll('br, p, div, li').forEach(el => el.append(' '))
  return (doc.body.textContent || '').replace(/\s+/g, ' ').trim()
}

/** True when the editor holds no visible text (e.g. only <br> or empty list items). */
export function isRichTextEmpty(html: string | null | undefined): boolean {
  return richTextToPlain(html) === ''
}
