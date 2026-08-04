import { useEffect, useState } from 'react'
import { fetchPortfolio } from './api'
import { AllocationBar } from './components/AllocationBar'
import { FileDrop } from './components/FileDrop'
import { HoldingsTable, type GroupMode } from './components/HoldingsTable'
import { won } from './format'
import { clearState, loadState, saveState } from './storage'
import type { Overrides, Summary } from './types'

export default function App() {
  const [summary, setSummary] = useState<Summary | null>(null)
  const [overrides, setOverrides] = useState<Overrides>({})
  const [mode, setMode] = useState<GroupMode>('name')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [restored, setRestored] = useState(false)

  useEffect(() => {
    loadState().then((state) => {
      if (state?.summary) setSummary(state.summary)
      if (state?.overrides) setOverrides(state.overrides)
      setRestored(true)
    })
  }, [])

  async function upload(files: File[]) {
    setBusy(true)
    setError(null)
    try {
      const result = await fetchPortfolio(files, overrides)
      setSummary(result)
      await saveState(result, overrides)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
    } finally {
      setBusy(false)
    }
  }

  async function reset() {
    setSummary(null)
    setError(null)
    await clearState()
  }

  return (
    <main>
      <header className="page-head">
        <h1>OneFolio</h1>
        {summary && (
          <div className="total">
            <span className="total-label">총자산</span>
            <strong>{won(summary.totalAsset)}</strong>
          </div>
        )}
      </header>

      <FileDrop onFiles={upload} busy={busy} compact={summary !== null} />

      {error && <p className="error">{error}</p>}

      {summary && (
        <>
          <AllocationBar categories={summary.categories} />
          <HoldingsTable holdings={summary.holdings} mode={mode} onModeChange={setMode} />
          <footer className="page-foot">
            <p>자산 데이터는 이 브라우저에만 저장됩니다. 서버는 계산 후 아무것도 남기지 않습니다.</p>
            <button type="button" className="link" onClick={reset}>
              저장된 데이터 지우기
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
