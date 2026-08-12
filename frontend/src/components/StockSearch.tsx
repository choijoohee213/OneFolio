import { useEffect, useRef, useState } from 'react'
import { searchStocks } from '../api'
import type { StockEntry } from '../types'

interface Props {
  value: string
  onChange: (name: string, code: string) => void
  disabled?: boolean
}

const KIND_LABEL: Record<string, string> = {
  DS: '국내주식',
  DE: '국내ETF',
  FS: '해외주식',
  FE: '해외ETF',
}

export function StockSearch({ value, onChange, disabled }: Props) {
  const [query, setQuery] = useState(value)
  const [results, setResults] = useState<StockEntry[]>([])
  const [open, setOpen] = useState(false)
  const [highlight, setHighlight] = useState(-1)
  const timerRef = useRef(0)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setQuery(value)
  }, [value])

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  function search(q: string) {
    setQuery(q)
    clearTimeout(timerRef.current)
    if (q.trim().length < 1) {
      setResults([])
      setOpen(false)
      return
    }
    timerRef.current = window.setTimeout(async () => {
      const entries = await searchStocks(q.trim())
      setResults(entries)
      setOpen(entries.length > 0)
      setHighlight(-1)
    }, 200)
  }

  function pick(entry: StockEntry) {
    setQuery(entry.name)
    setOpen(false)
    onChange(entry.name, entry.code)
  }

  function pickDirect() {
    setOpen(false)
    onChange(query.trim(), '')
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (!open) return
    const total = results.length + 1
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setHighlight((h) => (h + 1) % total)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHighlight((h) => (h - 1 + total) % total)
    } else if (e.key === 'Enter' && highlight >= 0) {
      e.preventDefault()
      if (highlight < results.length) {
        pick(results[highlight])
      } else {
        pickDirect()
      }
    } else if (e.key === 'Escape') {
      setOpen(false)
    }
  }

  return (
    <div className="stock-search" ref={containerRef}>
      <input
        value={query}
        onChange={(e) => search(e.target.value)}
        onFocus={() => results.length > 0 && setOpen(true)}
        onKeyDown={handleKeyDown}
        placeholder="종목명 또는 코드로 검색"
        disabled={disabled}
        required
      />
      {open && (
        <ul className="stock-search-dropdown">
          {results.map((entry, i) => (
            <li
              key={entry.code + entry.name}
              className={i === highlight ? 'highlighted' : ''}
              onMouseDown={() => pick(entry)}
            >
              <span className="stock-name">{entry.name}</span>
              <span className="stock-meta">
                {entry.code} &middot; {KIND_LABEL[entry.kind] ?? entry.kind}
              </span>
            </li>
          ))}
          <li
            className={`stock-direct ${highlight === results.length ? 'highlighted' : ''}`}
            onMouseDown={pickDirect}
          >
            "{query}" 직접 입력 (시세 조회 불가)
          </li>
        </ul>
      )}
    </div>
  )
}
