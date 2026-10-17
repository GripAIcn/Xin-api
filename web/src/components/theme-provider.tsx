import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

type ResolvedTheme = 'light' | 'dark'
export type Theme = ResolvedTheme | 'system'

/** 与 index.html / app.html 中首帧内联脚本、以及 next-themes 旧存储键保持一致 */
const STORAGE_KEY = 'xin-api-theme'
const DEFAULT_THEME: Theme = 'light'
const MEDIA_QUERY = '(prefers-color-scheme: dark)'
const THEME_NAMES = ['light', 'dark', 'system'] as const

interface ThemeContextValue {
  /** 用户选择的主题（可能是 system） */
  theme: Theme
  /** 实际生效的主题（system 已解析为 light/dark） */
  resolvedTheme: ResolvedTheme
  /** 系统偏好主题 */
  systemTheme: ResolvedTheme
  themes: string[]
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

/** 读取已持久化主题；SSR 阶段（无 localStorage）返回默认值 */
function readStoredTheme(): Theme {
  if (typeof window === 'undefined') return DEFAULT_THEME
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored && (THEME_NAMES as readonly string[]).includes(stored)) {
      return stored as Theme
    }
  } catch {
    // localStorage 不可用时回退默认主题
  }
  return DEFAULT_THEME
}

function readSystemTheme(): ResolvedTheme {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia(MEDIA_QUERY).matches ? 'dark' : 'light'
}

/**
 * 主题提供者：只维护主题状态并应用到 <html>，**不在 React 树内注入 <script>**。
 *
 * 首帧防闪烁由 index.html / app.html 中的内联脚本完成，因此无需像 next-themes
 * 那样在组件树里渲染引导脚本——该写法在 React 19 会触发
 * "Encountered a script tag while rendering React component" 警告，
 * 且客户端创建的 script 节点并不会执行。
 */
export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(readStoredTheme)
  const [systemTheme, setSystemTheme] = useState<ResolvedTheme>(readSystemTheme)

  const resolvedTheme: ResolvedTheme = theme === 'system' ? systemTheme : theme

  // 应用主题到 <html>；切换瞬间禁用过渡，避免颜色过渡导致的闪烁
  useEffect(() => {
    const root = document.documentElement
    const style = document.createElement('style')
    style.appendChild(document.createTextNode('*{transition:none!important}'))
    document.head.appendChild(style)

    root.classList.toggle('dark', resolvedTheme === 'dark')
    root.style.colorScheme = resolvedTheme

    const timer = window.setTimeout(() => style.remove(), 1)
    return () => {
      window.clearTimeout(timer)
      style.remove()
    }
  }, [resolvedTheme])

  // 跟随系统：监听系统明暗偏好变化
  useEffect(() => {
    const media = window.matchMedia(MEDIA_QUERY)
    const listener = (event: MediaQueryListEvent) => {
      setSystemTheme(event.matches ? 'dark' : 'light')
    }
    media.addEventListener('change', listener)
    return () => media.removeEventListener('change', listener)
  }, [])

  // 跨标签页同步：其他标签页修改主题后同步过来
  useEffect(() => {
    const listener = (event: StorageEvent) => {
      if (event.key === STORAGE_KEY) setThemeState(readStoredTheme())
    }
    window.addEventListener('storage', listener)
    return () => window.removeEventListener('storage', listener)
  }, [])

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next)
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // 忽略不可写场景
    }
  }, [])

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme,
      resolvedTheme,
      systemTheme,
      themes: [...THEME_NAMES],
      setTheme,
    }),
    [theme, resolvedTheme, systemTheme, setTheme],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

/** 未包裹 ThemeProvider 时的安全兜底，避免调用方取值为 undefined */
const FALLBACK: ThemeContextValue = {
  theme: DEFAULT_THEME,
  resolvedTheme: DEFAULT_THEME,
  systemTheme: DEFAULT_THEME,
  themes: [...THEME_NAMES],
  setTheme: () => {},
}

export function useTheme(): ThemeContextValue {
  return useContext(ThemeContext) ?? FALLBACK
}
