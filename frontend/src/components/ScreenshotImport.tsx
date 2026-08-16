import { useEffect, useRef, useState } from 'react'
import { extractFromScreenshot } from '../api'
import type { ExtractedHolding } from '../types'

interface Props {
  open: boolean
  busy: boolean
  onClose: () => void
  onConfirm: (holdings: ExtractedHolding[]) => void
}

function dedup(holdings: ExtractedHolding[]): ExtractedHolding[] {
  const seen = new Map<string, ExtractedHolding>()
  for (const h of holdings) {
    const key = `${h.accountNumber ?? ''}::${h.name}`
    if (!seen.has(key)) seen.set(key, h)
  }
  return [...seen.values()]
}

type Step = 'upload' | 'preview' | 'extracting' | 'review'

function formatKRW(v: number | null): string {
  if (v == null) return ''
  return Math.round(v).toLocaleString('ko-KR')
}

function formatSignedKRW(v: number | null): string {
  if (v == null) return ''
  const s = Math.round(v).toLocaleString('ko-KR')
  return v > 0 ? `+${s}` : s
}

function formatRate(v: number | null): string {
  if (v == null) return ''
  const s = v.toFixed(2)
  return v > 0 ? `+${s}` : s
}

function parseKRW(s: string): number | null {
  const raw = s.replace(/[,+]/g, '')
  if (raw === '') return null
  const n = Number(raw)
  return isNaN(n) ? null : n
}

async function pooledMap<T, R>(items: T[], fn: (item: T) => Promise<R>, concurrency: number): Promise<R[]> {
  const results: R[] = new Array(items.length)
  let next = 0
  async function worker() {
    while (next < items.length) {
      const i = next++
      results[i] = await fn(items[i])
    }
  }
  await Promise.all(Array.from({ length: Math.min(concurrency, items.length) }, () => worker()))
  return results
}

const MAX_DIMENSION = 1024

function resizeImage(file: File): Promise<File> {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => {
      const { width, height } = img
      if (width <= MAX_DIMENSION && height <= MAX_DIMENSION) {
        URL.revokeObjectURL(img.src)
        resolve(file)
        return
      }
      const scale = MAX_DIMENSION / Math.max(width, height)
      const canvas = document.createElement('canvas')
      canvas.width = Math.round(width * scale)
      canvas.height = Math.round(height * scale)
      canvas.getContext('2d')!.drawImage(img, 0, 0, canvas.width, canvas.height)
      URL.revokeObjectURL(img.src)
      canvas.toBlob(
        (blob) => resolve(new File([blob!], file.name, { type: 'image/jpeg' })),
        'image/jpeg',
        0.85,
      )
    }
    img.src = URL.createObjectURL(file)
  })
}

function useProgress(active: boolean, tick: number) {
  const [pct, setPct] = useState(0)
  useEffect(() => {
    if (!active) { setPct(0); return }
    setPct(5)
    const t1 = setTimeout(() => setPct(20), 300)
    const t2 = setTimeout(() => setPct(40), 1200)
    const t3 = setTimeout(() => setPct(60), 3000)
    const t4 = setTimeout(() => setPct(75), 5000)
    const t5 = setTimeout(() => setPct(85), 8000)
    const t6 = setTimeout(() => setPct(92), 12000)
    return () => { clearTimeout(t1); clearTimeout(t2); clearTimeout(t3); clearTimeout(t4); clearTimeout(t5); clearTimeout(t6) }
  }, [active, tick])
  return pct
}

