import { useEffect, useState } from 'react'
import { searchStocks } from '../api'
import { Modal } from './Modal'
import type { StockEntry, StockMappings } from '../types'

export type NameUpdates = Record<string, string>

interface Props {
  names: string[]
  existing: StockMappings
  busy: boolean
  onResolve: (mappings: StockMappings, nameUpdates: NameUpdates) => void
  onClose: () => void
}

const KIND_LABEL: Record<string, string> = {
  DS: '국내주식',
  DE: '국내ETF',
  FS: '해외주식',
  FE: '해외ETF',
}

export function UnmatchedResolver({ names, existing, busy, onResolve, onClose }: Props) {
  const unresolved = names.filter((n) => !(n in existing))
  const [current, setCurrent] = useState(0)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<StockEntry[]>([])
  const [searching, setSearching] = useState(false)
  const [selected, setSelected] = useState<StockEntry | null>(null)
  const [draft, setDraft] = useState<StockMappings>({})
  const [namesUpdates, setNamesUpdates] = useState<NameUpdates>({})
  const [suggestion, setSuggestion] = useState<StockEntry | null>(null)

  const name = unresolved[current]

  useEffect(() => {
    let cancelled = false
    setSuggestion(null)
    if (!name) return
    searchStocks(name).then((entries) => {
      if (!cancelled) setSuggestion(entries[0] ?? null)
    })
    return () => {
      cancelled = true
    }
  }, [name])

  if (!name) return null

  function chooseEntry(entry: StockEntry) {
    const nextDraft = { ...draft, [name]: entry.code }
    const nextNames = { ...namesUpdates, [name]: entry.name }
    setDraft(nextDraft)
    setNamesUpdates(nextNames)
    advance(nextDraft, nextNames)
  }

  async function doSearch(q: string) {
    setQuery(q)
    setSelected(null)
    if (q.trim().length < 1) {
      setResults([])
      return
    }
    setSearching(true)
    const entries = await searchStocks(q.trim())
    setResults(entries)
    setSearching(false)
  }

  function confirm() {
    if (!selected) return
    chooseEntry(selected)
  }

  function skip() {
    const nextDraft = { ...draft, [name]: '' }
    setDraft(nextDraft)
    advance(nextDraft, namesUpdates)
  }

  function advance(nextDraft: StockMappings, nextNames: NameUpdates) {
    setQuery('')
    setResults([])
    setSelected(null)
    if (current + 1 >= unresolved.length) {
      onResolve({ ...existing, ...nextDraft }, nextNames)
    } else {
      setCurrent(current + 1)
    }
  }

  return (
    <Modal
      open
      title={`종목 확인 (${current + 1}/${unresolved.length})`}
      onClose={onClose}
    >
      <div className="unmatched-resolver">
        <p className="unmatched-name">
          <strong>{name}</strong>을 종목 목록에서 찾지 못했습니다.
        </p>
        <p className="unmatched-hint">검색해서 맞는 종목을 고르거나, 기타로 처리하세요.</p>

        {suggestion && (
          <div className="unmatched-suggestion">
            <p className="unmatched-suggestion-label">혹시 이 종목인가요?</p>
            <div className="unmatched-suggestion-card">
              <span className="stock-name">{suggestion.name}</span>
              <span className="stock-meta">
                {suggestion.code} &middot; {KIND_LABEL[suggestion.kind] ?? suggestion.kind}
              </span>
              <button
                type="button"
                className="modal-confirm"
                disabled={busy}
                onClick={() => chooseEntry(suggestion)}
              >
                맞아요
              </button>
            </div>
          </div>
        )}

        <input
          className="unmatched-search"
          value={query}
          onChange={(e) => doSearch(e.target.value)}
          placeholder="종목명 또는 코드로 검색"
          disabled={busy}
          autoFocus
        />

        {results.length > 0 && (
          <ul className="stock-search-dropdown static">
            {results.map((entry) => (
              <li
                key={entry.code + entry.name}
                className={selected?.code === entry.code ? 'selected' : ''}
                onMouseDown={() => setSelected(entry)}
              >
                <span className="stock-name">{entry.name}</span>
                <span className="stock-meta">
                  {entry.code} &middot; {KIND_LABEL[entry.kind] ?? entry.kind}
                </span>
              </li>
            ))}
          </ul>
        )}

        {selected && (
          <p className="unmatched-selected">
            <strong>{selected.name}</strong> ({selected.code}) 선택됨
          </p>
        )}

        {query && !searching && results.length === 0 && (
          <p className="unmatched-empty">검색 결과가 없습니다.</p>
        )}

        <footer className="modal-actions">
          <button type="button" className="modal-cancel" disabled={busy} onClick={skip}>
            기타 (시세 조회 불가)
          </button>
          <button
            type="button"
            className="modal-confirm"
            disabled={busy || !selected}
            onClick={confirm}
          >
            확인
          </button>
        </footer>
      </div>
    </Modal>
  )
}
