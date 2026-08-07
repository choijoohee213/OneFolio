import { useEffect, useState } from 'react'
import { toUploadedFiles } from './api'
import { recompute, withoutAccount } from './collection'
import { AccountsPanel } from './components/AccountsPanel'
import { AllocationPie } from './components/AllocationPie'
import { FileDrop } from './components/FileDrop'
import { HoldingsTable, type GroupMode } from './components/HoldingsTable'
import { ThemeToggle } from './components/ThemeToggle'
import { won } from './format'
import { clearState, loadState, saveState } from './storage'
import { applyTheme, loadTheme, type Theme } from './theme'
import type { Category, Overrides, Summary, UploadedFile } from './types'

export default function App() {
  const [files, setFiles] = useState<UploadedFile[]>([])
  const [summary, setSummary] = useState<Summary | null>(null)
  const [overrides, setOverrides] = useState<Overrides>({})
  const [mode, setMode] = useState<GroupMode>('all')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [restored, setRestored] = useState(false)
  const [theme, setTheme] = useState<Theme>(loadTheme)

  useEffect(() => {
    loadState().then((state) => {
      if (state) {
        setFiles(state.files)
        setSummary(state.summary)
        setOverrides(state.overrides)
      }
      setRestored(true)
    })
  }, [])

  async function apply(nextFiles: UploadedFile[], nextOverrides: Overrides = overrides) {
    setBusy(true)
    setError(null)
    try {
      const collection = await recompute(nextFiles, nextOverrides)
      setFiles(collection.files)
      setSummary(collection.summary)
      setOverrides(nextOverrides)
      await saveState(collection.files, collection.summary, nextOverrides)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
    } finally {
      setBusy(false)
    }
  }

  // 분류를 바꾸면 서버가 그 매핑으로 다시 집계한다. 비중과 카테고리 합계가
  // 함께 움직여야 해서 화면에서 값만 갈아끼울 수는 없다.
  function changeCategory(name: string, category: Category | null) {
    const next = { ...overrides }
    if (category === null) {
      delete next[name]
    } else {
      next[name] = category
    }
    return apply(files, next)
  }

  async function reset() {
    setFiles([])
    setSummary(null)
    setError(null)
    await clearState()
  }

  return (
    <main>
      <header className="page-head">
        <h1>OneFolio</h1>
        <ThemeToggle
          theme={theme}
          onChange={(next) => {
            setTheme(next)
            applyTheme(next)
          }}
        />
        {summary && (
          <div className="total">
            <span className="total-label">계좌 총합</span>
            <strong>{won(summary.totalAsset)}</strong>
          </div>
        )}
      </header>

      <FileDrop
        onFiles={async (picked) => apply([...files, ...(await toUploadedFiles(picked))])}
        busy={busy}
        compact={summary !== null}
      />

      {error && <p className="error">{error}</p>}

      {summary && (
        <>
          <AccountsPanel
            accounts={summary.accounts}
            coveredAsset={summary.coveredAsset}
            busy={busy}
            onRemove={(number) => apply(withoutAccount(files, number))}
          />
          <AllocationPie categories={summary.categories} coveredAsset={summary.coveredAsset} />
          <HoldingsTable
            holdings={summary.holdings}
            accounts={summary.accounts}
            mode={mode}
            onModeChange={setMode}
            overrides={overrides}
            busy={busy}
            onCategoryChange={changeCategory}
          />
          <footer className="page-foot">
            <p>
              올린 잔고파일 {files.length}개와 집계 결과는 이 브라우저에만 저장됩니다. 서버는 계산
              후 아무것도 남기지 않습니다.
            </p>
            <button type="button" className="link" onClick={reset}>
              전부 지우기
            </button>
          </footer>
        </>
      )}

      {restored && !summary && !busy && !error && (
        <p className="empty">잔고파일을 올리면 자산 배분과 종목별 손익을 보여줍니다.</p>
      )}
    </main>
  )
}
