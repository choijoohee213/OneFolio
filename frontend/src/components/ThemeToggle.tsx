import { THEMES, THEME_LABELS, type Theme } from '../theme'

interface Props {
  theme: Theme
  onChange: (theme: Theme) => void
}

export function ThemeToggle({ theme, onChange }: Props) {
  return (
    <div className="toggle theme-toggle" role="group" aria-label="화면 테마">
      {THEMES.map((option) => (
        <button
          key={option}
          type="button"
          aria-pressed={theme === option}
          onClick={() => onChange(option)}
        >
          {THEME_LABELS[option]}
        </button>
      ))}
    </div>
  )
}
