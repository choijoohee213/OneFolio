import { useEffect, useState } from 'react'
import { toUploadedFiles } from './api'
import { recompute, withoutAccount } from './collection'
import { AccountsPanel } from './components/AccountsPanel'
import { AllocationPie } from './components/AllocationPie'
import { FileDrop } from './components/FileDrop'
import { HoldingsTable, type GroupMode } from './components/HoldingsTable'
import { ManualAssets } from './components/ManualAssets'
import { ThemeToggle } from './components/ThemeToggle'
import { won } from './format'
import { clearState, loadState, saveState, supersededManualAccounts } from './storage'
import { applyTheme, loadTheme, type Theme } from './theme'
import type { Category, ManualAccount, ManualHolding, Overrides, Summary, UploadedFile } from './types'

export default function App() {
  const [files, setFiles] = useState<UploadedFile[]>([])
  const [manualAccounts, setManualAccounts] = useState<ManualAccount[]>([])
  const [manualHoldings, setManualHoldings] = useState<ManualHolding[]>([])
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
        setManualAccounts(state.manualAccounts)
        setManualHoldings(state.manualHoldings)
        setSummary(state.summary)
        setOverrides(state.overrides)
      }
      setRestored(true)
    })
  }, [])

  async function apply(
    nextFiles: UploadedFile[] = files,
    nextOverrides: Overrides = overrides,
    nextManualAccounts: ManualAccount[] = manualAccounts,
    nextManualHoldings: ManualHolding[] = manualHoldings,
  ) {
    setBusy(true)
    setError(null)
    try {
      const collection = await recompute(nextFiles, nextOverrides, nextManualAccounts, nextManualHoldings)
      setFiles(collection.files)
      setManualAccounts(nextManualAccounts)
      setManualHoldings(nextManualHoldings)
      setSummary(collection.summary)
      setOverrides(nextOverrides)
      await saveState(collection.files, nextManualAccounts, nextManualHoldings, collection.summary, nextOverrides)
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

  function addManualAccount(input: { name: string; totalAsset: number; accountNumber?: string }) {
    const id = crypto.randomUUID()
    return apply(files, overrides, [...manualAccounts, { id, ...input }])
  }

  function updateManualAccount(
    id: string,
    input: { name: string; totalAsset: number; accountNumber?: string },
  ) {
    const next = manualAccounts.map((a) => (a.id === id ? { ...a, ...input } : a))
    return apply(files, overrides, next)
  }

  // 계좌를 지우면 거기 붙어 있던 종목은 갈 곳이 없어진다. 같이 지우지 않으면
  // 사라진 계좌를 가리킨 채로 남아 집계에서 조용히 빠진다.
  function removeManualAccount(id: string) {
    const orphaned = manualHoldings.filter((h) => h.accountId === id)
    const nextOverrides = { ...overrides }
    orphaned.forEach((h) => delete nextOverrides[h.name])

    return apply(
      files,
      nextOverrides,
      manualAccounts.filter((a) => a.id !== id),
      manualHoldings.filter((h) => h.accountId !== id),
    )
  }

  function addManualHolding(input: { name: string; category: Category; evalAmount: number; accountId?: string }) {
    const { category, ...rest } = input
    const id = crypto.randomUUID()
    const nextManual = [...manualHoldings, { id, ...rest }]
    const nextOverrides = { ...overrides, [input.name]: category }
    return apply(files, nextOverrides, manualAccounts, nextManual)
  }

  // 이름을 바꾸면 예전 이름에 걸려 있던 분류 매핑은 더 이상 이 종목을 못 찾는다.
  // 그대로 두면 새 이름은 분류를 잃고 자동 추정으로 떨어지므로, 매핑을 새 이름으로 옮긴다.
  function updateManualHolding(
    id: string,
    input: { name: string; category: Category; evalAmount: number; accountId?: string },
  ) {
    const current = manualHoldings.find((item) => item.id === id)
    if (!current) return

    const { category, ...rest } = input
    const nextManual = manualHoldings.map((item) => (item.id === id ? { ...item, ...rest } : item))
    const nextOverrides = { ...overrides }
    if (current.name !== input.name) delete nextOverrides[current.name]
    nextOverrides[input.name] = category

    return apply(files, nextOverrides, manualAccounts, nextManual)
  }

  function removeManualHolding(id: string) {
    return apply(
      files,
      overrides,
      manualAccounts,
      manualHoldings.filter((item) => item.id !== id),
    )
  }

  async function reset() {
    setFiles([])
    setManualAccounts([])
    setManualHoldings([])
    setSummary(null)
    setError(null)
    await clearState()
  }

  const superseded = supersededManualAccounts(manualAccounts, summary)

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
            {/* 계좌가 하나도 없으면(계좌 없이 종목만 있을 때) 계좌 총합은 0원이라
                아무것도 없는 것처럼 보인다. 그럴 땐 실제로 집계된 금액을 보여준다. */}
            <span className="total-label">
              {summary.accounts.length > 0 ? '계좌 총합' : '직접 추가한 자산'}
            </span>
            <strong>{won(summary.accounts.length > 0 ? summary.totalAsset : summary.coveredAsset)}</strong>
          </div>
        )}
      </header>

      <FileDrop
        onFiles={async (picked) => apply([...files, ...(await toUploadedFiles(picked))])}
        busy={busy}
        compact={summary !== null}
      />

      {error && <p className="error">{error}</p>}

      <AccountsPanel
        accounts={summary?.accounts ?? []}
        manualAccounts={manualAccounts}
        superseded={superseded}
        coveredAsset={summary?.coveredAsset ?? 0}
        busy={busy}
        onRemove={(number) => apply(withoutAccount(files, number))}
        onAddAccount={addManualAccount}
        onUpdateAccount={updateManualAccount}
        onRemoveAccount={removeManualAccount}
      />

      <ManualAssets
        manualHoldings={manualHoldings}
        // 파일로 갈음된 계좌는 고를 수 없어야 한다. 붙여 봐야 서버가 조용히 버린다.
        manualAccounts={manualAccounts.filter((account) => !superseded.includes(account))}
        holdings={summary?.holdings ?? []}
        busy={busy}
        onAdd={addManualHolding}
        onUpdate={updateManualHolding}
        onRemove={removeManualHolding}
      />

      {summary && (
        <>
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
        <p className="empty">잔고파일을 올리거나 계좌·자산을 직접 추가하면 자산 배분과 손익을 보여줍니다.</p>
      )}
    </main>
  )
}
