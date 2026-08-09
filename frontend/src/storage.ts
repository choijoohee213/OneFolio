import type {
  HoldingEdit,
  ManualAccount,
  ManualHolding,
  Overrides,
  Summary,
  UploadedFile,
} from './types'
import { holdingKey, isConflicting, MANUAL_ACCOUNT_PREFIX } from './types'

// 자산 데이터는 서버에 남기지 않는다. 올린 잔고파일, 직접 추가한 계좌·종목, 마지막
// 집계 결과, 분류 매핑은 브라우저 안에만 둔다. 파일을 들고 있어야 계좌를 추가할 때
// 누적 집계가 된다.
const DB_NAME = 'onefolio'
const STORE = 'state'
const VERSION = 1

interface SavedState {
  files: UploadedFile[]
  manualAccounts: ManualAccount[]
  manualHoldings: ManualHolding[]
  holdingEdits: HoldingEdit[]
  summary: Summary | null
  overrides: Overrides
  savedAt: string
}

// 응답 모양이 바뀌면 예전 기록이 남아 화면이 깨진다. 버전 번호로는 못 잡는 경우가
// 있어서(같은 번호로 저장된 옛 응답) 화면이 실제로 읽는 필드가 다 있는지 본다.
function renderable(summary: unknown): summary is Summary {
  const s = summary as Summary | null
  return (
    !!s &&
    typeof s.coveredAsset === 'number' &&
    typeof s.totalAsset === 'number' &&
    Array.isArray(s.accounts) &&
    Array.isArray(s.categories) &&
    Array.isArray(s.holdings) &&
    Array.isArray(s.sources)
  )
}

function storedFiles(files: unknown): UploadedFile[] {
  if (!Array.isArray(files)) return []
  return files.filter(
    (file: UploadedFile) =>
      typeof file?.name === 'string' &&
      file.data instanceof ArrayBuffer &&
      Array.isArray(file.accounts),
  )
}

function storedManualAccounts(manualAccounts: unknown): ManualAccount[] {
  if (!Array.isArray(manualAccounts)) return []
  return manualAccounts.filter(
    (a: ManualAccount) =>
      typeof a?.id === 'string' && typeof a.name === 'string' && typeof a.totalAsset === 'number',
  )
}

/** 서버가 파일로 갈음해 버린 수동 계좌. 응답의 계좌 목록에 없으면 대체된 것이다. */
export function supersededManualAccounts(
  manualAccounts: ManualAccount[],
  summary: Summary | null,
): ManualAccount[] {
  if (!summary) return []
  const shown = new Set(summary.accounts.map((account) => account.number))
  return manualAccounts.filter((account) => !shown.has(MANUAL_ACCOUNT_PREFIX + account.id))
}

function storedManualHoldings(manualHoldings: unknown): ManualHolding[] {
  if (!Array.isArray(manualHoldings)) return []
  return manualHoldings.filter(
    (m: ManualHolding) =>
      typeof m?.id === 'string' && typeof m.name === 'string' && typeof m.evalAmount === 'number',
  )
}

function storedHoldingEdits(edits: unknown): HoldingEdit[] {
  if (!Array.isArray(edits)) return []
  return edits.filter(
    (e: HoldingEdit) =>
      typeof e?.accountNumber === 'string' && typeof e.name === 'string' && !!e.basedOn,
  )
}

/** 파일 값이 수정 당시와 달라진 종목. 사용자가 어느 쪽을 쓸지 골라야 한다. */
export function conflictingEdits(edits: HoldingEdit[], summary: Summary | null): HoldingEdit[] {
  if (!summary) return []
  const fileValues = new Map(
    summary.holdings
      .filter((h) => h.original)
      .map((h) => [holdingKey(h.accountNumber, h.name), h.original!]),
  )
  return edits.filter((edit) => {
    const current = fileValues.get(holdingKey(edit.accountNumber, edit.name))
    return current !== undefined && isConflicting(edit, current)
  })
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, VERSION)
    request.onupgradeneeded = () => request.result.createObjectStore(STORE)
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

function run<T>(mode: IDBTransactionMode, action: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  return openDB().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const request = action(db.transaction(STORE, mode).objectStore(STORE))
        request.onsuccess = () => resolve(request.result)
        request.onerror = () => reject(request.error)
      }),
  )
}

export async function loadState(): Promise<SavedState | null> {
  try {
    const state = await run<SavedState | undefined>('readonly', (store) => store.get('current'))
    if (!state) return null
    return {
      ...state,
      files: storedFiles(state.files),
      manualAccounts: storedManualAccounts(state.manualAccounts),
      manualHoldings: storedManualHoldings(state.manualHoldings),
      holdingEdits: storedHoldingEdits(state.holdingEdits),
      summary: renderable(state.summary) ? state.summary : null,
      overrides: state.overrides ?? {},
    }
  } catch {
    return null
  }
}

export async function saveState(
  files: UploadedFile[],
  manualAccounts: ManualAccount[],
  manualHoldings: ManualHolding[],
  holdingEdits: HoldingEdit[],
  summary: Summary | null,
  overrides: Overrides,
): Promise<void> {
  const state: SavedState = {
    files,
    manualAccounts,
    manualHoldings,
    holdingEdits,
    summary,
    overrides,
    savedAt: new Date().toISOString(),
  }
  try {
    await run('readwrite', (store) => store.put(state, 'current'))
  } catch {
    // 저장 실패는 화면 동작을 막지 않는다
  }
}

export async function clearState(): Promise<void> {
  try {
    await run('readwrite', (store) => store.delete('current'))
  } catch {
    // 무시
  }
}
