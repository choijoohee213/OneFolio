import { useEffect, useState } from 'react'
import { toUploadedFiles } from './api'
import { recompute, withoutAccount } from './collection'
import { AccountsPanel } from './components/AccountsPanel'
import { AllocationPie } from './components/AllocationPie'
import { FileDrop } from './components/FileDrop'
import { HoldingsTable, type GroupMode } from './components/HoldingsTable'
import { won } from './format'
import { clearState, loadState, saveState } from './storage'
import type { Overrides, Summary, UploadedFile } from './types'

export default function App() {
  const [files, setFiles] = useState<UploadedFile[]>([])
  const [summary, setSummary] = useState<Summary | null>(null)
  const [overrides, setOverrides] = useState<Overrides>({})
  const [mode, setMode] = useState<GroupMode>('all')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [restored, setRestored] = useState(false)

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

  async function apply(next: UploadedFile[]) {
    setBusy(true)
    setError(null)
    try {
      const collection = await recompute(next, overrides)
      setFiles(collection.files)
      setSummary(collection.summary)
      await saveState(collection.files, collection.summary, overrides)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
    } finally {
      setBusy(false)
    }
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
