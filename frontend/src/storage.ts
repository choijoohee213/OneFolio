import type { Overrides, Summary } from './types'

// 자산 데이터는 서버에 남기지 않는다. 마지막 집계 결과와 사용자 분류 매핑만
// 브라우저 안에 보관해서 앱을 다시 열었을 때 파일을 또 올리지 않아도 되게 한다.
const DB_NAME = 'onefolio'
const STORE = 'state'
const VERSION = 1

interface SavedState {
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
    Array.isArray(s.holdings)
  )
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
      summary: renderable(state.summary) ? state.summary : null,
      overrides: state.overrides ?? {},
    }
  } catch {
    return null
  }
}

export async function saveState(summary: Summary | null, overrides: Overrides): Promise<void> {
  const state: SavedState = { summary, overrides, savedAt: new Date().toISOString() }
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