export function ScreenshotImport({ open, busy, onClose, onConfirm }: Props) {
  const dialog = useRef<HTMLDialogElement>(null)
  const input = useRef<HTMLInputElement>(null)
  const addFromReview = useRef(false)
  const [step, setStep] = useState<Step>('upload')
  const [holdings, setHoldings] = useState<ExtractedHolding[]>([])
  const [error, setError] = useState<string | null>(null)
  const [imageFiles, setImageFiles] = useState<File[]>([])
  const [previews, setPreviews] = useState<string[]>([])
  const [fullImageIndex, setFullImageIndex] = useState<number | null>(null)
  const [processingIndex, setProcessingIndex] = useState(0)
  const [processingTotal, setProcessingTotal] = useState(1)
  const [activeTab, setActiveTab] = useState<string | null>(null)

  const withinPct = useProgress(step === 'extracting', processingIndex)
  const overallProgress = processingTotal > 0
    ? Math.round(((processingIndex + withinPct / 100) / processingTotal) * 100)
    : 0

  function reset() {
    setStep('upload')
    setHoldings([])
    setError(null)
    setFullImageIndex(null)
    setActiveTab(null)
    setProcessingIndex(0)
    setProcessingTotal(1)
    previews.forEach(URL.revokeObjectURL)
    setPreviews([])
    setImageFiles([])
    addFromReview.current = false
  }

  function handleClose() {
    reset()
    onClose()
  }

  useEffect(() => {
    const el = dialog.current
    if (!el) return
    if (open && !el.open) el.showModal()
    if (!open && el.open) el.close()
  }, [open])

  function handleInputFiles(fileList: FileList | null) {
    if (!fileList || fileList.length === 0) return
    const files = Array.from(fileList).filter((f) => f.type.startsWith('image/'))
    if (files.length === 0) {
      setError('이미지 파일만 가능합니다')
      return
    }

    const newPreviews = files.map((f) => URL.createObjectURL(f))
    setImageFiles((prev) => [...prev, ...files])
    setPreviews((prev) => [...prev, ...newPreviews])
    setError(null)

    if (addFromReview.current) {
      addFromReview.current = false
      doExtract(files)
    } else {
      setStep('preview')
    }
  }

  function removeImage(index: number) {
    URL.revokeObjectURL(previews[index])
    const nextFiles = imageFiles.filter((_, i) => i !== index)
    const nextPreviews = previews.filter((_, i) => i !== index)
    setImageFiles(nextFiles)
    setPreviews(nextPreviews)
    if (nextFiles.length === 0) setStep('upload')
  }

  async function doExtract(files: File[]) {
    if (files.length === 0) return
    setStep('extracting')
    setProcessingIndex(0)
    setProcessingTotal(files.length)

    try {
      let completed = 0
      const results = await pooledMap(files, async (file) => {
        const resized = await resizeImage(file)
        const result = await extractFromScreenshot(resized)
        completed++
        setProcessingIndex(completed)
        return result
      }, 2)

      const newHoldings: ExtractedHolding[] = []
      for (const result of results) {
        if (result.holdings) newHoldings.push(...result.holdings)
      }

      if (newHoldings.length === 0 && holdings.length === 0) {
        setError('종목을 찾지 못했습니다. 다른 캡처를 시도해보세요.')
        setStep('preview')
        return
      }

      setHoldings((prev) => dedup([...prev, ...newHoldings]))
      setStep('review')
    } catch (cause) {
      const msg = cause instanceof Error ? cause.message : String(cause)
      setError(msg.includes('429') || msg.includes('quota') ? 'API 요청 한도를 초과했습니다. 잠시 후 다시 시도해주세요.' : msg)
      setStep(holdings.length > 0 ? 'review' : 'preview')
    }
  }

  function handleAddMore() {
    addFromReview.current = true
    input.current?.click()
  }

  function updateHolding(index: number, field: keyof ExtractedHolding, value: string) {
    setHoldings((prev) =>
      prev.map((h, i) => {
        if (i !== index) return h
        if (field === 'name') return { ...h, name: value }
        const num = parseKRW(value)
        if (value !== '' && num === null) return h
        return { ...h, [field]: num }
      }),
    )
  }

  function removeHolding(index: number) {
    setHoldings((prev) => prev.filter((_, i) => i !== index))
  }

  const accountKeys = [...new Set(holdings.map((h) => h.accountNumber).filter(Boolean))] as string[]
  const hasMultipleAccounts = accountKeys.length > 1
  const visibleHoldings = activeTab
    ? holdings.filter((h) => h.accountNumber === activeTab)
    : holdings

  return (
    <dialog
      ref={dialog}
      className={`modal${step === 'review' ? ' modal-wide' : ''}`}
      onClose={handleClose}
      onClick={(event) => {
        if (step === 'extracting') return
        if (event.target === dialog.current) handleClose()
      }}
    >
      <div className="modal-body">
        <input
          ref={input}
          type="file"
          accept="image/*"
          multiple
          hidden
          onChange={(e) => {
            handleInputFiles(e.target.files)
            e.target.value = ''
          }}
        />

        <header className="modal-head">
          <h3>
            {step === 'review'
              ? '추출 결과 확인'
              : step === 'preview'
                ? '이미지 확인'
                : '스크린샷으로 종목 추가'}
          </h3>
          <button type="button" className="modal-close" onClick={handleClose}>
            닫기
          </button>
        </header>

        {step === 'upload' && (
          <div className="screenshot-upload">
            <div
              className="screenshot-drop"
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault()
                handleInputFiles(e.dataTransfer.files)
              }}
            >
              <p>증권 앱 캡처를 여기에 끌어다 놓거나</p>
              <button type="button" onClick={() => input.current?.click()}>
                이미지 선택
              </button>
            </div>
            {error && <p className="screenshot-error">{error}</p>}
          </div>
        )}

        {step === 'preview' && (
          <div className="screenshot-preview-step">
            <div className="screenshot-thumbnails">
              {previews.map((url, i) => (
                <div key={i} className="screenshot-thumb">
                  <img
                    src={url}
                    alt={`캡처 ${i + 1}`}
                    onClick={() => setFullImageIndex(i)}
                  />
                  <button
                    type="button"
                    className="thumb-remove"
                    onClick={() => removeImage(i)}
                  >
                    ×
                  </button>
                </div>
              ))}
              <button
                type="button"
                className="thumb-add"
                onClick={() => input.current?.click()}
              >
                +
              </button>
            </div>
            {error && <p className="screenshot-error">{error}</p>}
            <footer className="modal-actions">
              <button
                type="button"
                className="modal-cancel"
                onClick={() => {
                  previews.forEach(URL.revokeObjectURL)
                  setImageFiles([])
                  setPreviews([])
                  setError(null)
                  input.current?.click()
                }}
              >
                재선택
              </button>
              <button
                type="button"
                className="modal-confirm"
                disabled={imageFiles.length === 0}
                onClick={() => doExtract(imageFiles)}
              >
                종목 추출{imageFiles.length > 1 ? ` (${imageFiles.length}장)` : ''}
              </button>
            </footer>
          </div>
        )}

        {step === 'extracting' && (
          <div className="screenshot-extracting">
            {previews.length > 1 && (
              <div className="screenshot-thumbnails extracting">
                {previews.map((url, i) => (
                  <div
                    key={i}
                    className={`screenshot-thumb ${i < processingIndex ? 'done' : 'processing'}`}
                  >
                    <img src={url} alt={`캡처 ${i + 1}`} />
                  </div>
                ))}
              </div>
            )}
            {previews.length === 1 && (
              <img src={previews[0]} alt="캡처" className="screenshot-preview" />
            )}
            <div className="extracting-loader">
              <div className="extracting-progress-wrap">
                <div
                  className="extracting-progress-bar"
                  style={{ width: `${overallProgress}%` }}
                />
              </div>
              <p className="extracting-text">
                {overallProgress}%
                {processingTotal > 1
                  ? ` · ${processingIndex}/${processingTotal}개 완료`
                  : ''}
                {' · 종목을 읽고 있습니다...'}
              </p>
            </div>
          </div>
        )}

        {step === 'review' && (
          <div className="screenshot-review">
            <div className="screenshot-thumbnails review">
              {previews.map((url, i) => (
                <div key={i} className="screenshot-thumb">
                  <img
                    src={url}
                    alt={`캡처 ${i + 1}`}
                    onClick={() => setFullImageIndex(i)}
                  />
                </div>
              ))}
              <button type="button" className="thumb-add" onClick={handleAddMore}>
                +
              </button>
            </div>
            {hasMultipleAccounts && (
              <div className="review-tabs" role="tablist">
                <button
                  type="button"
                  aria-pressed={activeTab === null}
                  onClick={() => setActiveTab(null)}
                >
                  전체 ({holdings.length})
                </button>
                {accountKeys.map((key) => {
                  const acct = holdings.find((h) => h.accountNumber === key)
                  const label = [key, acct?.accountType].filter(Boolean).join(' ')
                  const count = holdings.filter((h) => h.accountNumber === key).length
                  return (
                    <button
                      key={key}
                      type="button"
                      aria-pressed={activeTab === key}
                      onClick={() => setActiveTab(key)}
                    >
                      {label} ({count})
                    </button>
                  )
                })}
              </div>
            )}
            {!hasMultipleAccounts && accountKeys.length === 1 && (
              <p className="review-account">
                {[accountKeys[0], holdings[0]?.accountType].filter(Boolean).join(' ')}
              </p>
            )}
            <p className="review-hint">
              추출된 {holdings.length}개 종목을 확인하세요. 틀린 값은 직접 수정할 수 있습니다.
            </p>
            <div className="review-table-wrap">
              <table className="review-table">
                <colgroup>
                  <col className="col-name" />
                  <col className="col-qty" />
                  <col className="col-eval" />
                  <col className="col-avg" />
                  <col className="col-pl" />
                  <col className="col-rate" />
                  <col className="col-act" />
                </colgroup>
                <thead>
                  <tr>
                    <th>종목명</th>
                    <th>수량</th>
                    <th>평가금액(원)</th>
                    <th>평균매입가(원)</th>
                    <th>평가손익(원)</th>
                    <th>손익률(%)</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {visibleHoldings.map((h) => {
                    const idx = holdings.indexOf(h)
                    return (
                      <tr key={idx}>
                        <td>
                          <input
                            value={h.name}
                            onChange={(e) => updateHolding(idx, 'name', e.target.value)}
                          />
                        </td>
                        <td>
                          <input
                            type="number"
                            value={h.quantity ?? ''}
                            onChange={(e) => updateHolding(idx, 'quantity', e.target.value)}
                          />
                        </td>
                        <td>
                          <input
                            value={formatKRW(h.evalAmount)}
                            onChange={(e) => updateHolding(idx, 'evalAmount', e.target.value)}
                          />
                        </td>
                        <td>
                          <input
                            value={formatKRW(h.avgBuyPrice)}
                            onChange={(e) => updateHolding(idx, 'avgBuyPrice', e.target.value)}
                          />
                        </td>
                        <td>
                          <input
                            className={
                              h.profitLoss != null
                                ? h.profitLoss >= 0
                                  ? 'profit'
                                  : 'loss'
                                : undefined
                            }
                            value={formatSignedKRW(h.profitLoss)}
                            onChange={(e) => updateHolding(idx, 'profitLoss', e.target.value)}
                          />
                        </td>
                        <td>
                          <input
                            className={
                              h.profitRate != null
                                ? h.profitRate >= 0
                                  ? 'profit'
                                  : 'loss'
                                : undefined
                            }
                            value={formatRate(h.profitRate)}
                            onChange={(e) => updateHolding(idx, 'profitRate', e.target.value)}
                          />
                        </td>
                        <td>
                          <button
                            type="button"
                            className="link danger"
                            onClick={() => removeHolding(idx)}
                          >
                            삭제
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            {error && <p className="screenshot-error">{error}</p>}
            <footer className="modal-actions">
              <button type="button" className="modal-cancel" onClick={handleClose}>
                취소
              </button>
              <button
                type="button"
                className="modal-confirm"
                disabled={busy || holdings.length === 0}
                onClick={() => {
                  onConfirm(holdings)
                  handleClose()
                }}
              >
                {holdings.length}개 종목 추가
              </button>
            </footer>
          </div>
        )}
      </div>

      {fullImageIndex !== null && previews[fullImageIndex] && (
        <div className="lightbox" onClick={() => setFullImageIndex(null)}>
          <img src={previews[fullImageIndex]} alt="캡처 원본" />
        </div>
      )}
    </dialog>
  )
}
