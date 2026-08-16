import { useEffect, useState } from 'react'
import { toUploadedFiles } from './api'
import { createSampleFiles } from './sampleData'
import { recompute, withoutAccount } from './collection'
import { AccountForm, type AccountInput } from './components/AccountForm'
import { AccountsPanel } from './components/AccountsPanel'
import { AllocationPie } from './components/AllocationPie'
import { EditConflicts } from './components/EditConflicts'
import { FileDrop } from './components/FileDrop'
import { HoldingForm, type FileInput, type HoldingTarget, type ManualInput } from './components/HoldingForm'
import { ScreenshotImport } from './components/ScreenshotImport'
import { HoldingsTable, type GroupMode } from './components/HoldingsTable'
import { ThemeToggle } from './components/ThemeToggle'
import { UnmatchedResolver } from './components/UnmatchedResolver'
import { won } from './format'
import { clearState, conflictingEdits, loadState, saveState, supersededManualAccounts } from './storage'
import { applyTheme, loadTheme, type Theme } from './theme'
import type {
  ExtractedHolding,
  FileValues,
  Holding,
  HoldingEdit,
  ManualAccount,
  ManualHolding,
  Overrides,
  StockMappings,
  Summary,
  UploadedFile,
} from './types'
import { holdingKey, isManualHolding, MANUAL_ACCOUNT_PREFIX, MANUAL_HOLDING_PREFIX } from './types'

