import { useEffect, useRef, useState } from 'react'
import { fetchQuotes, toUploadedFiles } from './api'
import { createSampleFiles } from './sampleData'
import { recompute, withoutAccount } from './collection'
import { AccountForm, type AccountInput } from './components/AccountForm'
import { AccountsPanel } from './components/AccountsPanel'
import { AllocationPie } from './components/AllocationPie'
import { Treemap } from './components/Treemap'
import { ProfitBars } from './components/ProfitBars'
import { EditConflicts } from './components/EditConflicts'
import { AddMenu } from './components/AddMenu'
import { FileUploadModal } from './components/FileUploadModal'
import { HoldingForm, type FileInput, type HoldingTarget, type ManualInput } from './components/HoldingForm'
import { ScreenshotImport } from './components/ScreenshotImport'
import { HoldingsTable, type GroupMode } from './components/HoldingsTable'
import { ThemeToggle } from './components/ThemeToggle'
import { UnmatchedResolver } from './components/UnmatchedResolver'
import { won } from './format'
import { applyLiveQuotes, type Quote } from './liveQuotes'
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
  const [showFileUpload, setShowFileUpload] = useState(false)
  const [liveQuotes, setLiveQuotes] = useState<Record<string, Quote> | null>(null)
  const [usdKrw, setUsdKrw] = useState<number | null>(null)
  const [showLive, setShowLive] = useState(false)
  const [showUSD, setShowUSD] = useState(false)

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

  // market 은 나중에 추가된 값이라 그 전에 저장된 집계에는 없다. 시장·통화 차트가
  // "미상"만 띄우지 않도록, 복원 직후 한 번 다시 계산해 채운다.
  useEffect(() => {
    if (!restored || !summary) return
    if (summary.holdings.some((holding) => holding.code && !holding.market)) void apply({})
    // 복원 직후 한 번만 본다. summary 를 넣으면 재계산 결과에 다시 반응한다.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [restored])

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

  // 종목코드를 아는 것만 새로고침할 수 있다. 받은 시세는 저장하지 않고
  // 화면에만 겹쳐 보여준다 — 토글을 끄면 바로 원래(파일·수동입력) 값으로
  // 돌아간다. 평단가가 원화가 아니라 달러로 들어간 종목은 손익이 잘못
  // 나올 수 있어(입력 시점 통화를 저장하지 않음), 되돌릴 수 있는 게 중요하다.
  // 새 요청이 이전 요청보다 먼저 끝나는 경우를 막는다 — 재시도로 한 번의
  // 조회가 폴링 주기보다 오래 걸리면 다음 틱은 건너뛴다(쌓이지 않고).
  const fetchingQuotesRef = useRef(false)

  async function fetchLiveQuotes(): Promise<boolean> {
    if (!summary || fetchingQuotesRef.current) return false
    const withCode = summary.holdings.filter((h) => h.code)
    if (withCode.length === 0) return false

    fetchingQuotesRef.current = true
    try {
      const codes = [...new Set(withCode.map((h) => h.code!))]
      const result = await fetchQuotes(codes)
      setLiveQuotes(result.quotes)
      setUsdKrw(result.usdKrw ?? null)
      return true
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
      return false
    } finally {
      fetchingQuotesRef.current = false
    }
  }

  // "실시간 시세"와 "달러로 보기" 둘 다 이 시세 데이터가 있어야 동작한다.
  // 어느 쪽을 먼저 켜도 없으면 그때 한 번 받아 온다. 이미 있으면 그대로 쓴다 —
  // 켜져 있는 동안의 주기적 갱신은 아래 useEffect 가 따로 맡는다.
  async function ensureQuotesLoaded(): Promise<boolean> {
    if (liveQuotes !== null) return true
    setBusy(true)
    setError(null)
    try {
      return await fetchLiveQuotes()
    } finally {
      setBusy(false)
    }
  }

  async function toggleLiveQuotes() {
    if (showLive) {
      setShowLive(false)
      return
    }
    if (await ensureQuotesLoaded()) setShowLive(true)
  }

  async function toggleUSD() {
    if (showUSD) {
      setShowUSD(false)
      return
    }
    if (await ensureQuotesLoaded()) setShowUSD(true)
  }

  // 실시간 시세가 켜져 있는 동안엔 3초마다 조용히(busy 표시 없이) 다시 받아
  // 화면 값을 갱신한다. REST 폴링이 낼 수 있는 한도에서 최대한 자주 — 한투
  // API 자체가 완전 틱단위 실시간은 아니라 이보다 빨라도 체감 이득이 적고,
  // 종목이 많으면 한 번 조회에 걸리는 시간(순차 호출)도 있어 너무 빠르면
  // 요청이 겹친다. 꺼지거나 언마운트되면 멈춘다.
  useEffect(() => {
    if (!showLive) return
    const id = setInterval(fetchLiveQuotes, 3_000)
    return () => clearInterval(id)
  }, [showLive, summary])

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
      // 평단가가 달러로 찍혀 있으면(currency==='USD') 그 숫자를 그대로 저장하면
      // 본 화면에서 원화인 척 표시돼 이상해 보인다. evalAmount·profitLoss 는
      // 이미 정확한 원화값이라 여기서 역산하면 새로 환율을 몰라도 원화 평단가가
      // 나온다: 매입금액(원) = evalAmount - profitLoss, 평단가(원) = 매입금액 / 수량.
      avgBuyPrice: krwAvgBuyPrice(h),
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
  const displaySummary = showLive && summary && liveQuotes ? applyLiveQuotes(summary, liveQuotes, usdKrw) : summary

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
        {displaySummary && (
          <div className="total">
            {/* 계좌가 하나도 없으면(계좌 없이 종목만 있을 때) 계좌 총합은 0원이라
                아무것도 없는 것처럼 보인다. 그럴 땐 실제로 집계된 금액을 보여준다. */}
            <span className="total-label">
              {displaySummary.accounts.length > 0 ? '계좌 총합' : '직접 추가한 자산'}
            </span>
            <strong>
              {won(displaySummary.accounts.length > 0 ? displaySummary.totalAsset : displaySummary.coveredAsset)}
            </strong>
          </div>
        )}
        {summary && (
          <AddMenu
            busy={busy}
            onFileUpload={() => setShowFileUpload(true)}
            onScreenshot={() => setShowScreenshot(true)}
            onAddAccount={() => setAccountTarget('new')}
            onAddHolding={() => setHoldingTarget({ kind: 'new' })}
          />
        )}
      </header>

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

      {displaySummary && (
        <AccountsPanel
          accounts={displaySummary.accounts}
          holdings={displaySummary.holdings}
          manualAccounts={manualAccounts}
          superseded={superseded}
          coveredAsset={displaySummary.coveredAsset}
          busy={busy}
          onRemove={(number) => apply({ files: withoutAccount(files, number) })}
          onEditAccount={(id) => setAccountTarget(manualAccounts.find((a) => a.id === id) ?? null)}
          onRemoveAccount={removeAccount}
        />
      )}

      {displaySummary && (
        <>
          <AllocationPie
            categories={displaySummary.categories}
            holdings={displaySummary.holdings}
            accounts={displaySummary.accounts}
            coveredAsset={displaySummary.coveredAsset}
          />
          <Treemap holdings={displaySummary.holdings} coveredAsset={displaySummary.coveredAsset} />
          <ProfitBars holdings={displaySummary.holdings} />
          <HoldingsTable
            holdings={displaySummary.holdings}
            accounts={displaySummary.accounts}
            mode={mode}
            onModeChange={setMode}
            busy={busy}
            onEditHolding={openHoldingEditor}
            showLive={showLive}
            onToggleLive={toggleLiveQuotes}
            showUSD={showUSD}
            onToggleUSD={toggleUSD}
            quotes={liveQuotes}
            usdKrw={usdKrw}
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

      {restored && !summary && busy && <p className="home-busy">계산하는 중…</p>}

      {/* 보유 종목 표가 없으면 그쪽 "종목 추가" 버튼도 없다. 파일도 계좌도 없이
          종목 하나만 넣으려는 경로를 여기서 열어 준다. */}
      {restored && !summary && !busy && !error && (
        <div className="home">
          <div className="home-grid">
            <button type="button" className="home-card" onClick={() => setShowFileUpload(true)}>
              <span className="home-card-title">잔고파일 추가</span>
              <span className="home-card-desc">미래에셋증권 계좌별 잔고 엑셀 업로드</span>
            </button>
            <button type="button" className="home-card" onClick={() => setShowScreenshot(true)}>
              <span className="home-card-title">스크린샷으로 추가</span>
              <span className="home-card-desc">증권 앱 캡처에서 자동으로 인식</span>
            </button>
            <button type="button" className="home-card" onClick={() => setAccountTarget('new')}>
              <span className="home-card-title">계좌 추가</span>
              <span className="home-card-desc">총액만 적어 계좌로 등록</span>
            </button>
            <button type="button" className="home-card" onClick={() => setHoldingTarget({ kind: 'new' })}>
              <span className="home-card-title">종목 추가</span>
              <span className="home-card-desc">보유 종목을 하나씩 직접 입력</span>
            </button>
          </div>

          <div className="home-divider">
            <span>또는</span>
          </div>
          <button
            type="button"
            className="home-sample-btn"
            onClick={async () => apply({ files: await toUploadedFiles(createSampleFiles()) })}
          >
            실제 데이터 없이 샘플로 둘러보기
          </button>
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

      <FileUploadModal
        open={showFileUpload}
        busy={busy}
        onClose={() => setShowFileUpload(false)}
        onFiles={async (picked) => apply({ files: [...files, ...(await toUploadedFiles(picked))] })}
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

// 평단가가 달러로 찍힌 종목은 evalAmount·profitLoss(이미 정확한 원화값)에서
// 매입금액을 역산해 원화 평단가를 낸다. 새로 환율을 몰라도 되고, 백엔드가
// 계산한 값과 항상 일치한다.
function krwAvgBuyPrice(h: ExtractedHolding): number | undefined {
  if (h.currency === 'USD' && h.evalAmount != null && h.profitLoss != null && h.quantity) {
    return (h.evalAmount - h.profitLoss) / h.quantity
  }
  return h.avgBuyPrice ?? undefined
}
