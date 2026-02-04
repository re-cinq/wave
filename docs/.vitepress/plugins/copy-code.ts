/**
 * Copy Code Plugin for VitePress
 * Injects copy buttons into code blocks
 */

import type { App } from 'vue'

export function setupCopyCode() {
  if (typeof window === 'undefined') return

  // Wait for DOM to be ready
  const initCopyButtons = () => {
    const codeBlocks = document.querySelectorAll('.vp-doc div[class*="language-"]')

    codeBlocks.forEach((block) => {
      // Skip if already has copy button
      if (block.querySelector('.copy-code-button')) return

      const pre = block.querySelector('pre')
      const code = block.querySelector('code')
      if (!code) return

      const button = document.createElement('button')
      button.className = 'copy-code-button'
      button.setAttribute('aria-label', 'Copy code')
      button.setAttribute('title', 'Copy to clipboard')
      button.innerHTML = `
        <svg class="copy-icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
        </svg>
        <svg class="check-icon" style="display:none" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
        <span class="copy-text">Copy</span>
      `

      button.addEventListener('click', async () => {
        const text = code.textContent || ''

        try {
          await navigator.clipboard.writeText(text)
          showCopied(button)
        } catch (err) {
          // Fallback
          const textarea = document.createElement('textarea')
          textarea.value = text
          textarea.style.position = 'fixed'
          textarea.style.opacity = '0'
          document.body.appendChild(textarea)
          textarea.select()

          try {
            document.execCommand('copy')
            showCopied(button)
          } catch (fallbackErr) {
            console.error('Copy failed:', fallbackErr)
          } finally {
            document.body.removeChild(textarea)
          }
        }
      })

      block.style.position = 'relative'
      block.appendChild(button)
    })
  }

  function showCopied(button: HTMLButtonElement) {
    const copyIcon = button.querySelector('.copy-icon') as HTMLElement
    const checkIcon = button.querySelector('.check-icon') as HTMLElement
    const copyText = button.querySelector('.copy-text') as HTMLElement

    button.classList.add('copied')
    if (copyIcon) copyIcon.style.display = 'none'
    if (checkIcon) checkIcon.style.display = 'block'
    if (copyText) copyText.textContent = 'Copied!'

    setTimeout(() => {
      button.classList.remove('copied')
      if (copyIcon) copyIcon.style.display = 'block'
      if (checkIcon) checkIcon.style.display = 'none'
      if (copyText) copyText.textContent = 'Copy'
    }, 2000)
  }

  // Initialize on page load and navigation
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initCopyButtons)
  } else {
    initCopyButtons()
  }

  // Re-initialize on route changes (for SPA navigation)
  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
        // Debounce to avoid multiple calls
        setTimeout(initCopyButtons, 100)
        break
      }
    }
  })

  observer.observe(document.body, {
    childList: true,
    subtree: true
  })
}

// Styles for the copy button (injected via JS)
export function injectCopyCodeStyles() {
  if (typeof window === 'undefined') return
  if (document.getElementById('copy-code-styles')) return

  const style = document.createElement('style')
  style.id = 'copy-code-styles'
  style.textContent = `
    .copy-code-button {
      position: absolute;
      top: 8px;
      right: 8px;
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 6px 10px;
      font-size: 12px;
      font-weight: 500;
      color: var(--vp-c-text-2);
      background: var(--vp-c-bg-soft);
      border: 1px solid var(--vp-c-divider);
      border-radius: 6px;
      cursor: pointer;
      opacity: 0;
      transition: all 0.15s ease;
      z-index: 10;
    }

    div[class*="language-"]:hover .copy-code-button {
      opacity: 1;
    }

    .copy-code-button:hover {
      color: var(--vp-c-text-1);
      background: var(--vp-c-bg-mute);
      border-color: var(--vp-c-brand-1);
    }

    .copy-code-button.copied {
      color: #10b981;
      border-color: #10b981;
    }

    .copy-code-button svg {
      flex-shrink: 0;
    }
  `
  document.head.appendChild(style)
}
