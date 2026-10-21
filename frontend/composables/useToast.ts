export type AlertType = 'success' | 'error' | 'warning' | 'info'

export interface ToastItem {
  id: number
  type: AlertType
  title: string
  message: string
  /** Milliseconds before it closes on its own (paused while hovered). */
  duration: number
}

interface ToastOptions {
  title?: string
  duration?: number
}

const DEFAULT_TITLES: Record<AlertType, string> = {
  success: 'Done',
  error: 'Something went wrong',
  warning: 'Heads up',
  info: 'Good to know',
}
const DEFAULT_DURATIONS: Record<AlertType, number> = {
  success: 4000,
  error: 7000,
  warning: 6000,
  info: 5000,
}
const MAX_TOASTS = 4

let nextId = 1

/** API errors arrive lower-case ("branch code already exists"); show them as sentences. */
export function sentence(text: string) {
  return text ? text.charAt(0).toUpperCase() + text.slice(1) : text
}

/**
 * Pop-up messages shown by <AppToaster /> (mounted once in app.vue).
 * Use it for the result of an action (saved, deleted, approved…); form errors belong inline in <AppAlert />.
 */
export function useToast() {
  const toasts = useState<ToastItem[]>('app-toasts', () => [])

  function show(type: AlertType, message: string, options: ToastOptions = {}) {
    const id = nextId++
    const item: ToastItem = {
      id,
      type,
      title: options.title || DEFAULT_TITLES[type],
      message: sentence(message),
      duration: options.duration ?? DEFAULT_DURATIONS[type],
    }
    toasts.value = [...toasts.value, item].slice(-MAX_TOASTS)
    return id
  }

  function dismiss(id: number) {
    toasts.value = toasts.value.filter(toast => toast.id !== id)
  }

  return {
    toasts,
    show,
    dismiss,
    success: (message: string, options?: ToastOptions) => show('success', message, options),
    error: (message: string, options?: ToastOptions) => show('error', message, options),
    warning: (message: string, options?: ToastOptions) => show('warning', message, options),
    info: (message: string, options?: ToastOptions) => show('info', message, options),
  }
}