export default function App() {
  const [files, setFiles] = useState<UploadedFile[]>([])
  const [manualAccounts, setManualAccounts] = useState<ManualAccount[]>([])
  const [manualHoldings, setManualHoldings] = useState<ManualHolding[]>([])
  const [holdingEdits, setHoldingEdits] = useState<HoldingEdit[]>([])
  const [summary, setSummary] = useState<Summary | null>(null)
  const [overrides, setOverrides] = useState<Overrides>({})
  const [mode, setMode] = useState<GroupMode>('all')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [restored, setRestored] = useState(false)
  const [theme, setTheme] = useState<Theme>(loadTheme)
  const [stockMappings, setStockMappings] = useState<StockMappings>({})
  const [holdingTarget, setHoldingTarget] = useState<HoldingTarget | null>(null)
  const [accountTarget, setAccountTarget] = useState<ManualAccount | null | 'new'>(null)
  const [showUnmatched, setShowUnmatched] = useState(false)
  const [showScreenshot, setShowScreenshot] = useState(false)

  useEffect(() => {
    loadState().then((state) => {
      if (state) {
        setFiles(state.files)
        setManualAccounts(state.manualAccounts)
        setManualHoldings(state.manualHoldings)
        setHoldingEdits(state.holdingEdits)
        setSummary(state.summary)
        setOverrides(state.overrides)
        setStockMappings(state.stockMappings)
      }
      setRestored(true)
    })
  }, [])

  async function apply(next: {
    files?: UploadedFile[]
    overrides?: Overrides
    manualAccounts?: ManualAccount[]
    manualHoldings?: ManualHolding[]
    holdingEdits?: HoldingEdit[]
    stockMappings?: StockMappings
  }) {
    const nextFiles = next.files ?? files
    const nextOverrides = next.overrides ?? overrides
    const nextAccounts = next.manualAccounts ?? manualAccounts
    const nextHoldings = next.manualHoldings ?? manualHoldings
    const nextEdits = next.holdingEdits ?? holdingEdits
    const nextMappings = next.stockMappings ?? stockMappings

    setBusy(true)
    setError(null)
    try {
      const collection = await recompute(nextFiles, nextOverrides, nextAccounts, nextHoldings, nextEdits, nextMappings)
      setFiles(collection.files)
      setManualAccounts(nextAccounts)
      setManualHoldings(nextHoldings)
      setHoldingEdits(nextEdits)
      setSummary(collection.summary)
      setOverrides(nextOverrides)
      setStockMappings(nextMappings)

      const unmatched = collection.summary?.unmatched ?? []
      const hasNew = unmatched.some((n) => !(n in nextMappings))
      if (hasNew) setShowUnmatched(true)

      await saveState(collection.files, nextAccounts, nextHoldings, nextEdits, collection.summary, nextOverrides, nextMappings)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
    } finally {
      setBusy(false)
    }
  }

  // ---------- 계좌 ----------

  function submitAccount(input: AccountInput) {
    const editing = accountTarget !== 'new' ? accountTarget : null
    const next = editing
      ? manualAccounts.map((a) => (a.id === editing.id ? { ...a, ...input } : a))
      : [...manualAccounts, { id: crypto.randomUUID(), ...input }]
    setAccountTarget(null)
    return apply({ manualAccounts: next })
  }

  // 계좌를 지우면 거기 붙어 있던 종목은 갈 곳이 없어진다. 같이 지우지 않으면
  // 사라진 계좌를 가리킨 채로 남아 집계에서 조용히 빠진다.
  function removeAccount(id: string) {
    const orphaned = manualHoldings.filter((h) => h.accountId === id)
    const nextOverrides = { ...overrides }
    orphaned.forEach((h) => delete nextOverrides[h.name])

    return apply({
      overrides: nextOverrides,
      manualAccounts: manualAccounts.filter((a) => a.id !== id),
      manualHoldings: manualHoldings.filter((h) => h.accountId !== id),
    })
  }

  // ---------- 종목 ----------

  // 표의 행에서 무엇을 편집하는지 가린다. 직접 추가한 종목은 원본 항목까지 찾아야
  // 이름·소속 계좌를 고칠 수 있다.
  function openHoldingEditor(holding: Holding) {
    if (!isManualHolding(holding)) {
      setHoldingTarget({ kind: 'file', holding })
      return
    }
    const item = manualHoldings.find(
      (m) =>
        holding.accountNumber ===
          (m.accountId ? MANUAL_ACCOUNT_PREFIX + m.accountId : MANUAL_HOLDING_PREFIX + m.id) &&
        m.name === holding.name,
    )
    if (item) setHoldingTarget({ kind: 'manual', item, holding })
  }

  function submitManualHolding(input: ManualInput) {
    const target = holdingTarget
    setHoldingTarget(null)
    if (!target || target.kind === 'file') return

    const { category, ...rest } = input
    const nextOverrides = { ...overrides }

    if (target.kind === 'new') {
      nextOverrides[input.name] = category
      return apply({
        overrides: nextOverrides,
        manualHoldings: [...manualHoldings, { id: crypto.randomUUID(), ...rest }],
      })
    }

    // 이름을 바꾸면 예전 이름에 걸려 있던 분류 매핑은 이 종목을 못 찾는다.
    if (target.item.name !== input.name) delete nextOverrides[target.item.name]
    nextOverrides[input.name] = category
    return apply({
      overrides: nextOverrides,
      manualHoldings: manualHoldings.map((m) => (m.id === target.item.id ? { ...m, ...rest } : m)),
    })
  }

  function submitFileHolding(holding: Holding, input: FileInput) {
    setHoldingTarget(null)

    // 이미 고친 종목을 또 고칠 때 basedOn 은 최초 파일 값을 유지해야 한다.
    // 내가 고친 값을 기준으로 삼으면 파일이 바뀌어도 충돌을 못 잡는다.
    const basedOn: FileValues = holding.original ?? {
      quantity: holding.quantity,
      avgBuyPrice: holding.avgBuyPrice,
      evalAmount: holding.evalAmount,
    }
    const edit: HoldingEdit = {
      accountNumber: holding.accountNumber,
      name: holding.name,
      quantity: input.quantity,
      avgBuyPrice: input.avgBuyPrice ?? undefined,
      evalAmount: input.evalAmount,
      basedOn,
    }

    return apply({
      overrides: { ...overrides, [holding.name]: input.category },
      holdingEdits: [...withoutEdit(holdingEdits, holding), edit],
    })
  }

  function resetFileHolding(holding: Holding) {
    setHoldingTarget(null)
    return apply({ holdingEdits: withoutEdit(holdingEdits, holding) })
  }

  function removeManualHolding(item: ManualHolding) {
    setHoldingTarget(null)
    const nextOverrides = { ...overrides }
    delete nextOverrides[item.name]
    return apply({
      overrides: nextOverrides,
      manualHoldings: manualHoldings.filter((m) => m.id !== item.id),
    })
  }

  // ---------- 충돌 ----------

  // "내 값 유지" 는 지금 파일 값을 새 기준으로 삼는다는 뜻이다. 그래야 다음에
  // 파일이 또 바뀌었을 때만 다시 물어본다.
  function keepMine(edit: HoldingEdit, current: FileValues) {
    return apply({
      holdingEdits: holdingEdits.map((e) =>
        e.accountNumber === edit.accountNumber && e.name === edit.name ? { ...e, basedOn: current } : e,
      ),
    })
  }

  function useFileValue(edit: HoldingEdit) {
    return apply({
      holdingEdits: holdingEdits.filter(
        (e) => !(e.accountNumber === edit.accountNumber && e.name === edit.name),
      ),
    })
  }

  function addFromScreenshot(extracted: ExtractedHolding[]) {
    let nextAccounts = [...manualAccounts]
    const accountIds = new Map<string, string>()

    for (const h of extracted) {
      if (!h.accountNumber || accountIds.has(h.accountNumber)) continue
      const existing = nextAccounts.find((a) => a.accountNumber === h.accountNumber)
      if (existing) {
        accountIds.set(h.accountNumber, existing.id)
      } else {
        const id = crypto.randomUUID()
        accountIds.set(h.accountNumber, id)
        nextAccounts.push({ id, name: h.accountNumber, totalAsset: 0, accountNumber: h.accountNumber })
      }
    }

    const newHoldings: ManualHolding[] = extracted.map((h) => ({
      id: crypto.randomUUID(),
      name: h.name,
      evalAmount: h.evalAmount ?? 0,
      accountId: h.accountNumber ? accountIds.get(h.accountNumber) : undefined,
      quantity: h.quantity ?? undefined,
      avgBuyPrice: h.avgBuyPrice ?? undefined,
      profitLoss: h.profitLoss ?? undefined,
      profitRate: h.profitRate ?? undefined,
    }))

    const newKeys = new Set(newHoldings.map((h) => `${h.accountId ?? ''}::${h.name}`))
    const allHoldings = [
      ...manualHoldings.filter((h) => !newKeys.has(`${h.accountId ?? ''}::${h.name}`)),
      ...newHoldings,
    ]
    for (const [, acctId] of accountIds) {
      const total = allHoldings.filter((h) => h.accountId === acctId).reduce((s, h) => s + h.evalAmount, 0)
      nextAccounts = nextAccounts.map((a) =>
        a.id === acctId ? { ...a, totalAsset: Math.max(a.totalAsset, total) } : a,
      )
    }

    const nextMappings = { ...stockMappings }
    for (const h of extracted) {
      if (h.ticker && !nextMappings[h.name]) {
        nextMappings[h.name] = h.ticker
      }
    }

    return apply({
      manualAccounts: nextAccounts,
      manualHoldings: allHoldings,
      stockMappings: nextMappings,
    })
  }

  async function reset() {
    setFiles([])
    setManualAccounts([])
    setManualHoldings([])
    setHoldingEdits([])
    setStockMappings({})
    setSummary(null)
    setError(null)
    await clearState()
  }

  const superseded = supersededManualAccounts(manualAccounts, summary)
  const conflicts = conflictingEdits(holdingEdits, summary)
  const editingManual = holdingTarget?.kind === 'manual' ? holdingTarget.item : null

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
        onFiles={async (picked) => apply({ files: [...files, ...(await toUploadedFiles(picked))] })}
        busy={busy}
        compact={summary !== null}
      />

      {error && <p className="error">{error}</p>}

      {summary && conflicts.length > 0 && (
        <EditConflicts
          conflicts={conflicts}
          summary={summary}
          busy={busy}
          onKeepMine={keepMine}
          onUseFile={useFileValue}
        />
      )}

      {summary && (
        <AccountsPanel
          accounts={summary.accounts}
          holdings={summary.holdings}
          manualAccounts={manualAccounts}
          superseded={superseded}
          coveredAsset={summary.coveredAsset}
          busy={busy}
          onRemove={(number) => apply({ files: withoutAccount(files, number) })}
          onAddAccount={() => setAccountTarget('new')}
          onEditAccount={(id) => setAccountTarget(manualAccounts.find((a) => a.id === id) ?? null)}
          onRemoveAccount={removeAccount}
        />
      )}

      {summary && (
        <>
          <AllocationPie categories={summary.categories} coveredAsset={summary.coveredAsset} />
          <HoldingsTable
            holdings={summary.holdings}
            accounts={summary.accounts}
            mode={mode}
            onModeChange={setMode}
            busy={busy}
            onAddHolding={() => setHoldingTarget({ kind: 'new' })}
            onScreenshot={() => setShowScreenshot(true)}
            onEditHolding={openHoldingEditor}
          />
          <footer className="page-foot">
            <p>
              올린 잔고파일 {files.length}개와 집계 결과는 이 브라우저에만 저장됩니다. 서버는 계산 후
              아무것도 남기지 않습니다.
            </p>
            <button type="button" className="link" onClick={reset}>
              전부 지우기
            </button>
          </footer>
        </>
      )}

      {/* 보유 종목 표가 없으면 그쪽 "종목 추가" 버튼도 없다. 파일도 계좌도 없이
          종목 하나만 넣으려는 경로를 여기서 열어 준다. */}
      {restored && !summary && !busy && !error && (
        <div className="onboarding">
          <p>또는 직접 추가하거나 샘플로 체험해보세요</p>
          <div className="onboarding-actions">
            <button type="button" onClick={() => setAccountTarget('new')}>
              계좌 추가
            </button>
            <button type="button" onClick={() => setHoldingTarget({ kind: 'new' })}>
              종목 추가
            </button>
            <button type="button" onClick={() => setShowScreenshot(true)}>
              스크린샷으로 추가
            </button>
            <button
              type="button"
              className="sample-btn"
              onClick={async () => apply({ files: await toUploadedFiles(createSampleFiles()) })}
            >
              샘플 잔고파일로 체험하기
            </button>
          </div>
        </div>
      )}

      <AccountForm
        open={accountTarget !== null}
        editing={accountTarget === 'new' ? null : accountTarget}
        busy={busy}
        onClose={() => setAccountTarget(null)}
        onSubmit={submitAccount}
      />

      <HoldingForm
        target={holdingTarget}
        accounts={summary?.accounts ?? []}
        manualAccounts={manualAccounts.filter((account) => !superseded.includes(account))}
        busy={busy}
        onClose={() => setHoldingTarget(null)}
        onSubmitManual={submitManualHolding}
        onSubmitFile={submitFileHolding}
        onResetFile={resetFileHolding}
        onRemoveManual={editingManual ? () => removeManualHolding(editingManual) : undefined}
      />

      <ScreenshotImport
        open={showScreenshot}
        busy={busy}
        onClose={() => setShowScreenshot(false)}
        onConfirm={addFromScreenshot}
      />

      {showUnmatched && summary?.unmatched && summary.unmatched.length > 0 && (
        <UnmatchedResolver
          names={summary.unmatched}
          existing={stockMappings}
          busy={busy}
          onResolve={(resolved, nameUpdates) => {
            setShowUnmatched(false)
            const renamedHoldings = manualHoldings.map((h) => {
              const newName = nameUpdates[h.name]
              return newName ? { ...h, name: newName } : h
            })
            const cleanedMappings = { ...resolved }
            for (const oldName of Object.keys(nameUpdates)) {
              delete cleanedMappings[oldName]
            }
            apply({ stockMappings: cleanedMappings, manualHoldings: renamedHoldings })
          }}
          onClose={() => setShowUnmatched(false)}
        />
      )}
    </main>
  )
}

function withoutEdit(edits: HoldingEdit[], holding: Holding): HoldingEdit[] {
  const key = holdingKey(holding.accountNumber, holding.name)
  return edits.filter((e) => holdingKey(e.accountNumber, e.name) !== key)
}
