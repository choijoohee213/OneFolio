export const THEMES = ['system', 'light', 'dark'] as const
export type Theme = (typeof THEMES)[number]

export const THEME_LABELS: Record<Theme, string> = {
  system: '시스템',
  light: '라이트',
  dark: '다크',
}

const KEY = 'onefolio-theme'

export function loadTheme(): Theme {
  const saved = localStorage.getItem(KEY)
  return THEMES.includes(saved as Theme) ? (saved as Theme) : 'system'
}

// system 은 속성을 지워서 prefers-color-scheme 에 맡긴다. 값을 박아두면
// 기기 설정을 바꿔도 따라가지 않는다.
export function applyTheme(theme: Theme): void {
  if (theme === 'system') {
    delete document.documentElement.dataset.theme
  } else {
    document.documentElement.dataset.theme = theme
  }
  localStorage.setItem(KEY, theme)
}
