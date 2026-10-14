import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  api,
  type Account,
  type Budget,
  type Category,
  type Health,
  type Transaction,
} from './api/client'
import { dDay, dShort, eur, monthLabel, neg, signed, todayIso, ymKey } from './lib/format'
import { parseEntry } from './lib/parseEntry'

type Page = 'inbox' | 'ledger' | 'budgets'

export default function App() {
  const [page, setPage] = useState<Page>('inbox')
  const [y, setY] = useState(2026)
  const [m, setM] = useState(9)
  const [q, setQ] = useState('')
  const [accounts, setAccounts] = useState<Account[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [budgets, setBudgets] = useState<Budget[]>([])
  const [tx, setTx] = useState<Transaction[]>([])
  const [demo, setDemo] = useState(true)
  const [health, setHealth] = useState<Health | null>(null)
  const [sel, setSel] = useState<number | null>(null)
  const [done, setDone] = useState(0)
  const [importing, setImporting] = useState(false)
  const [busy, setBusy] = useState(0)
  const [fileName, setFileName] = useState('')
  const [dragging, setDragging] = useState(false)
  const [toast, setToast] = useState<string | null>(null)
  const [entry, setEntry] = useState({ date: todayIso(), text: '', accountId: 0 })
  const [loading, setLoading] = useState(true)
  const fileRef = useRef<HTMLInputElement>(null)
  const dragDepth = useRef(0)
  const toastTimer = useRef<number | null>(null)

  const showToast = useCallback((t: string) => {
    if (toastTimer.current) window.clearTimeout(toastTimer.current)
    setToast(t)
    toastTimer.current = window.setTimeout(() => setToast(null), 2600)
  }, [])

  const reload = useCallback(async () => {
    const [acc, cats, buds, allTx, meta, h] = await Promise.all([
      api.accounts(),
      api.categories(),
      api.budgets(),
      api.transactions(),
      api.meta(),
      api.health().catch(() => null),
    ])
    setAccounts(acc)
    setCategories(cats)
    setBudgets(buds)
    setTx(allTx)
    setDemo(meta.demo)
    setHealth(h)
    setEntry((e) => ({ ...e, accountId: e.accountId || acc[0]?.id || 0 }))
  }, [])

  useEffect(() => {
    reload()
      .catch((e) => showToast(String(e.message || e)))
      .finally(() => setLoading(false))
  }, [reload, showToast])

  const inbox = useMemo(
    () => tx.filter((t) => !t.reviewed).sort((a, b) => b.date.localeCompare(a.date) || a.id - b.id),
    [tx],
  )
  const reviewed = useMemo(() => tx.filter((t) => t.reviewed), [tx])
  const key = ymKey(y, m)
  const monthTx = useMemo(
    () => reviewed.filter((t) => t.date.slice(0, 7) === key),
    [reviewed, key],
  )

  const curSel = inbox.some((t) => t.id === sel) ? sel : inbox[0]?.id ?? null

  const expenseCats = categories.filter((c) => !c.isIncome && !c.isSavings)
  const envelopes = budgets
    .map((b) => {
      const spent = -monthTx
        .filter((t) => t.categoryId === b.categoryId && t.amount < 0)
        .reduce((a, t) => a + t.amount, 0)
      const left = b.amount - spent
      const ratio = b.amount ? spent / b.amount : 0
      const over = left < -0.005
      const warn = !over && ratio > 0.8
      return {
        ...b,
        spent,
        left,
        ratio,
        over,
        warn,
        barW: `${Math.min(100, ratio * 100)}%`,
        barColor: over ? 'rgb(251,109,58)' : warn ? 'oklch(0.78 0.15 75)' : 'rgb(36,0,255)',
        leftColor: over ? 'rgb(196,72,28)' : 'rgb(26,28,30)',
        leftShort: over
          ? `${eur(-left).replace(/,\d\d$/, '')} te veel`
          : `${eur(left).replace(/,\d\d$/, '')} over`,
        leftLong: over ? `${eur(-left)} te veel` : `Nog ${eur(left)}`,
      }
    })
    .filter((e) => e.amount > 0)

  const leftTotal = envelopes.reduce((a, e) => a + Math.max(0, e.left), 0)
  const income = monthTx.filter((t) => t.amount > 0).reduce((a, t) => a + t.amount, 0)
  const saved = -monthTx.filter((t) => categories.find((c) => c.id === t.categoryId)?.isSavings).reduce((a, t) => a + t.amount, 0)
  const out = -monthTx.filter((t) => t.amount < 0 && !categories.find((c) => c.id === t.categoryId)?.isSavings).reduce((a, t) => a + t.amount, 0)
  const net = income - out - saved
  const tight = [...envelopes].filter((e) => e.ratio > 0.8).sort((a, b) => b.ratio - a.ratio).slice(0, 3)
  const recCount = new Set(reviewed.filter((t) => t.recurring).map((t) => t.description.toLowerCase())).size

  const pe = parseEntry(entry.text, categories)
  const totalProgress = inbox.length + done

  const shiftMonth = (d: number) => {
    const x = new Date(y, m + d, 1)
    setY(x.getFullYear())
    setM(x.getMonth())
  }

  const confirmTx = async (id: number) => {
    const list = inbox
    const i = list.findIndex((t) => t.id === id)
    const rest = list.filter((t) => t.id !== id)
    const next = rest[Math.min(i, rest.length - 1)]?.id ?? null
    await api.updateTransaction(id, { reviewed: true, clearFlag: true })
    setDone((d) => d + 1)
    setSel(next)
    await reload()
  }

  const discard = async (id: number) => {
    const list = inbox
    const i = list.findIndex((t) => t.id === id)
    const rest = list.filter((t) => t.id !== id)
    const next = rest[Math.min(i, rest.length - 1)]?.id ?? null
    await api.deleteTransaction(id)
    setDone((d) => d + 1)
    setSel(next)
    await reload()
  }

  const startImport = async (file?: File) => {
    setPage('inbox')
    setImporting(true)
    setBusy(0)
    setFileName(file?.name || 'Voorbeeldimport')
    const timers = [350, 900, 1500].map((t, i) => window.setTimeout(() => setBusy(i + 1), t))
    try {
      const res = file ? await api.importFile(file) : await api.demoImport()
      await reload()
      setDone(0)
      if (res.items[0]) setSel(res.items[0].id)
      showToast(res.message)
    } catch (e) {
      showToast(String((e as Error).message || e))
    } finally {
      timers.forEach(clearTimeout)
      setImporting(false)
    }
  }

  useEffect(() => {
    const onPaste = (e: ClipboardEvent) => {
      const it = [...(e.clipboardData?.items || [])].find((i) => i.kind === 'file' && /image|pdf/.test(i.type))
      if (it) {
        e.preventDefault()
        const f = it.getAsFile()
        if (f) void startImport(f)
      }
    }
    const onEnter = (e: DragEvent) => {
      if (![...(e.dataTransfer?.types || [])].includes('Files')) return
      dragDepth.current++
      setDragging(true)
    }
    const onLeave = () => {
      dragDepth.current = Math.max(0, dragDepth.current - 1)
      if (!dragDepth.current) setDragging(false)
    }
    const onOver = (e: DragEvent) => e.preventDefault()
    const onDrop = (e: DragEvent) => {
      e.preventDefault()
      dragDepth.current = 0
      setDragging(false)
      const f = e.dataTransfer?.files?.[0]
      if (f) void startImport(f)
    }
    const onKey = (e: KeyboardEvent) => {
      if (page !== 'inbox' || /INPUT|SELECT|TEXTAREA/.test((e.target as HTMLElement).tagName)) return
      if (!inbox.length) return
      const i = inbox.findIndex((t) => t.id === curSel)
      if (e.key === 'Enter') {
        e.preventDefault()
        void confirmTx(inbox[i].id)
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSel(inbox[Math.min(inbox.length - 1, i + 1)].id)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSel(inbox[Math.max(0, i - 1)].id)
      }
    }
    window.addEventListener('paste', onPaste)
    window.addEventListener('dragenter', onEnter)
    window.addEventListener('dragleave', onLeave)
    window.addEventListener('dragover', onOver)
    window.addEventListener('drop', onDrop)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('paste', onPaste)
      window.removeEventListener('dragenter', onEnter)
      window.removeEventListener('dragleave', onLeave)
      window.removeEventListener('dragover', onOver)
      window.removeEventListener('drop', onDrop)
      window.removeEventListener('keydown', onKey)
    }
  })

  const ledgerRows = useMemo(() => {
    let asc = [...monthTx].sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)
    let run = 0
    const runMap: Record<number, number> = {}
    asc.forEach((t) => {
      run += t.amount
      runMap[t.id] = run
    })
    let rows = [...monthTx].sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id)
    const qq = q.toLowerCase()
    if (qq) rows = rows.filter((t) => t.description.toLowerCase().includes(qq) || t.category.toLowerCase().includes(qq))
    return rows.map((t) => ({ ...t, running: runMap[t.id] }))
  }, [monthTx, q])

  const addEntry = async () => {
    if (!pe || !entry.accountId) return
    const cat = categories.find((c) => c.name === pe.category)
    await api.createTransaction({
      date: entry.date || todayIso(),
      description: pe.desc,
      amount: pe.amount,
      accountId: entry.accountId,
      categoryId: cat?.id,
      reviewed: true,
    })
    const [yy, mm] = (entry.date || todayIso()).split('-').map(Number)
    setY(yy)
    setM(mm - 1)
    setEntry((e) => ({ ...e, text: '' }))
    await reload()
    showToast('Toegevoegd')
  }

  const exportCsv = () => {
    const rows = [
      ['Datum', 'Omschrijving', 'Rekening', 'Categorie', 'Bedrag', 'Vast'],
      ...[...monthTx]
        .sort((a, b) => a.date.localeCompare(b.date))
        .map((t) => [t.date, t.description, t.account, t.category, t.amount.toFixed(2).replace('.', ','), t.recurring ? 'ja' : '']),
    ]
    const csv = '\ufeff' + rows.map((x) => x.map((v) => `"${String(v).replace(/"/g, '""')}"`).join(';')).join('\n')
    const a = document.createElement('a')
    a.href = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }))
    a.download = `kasboek-${key}.csv`
    a.click()
    showToast(`${monthTx.length} transacties geëxporteerd`)
  }

  if (loading) {
    return <div className="main"><div className="muted">Kasboek laden…</div></div>
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">K</div>
          <div className="brand-name">Kasboek</div>
        </div>
        <nav className="nav">
          {(
            [
              ['inbox', 'Inbox', inbox.length ? String(inbox.length) : ''],
              ['ledger', 'Kasboek', ''],
              ['budgets', 'Enveloppen', ''],
            ] as const
          ).map(([k, label, badge]) => (
            <button key={k} className={`nav-btn ${page === k ? 'active' : ''}`} onClick={() => setPage(k)}>
              <span>{label}</span>
              {badge ? <span className="nav-badge">{badge}</span> : null}
            </button>
          ))}
        </nav>
        <button className="import-btn" onClick={() => fileRef.current?.click()}>Importeren</button>
        <div className="sidebar-foot">
          <div className="muted">Rekeningen</div>
          {accounts.map((a) => (
            <div key={a.id} style={{ display: 'flex', justifyContent: 'space-between', gap: 8, fontSize: 13 }}>
              <span>{a.name}</span>
              <span className="muted">{reviewed.filter((t) => t.accountId === a.id).length}</span>
            </div>
          ))}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8 }}>
            <span className={`status-dot ${health?.database === 'ok' ? 'status-ok' : 'status-bad'}`} />
            <span className="muted">DB {health?.database ?? '?'}</span>
            <span className={`status-dot ${health?.ollama === 'ok' ? 'status-ok' : 'status-bad'}`} />
            <span className="muted">Ollama {health?.ollama ?? '?'}</span>
          </div>
          <button
            className="linkish"
            onClick={async () => {
              if (!window.confirm(demo ? 'Voorbeelddata wissen en leeg beginnen?' : 'Alle transacties wissen?')) return
              await api.clearData()
              await reload()
              showToast('Gewist, je begint met een lege lijst')
            }}
          >
            {demo ? 'Voorbeelddata wissen' : 'Alles wissen'}
          </button>
        </div>
      </aside>

      <main className="main">
        {page === 'inbox' && (
          <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1fr) 290px', gap: 24, alignItems: 'start' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 18, minWidth: 0 }}>
              <div>
                <div className="muted" style={{ fontSize: 13 }}>
                  {inbox.length ? 'Klopt de categorie? Bevestig, pas aan of verwijder.' : 'Je kasboek is bij.'}
                </div>
                <h1 className="h1">
                  {inbox.length
                    ? `${inbox.length} ${inbox.length === 1 ? 'transactie' : 'transacties'} om te controleren`
                    : 'Alles gecontroleerd'}
                </h1>
              </div>

              {importing && (
                <div className="card" style={{ padding: '18px 20px', display: 'flex', flexDirection: 'column', gap: 10 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, fontSize: 13 }}>
                    <span style={{ fontWeight: 600 }}>
                      {['Bestand lezen…', 'Transacties herkennen…', 'Categorieën toewijzen…', 'Bijna klaar…'][busy]}
                    </span>
                    <span className="muted">{fileName}</span>
                  </div>
                  <div style={{ height: 6, borderRadius: 99, background: 'rgb(240,241,245)', overflow: 'hidden' }}>
                    <div style={{ height: '100%', borderRadius: 99, background: 'var(--brand)', width: `${((busy + 1) / 4) * 100}%`, transition: 'width .4s ease' }} />
                  </div>
                </div>
              )}

              {inbox.length > 0 && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  {inbox.map((t) => {
                    const open = t.id === curSel
                    return (
                      <div
                        key={t.id}
                        className="card"
                        style={{
                          padding: '16px 20px',
                          display: 'flex',
                          flexDirection: 'column',
                          gap: 14,
                          boxShadow: open
                            ? 'inset 0 0 0 1.5px rgb(32,29,181), 0 4px 18px rgba(0,0,0,0.06)'
                            : undefined,
                        }}
                      >
                        <div
                          onClick={() => setSel(t.id)}
                          style={{ display: 'grid', gridTemplateColumns: '60px minmax(0,1fr) auto 116px', alignItems: 'center', gap: 16, cursor: 'pointer' }}
                        >
                          <div className="muted" style={{ fontSize: 12 }}>{dShort(t.date)}</div>
                          <div style={{ display: 'flex', flexDirection: 'column', gap: 3, minWidth: 0 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
                              <span style={{ fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{t.description}</span>
                              {t.flag ? (
                                <span style={{ flexShrink: 0, fontSize: 11, fontWeight: 600, padding: '3px 8px', borderRadius: 999, background: 'rgb(255,238,230)', color: 'rgb(176,62,20)' }}>
                                  {t.flag}
                                </span>
                              ) : null}
                            </div>
                            <div className="muted" style={{ fontSize: 12 }}>{t.account}</div>
                          </div>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 6, background: 'rgb(244,245,247)', borderRadius: 999, padding: '6px 12px', fontSize: 12, fontWeight: 500, whiteSpace: 'nowrap' }}>
                            <span style={{ width: 8, height: 8, borderRadius: 99, background: t.categoryColor }} />
                            {t.category}
                          </div>
                          <div style={{ textAlign: 'right', fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontVariantNumeric: 'tabular-nums', color: t.amount > 0 ? 'var(--green)' : 'var(--ink)' }}>
                            {signed(t.amount)}
                          </div>
                        </div>
                        {open && (
                          <div style={{ display: 'flex', flexDirection: 'column', gap: 14, borderTop: '1px solid rgb(240,241,245)', paddingTop: 14 }}>
                            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                              {categories
                                .filter((c) => (t.amount > 0 ? true : !c.isIncome))
                                .map((c) => (
                                  <button
                                    key={c.id}
                                    className={`pill ${c.id === t.categoryId ? 'active' : ''}`}
                                    onClick={async () => {
                                      await api.updateTransaction(t.id, {
                                        categoryId: c.id,
                                        ...(t.flag === 'Onbekend' ? { clearFlag: true } : {}),
                                      })
                                      await reload()
                                    }}
                                  >
                                    {c.name}
                                  </button>
                                ))}
                            </div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                              <button
                                style={{ border: 'none', background: 'none', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, padding: 0 }}
                                onClick={async () => {
                                  await api.updateTransaction(t.id, { recurring: !t.recurring })
                                  await reload()
                                }}
                              >
                                <span style={{ width: 36, height: 20, borderRadius: 99, background: t.recurring ? 'var(--brand)' : 'rgb(217,217,217)', position: 'relative', display: 'inline-block' }}>
                                  <span style={{ position: 'absolute', top: 2, left: t.recurring ? 18 : 2, width: 16, height: 16, borderRadius: 99, background: '#fff', transition: 'left .15s' }} />
                                </span>
                                Vaste last (elke maand)
                              </button>
                              <span style={{ flex: 1 }} />
                              <button className="btn-ghost" onClick={() => void discard(t.id)}>Verwijderen</button>
                              <button className="btn-primary" onClick={() => void confirmTx(t.id)}>Klopt</button>
                            </div>
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}

              {inbox.length === 0 && !importing && (
                <>
                  <div className="dropzone" onClick={() => fileRef.current?.click()}>
                    <div style={{ width: 60, height: 60, borderRadius: 20, background: 'var(--brand)', color: '#fff', display: 'grid', placeItems: 'center', fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontSize: 28, boxShadow: '0 10px 20px rgba(43,57,185,0.25)' }}>+</div>
                    <div style={{ fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontSize: 20 }}>Sleep je volgende afschrift hierheen</div>
                    <div className="muted" style={{ maxWidth: 440, lineHeight: 1.6 }}>
                      Screenshot of PDF van ABN AMRO, ABN AMRO Creditcard of ING. Nieuwe transacties komen hier te staan zodat je ze kunt controleren.
                    </div>
                  </div>
                  <button className="linkish" style={{ alignSelf: 'center', color: 'var(--brand-link)', fontSize: 13, fontWeight: 500 }} onClick={() => void startImport()}>
                    Probeer een voorbeeldimport
                  </button>
                </>
              )}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 16, position: 'sticky', top: 32 }}>
              {inbox.length > 0 && (
                <div className="card" style={{ padding: 20, display: 'flex', flexDirection: 'column', gap: 12 }}>
                  <div className="muted" style={{ fontSize: 13 }}>Gecontroleerd</div>
                  <div style={{ fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontSize: 22 }}>{done} van {totalProgress}</div>
                  <div style={{ height: 8, borderRadius: 99, background: 'rgb(240,241,245)', overflow: 'hidden' }}>
                    <div style={{ height: '100%', width: `${totalProgress ? (done / totalProgress) * 100 : 0}%`, background: 'var(--brand)', transition: 'width .3s' }} />
                  </div>
                  <button
                    className="btn-ghost"
                    onClick={async () => {
                      await api.confirmAll()
                      setDone((d) => d + inbox.length)
                      setSel(null)
                      await reload()
                    }}
                  >
                    Alles in één keer goedkeuren
                  </button>
                  <div className="muted" style={{ fontSize: 12, lineHeight: 1.5 }}>Enter = klopt · ↑ ↓ = volgende</div>
                </div>
              )}
              <div className="card" style={{ padding: 20, display: 'flex', flexDirection: 'column', gap: 12 }}>
                <div className="muted" style={{ fontSize: 13 }}>{monthLabel(y, m).replace(/^./, (c) => c.toUpperCase())}</div>
                {[
                  ['Uitgegeven', eur(out), 'var(--ink)'],
                  ['Binnengekomen', eur(income), 'var(--green)'],
                  ['Nog in enveloppen', eur(leftTotal), 'var(--brand)'],
                  ['Vaste lasten', String(recCount), 'var(--ink)'],
                ].map(([label, value, color]) => (
                  <div key={label as string} style={{ display: 'flex', justifyContent: 'space-between', gap: 12, fontSize: 13 }}>
                    <span>{label}</span>
                    <span style={{ fontWeight: 600, fontVariantNumeric: 'tabular-nums', color: color as string }}>{value}</span>
                  </div>
                ))}
                <button className="linkish" style={{ color: 'var(--brand-link)', fontSize: 13, fontWeight: 500 }} onClick={() => setPage('ledger')}>
                  Naar kasboek
                </button>
              </div>
              {tight.length > 0 && (
                <div className="card" style={{ padding: 20, display: 'flex', flexDirection: 'column', gap: 12 }}>
                  <div className="muted" style={{ fontSize: 13 }}>Bijna op</div>
                  {tight.map((e) => (
                    <div key={e.categoryId} style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, fontSize: 13 }}>
                        <span style={{ fontWeight: 500 }}>{e.categoryName}</span>
                        <span style={{ color: e.leftColor, fontWeight: 600 }}>{e.leftShort}</span>
                      </div>
                      <div style={{ height: 5, borderRadius: 99, background: 'rgb(232,233,236)', overflow: 'hidden' }}>
                        <div style={{ height: '100%', width: e.barW, background: e.barColor }} />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {page === 'ledger' && (
          <>
            {envelopes.length > 0 && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
                  <span className="muted" style={{ fontSize: 13 }}>Nog te besteden in {monthLabel(y, m).split(' ')[0]}</span>
                  <span style={{ fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontSize: 17, fontVariantNumeric: 'tabular-nums' }}>{eur(leftTotal)}</span>
                  <span style={{ flex: 1 }} />
                  <button className="linkish" style={{ color: 'var(--brand-link)', fontSize: 13, fontWeight: 500 }} onClick={() => setPage('budgets')}>
                    Enveloppen aanpassen
                  </button>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill,minmax(150px,1fr))', gap: 10 }}>
                  {envelopes.map((e) => (
                    <div key={e.categoryId} className="card" style={{ borderRadius: 14, padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 8 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, fontSize: 12 }}>
                        <span style={{ fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{e.categoryName}</span>
                        <span style={{ color: e.leftColor, fontWeight: 600, whiteSpace: 'nowrap' }}>{e.leftShort}</span>
                      </div>
                      <div style={{ height: 5, borderRadius: 99, background: 'rgb(232,233,236)', overflow: 'hidden' }}>
                        <div style={{ height: '100%', width: e.barW, background: e.barColor }} />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="card" style={{ borderRadius: 24, display: 'flex', flexDirection: 'column' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap', padding: '22px 24px 16px' }}>
                <h1 className="h1" style={{ textTransform: 'capitalize' }}>{monthLabel(y, m)}</h1>
                <div style={{ display: 'flex', gap: 2 }}>
                  <button className="nav-btn" style={{ width: 34, height: 34, padding: 0, justifyContent: 'center' }} onClick={() => shiftMonth(-1)}>‹</button>
                  <button className="nav-btn" style={{ width: 34, height: 34, padding: 0, justifyContent: 'center' }} onClick={() => shiftMonth(1)}>›</button>
                </div>
                <span style={{ flex: 1 }} />
                <input
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  placeholder="Zoeken…"
                  style={{ width: 200, border: '1px solid rgb(225,225,225)', borderRadius: 999, padding: '9px 16px', fontSize: 13, outline: 'none' }}
                />
                <button className="btn-ghost" onClick={exportCsv}>Exporteren (CSV)</button>
              </div>

              <div style={{ margin: '0 24px 8px', display: 'flex', flexWrap: 'wrap', gap: '8px 12px', alignItems: 'center', padding: '10px 12px 10px 16px', borderRadius: 14, boxShadow: 'inset 0 0 0 1px rgb(161,197,255)', background: 'rgb(240,242,255)' }}>
                <input type="date" value={entry.date} onChange={(e) => setEntry({ ...entry, date: e.target.value })} style={{ flex: '0 0 128px', border: 'none', background: 'transparent', outline: 'none', fontSize: 13, color: 'var(--brand-link)' }} />
                <input
                  value={entry.text}
                  onChange={(e) => setEntry({ ...entry, text: e.target.value })}
                  onKeyDown={(e) => { if (e.key === 'Enter') void addEntry() }}
                  placeholder="Typ omschrijving en bedrag, bv. Markt kaas 12,50 (of +25 Tikkie)"
                  style={{ flex: '1 1 260px', border: 'none', background: 'transparent', outline: 'none', fontSize: 14, minWidth: 0 }}
                />
                <select value={entry.accountId} onChange={(e) => setEntry({ ...entry, accountId: +e.target.value })} style={{ border: 'none', background: 'transparent', fontSize: 13, color: 'rgb(78,78,78)' }}>
                  {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
                </select>
                <span className="muted" style={{ fontSize: 12, whiteSpace: 'nowrap', marginLeft: 'auto' }}>
                  {pe ? `${pe.category} · ${signed(pe.amount)}` : entry.text ? 'Bedrag ontbreekt' : ''}
                </span>
                <button className="btn-primary" style={{ padding: '8px 16px', fontSize: 12, opacity: pe ? 1 : 0.4 }} onClick={() => void addEntry()}>
                  Toevoegen
                </button>
              </div>

              <div style={{ overflowX: 'auto' }}>
                <div style={{ minWidth: 700 }}>
                  <div style={{ display: 'grid', gridTemplateColumns: '88px minmax(140px,1fr) minmax(0,150px) minmax(0,120px) 104px 100px 28px', gap: 14, padding: '12px 24px 8px', fontSize: 12, color: 'var(--muted)' }}>
                    <span>Datum</span><span>Omschrijving</span><span>Categorie</span><span>Rekening</span>
                    <span style={{ textAlign: 'right' }}>Bedrag</span><span style={{ textAlign: 'right' }}>Stand maand</span><span />
                  </div>
                  {ledgerRows.map((r, idx) => {
                    const first = idx === 0 || r.date !== ledgerRows[idx - 1].date
                    return (
                      <div
                        key={r.id}
                        style={{
                          display: 'grid',
                          gridTemplateColumns: '88px minmax(140px,1fr) minmax(0,150px) minmax(0,120px) 104px 100px 28px',
                          gap: 14,
                          alignItems: 'center',
                          padding: '9px 24px',
                          borderTop: first ? '1px solid rgb(236,237,241)' : 'none',
                        }}
                      >
                        <span style={{ fontSize: 13, color: 'rgb(78,78,78)' }}>{first ? dDay(r.date) : ''}</span>
                        <span style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
                          <span style={{ fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{r.description}</span>
                          {r.recurring ? <span style={{ flexShrink: 0, fontSize: 11, fontWeight: 600, padding: '2px 8px', borderRadius: 999, background: 'var(--brand-soft)', color: 'var(--brand-link)' }}>vast</span> : null}
                        </span>
                        <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <span style={{ width: 7, height: 7, borderRadius: 99, flexShrink: 0, background: r.categoryColor }} />
                          <select
                            value={r.categoryId}
                            onChange={async (e) => {
                              await api.updateTransaction(r.id, { categoryId: +e.target.value })
                              await reload()
                            }}
                            style={{ border: 'none', background: 'transparent', fontSize: 13, padding: '2px 0', width: '100%' }}
                          >
                            {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
                          </select>
                        </span>
                        <span className="muted" style={{ fontSize: 13, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{r.accountShort}</span>
                        <span style={{ textAlign: 'right', fontVariantNumeric: 'tabular-nums', fontWeight: 600, color: r.amount > 0 ? 'var(--green)' : 'var(--ink)' }}>{signed(r.amount)}</span>
                        <span className="muted" style={{ textAlign: 'right', fontVariantNumeric: 'tabular-nums', fontSize: 13 }}>{neg(r.running)}</span>
                        <button
                          title="Verwijderen"
                          style={{ border: 'none', background: 'none', color: 'rgb(175,175,175)', fontSize: 16, width: 28, height: 28, borderRadius: 8 }}
                          onClick={async () => {
                            await api.deleteTransaction(r.id)
                            await reload()
                            showToast('Transactie verwijderd')
                          }}
                        >
                          ×
                        </button>
                      </div>
                    )
                  })}
                  {!ledgerRows.length && (
                    <div style={{ padding: '28px 24px' }} className="muted">
                      {q ? 'Niets gevonden.' : 'Nog geen gecontroleerde transacties in deze maand.'}
                    </div>
                  )}
                </div>
              </div>

              <div style={{ margin: '8px 24px 24px', display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(130px,1fr))', gap: 16, paddingTop: 16, borderTop: '2px solid var(--ink)' }}>
                {[
                  ['Inkomsten', eur(income), 'var(--green)'],
                  ['Uitgaven', eur(out), 'var(--ink)'],
                  ['Gespaard', eur(saved), 'var(--brand)'],
                  [key === todayIso().slice(0, 7) ? 'Over, tot nu toe' : 'Over, afsluiting', neg(net), net < -0.005 ? 'var(--warn-ink)' : 'var(--ink)'],
                ].map(([label, value, color]) => (
                  <div key={label as string} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                    <span className="muted" style={{ fontSize: 12 }}>{label}</span>
                    <span style={{ fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontSize: 18, fontVariantNumeric: 'tabular-nums', color: color as string }}>{value}</span>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}

        {page === 'budgets' && (
          <>
            <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', gap: 16, flexWrap: 'wrap' }}>
              <div>
                <div className="muted" style={{ fontSize: 13 }}>Wat je per maand per categorie wilt uitgeven</div>
                <h1 className="h1">Enveloppen</h1>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 2, background: '#fff', border: '1px solid var(--line)', borderRadius: 999, padding: 4 }}>
                <button className="nav-btn" style={{ width: 32, height: 32, padding: 0 }} onClick={() => shiftMonth(-1)}>‹</button>
                <div style={{ minWidth: 130, textAlign: 'center', fontFamily: 'Poppins, sans-serif', fontWeight: 500, fontSize: 14, textTransform: 'capitalize' }}>{monthLabel(y, m)}</div>
                <button className="nav-btn" style={{ width: 32, height: 32, padding: 0 }} onClick={() => shiftMonth(1)}>›</button>
              </div>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill,minmax(260px,1fr))', gap: 16 }}>
              {expenseCats.map((c) => {
                const b = budgets.find((x) => x.categoryId === c.id)
                const spent = -monthTx.filter((t) => t.categoryId === c.id && t.amount < 0).reduce((a, t) => a + t.amount, 0)
                const budget = b?.amount ?? 0
                const left = budget - spent
                const ratio = budget ? spent / budget : 0
                const over = left < -0.005
                const warn = !over && ratio > 0.8
                return (
                  <div key={c.id} className="card" style={{ borderRadius: 24, padding: 22, display: 'flex', flexDirection: 'column', gap: 14 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontSize: 15 }}>
                        <span style={{ width: 10, height: 10, borderRadius: 3, background: c.color }} />
                        {c.name}
                      </div>
                      <span style={{ fontSize: 12, fontWeight: 600, padding: '4px 10px', borderRadius: 999, background: over ? 'rgb(255,236,230)' : 'rgb(244,245,248)', color: over ? 'var(--warn-ink)' : 'var(--ink)' }}>
                        {over ? `${eur(-left)} te veel` : `Nog ${eur(left)}`}
                      </span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
                      <span style={{ fontFamily: 'Poppins, sans-serif', fontWeight: 600, fontSize: 22, fontVariantNumeric: 'tabular-nums' }}>{eur(spent)}</span>
                      <span className="muted" style={{ fontSize: 13 }}>uitgegeven</span>
                    </div>
                    <div style={{ height: 8, borderRadius: 99, background: 'rgb(240,241,245)', overflow: 'hidden' }}>
                      <div style={{ height: '100%', borderRadius: 99, width: `${Math.min(100, ratio * 100)}%`, background: over ? 'rgb(251,109,58)' : warn ? 'oklch(0.78 0.15 75)' : 'var(--brand)' }} />
                    </div>
                    <label style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, fontSize: 13, color: 'var(--muted)' }}>
                      Per maand
                      <span style={{ display: 'flex', alignItems: 'center', gap: 4, border: '1px solid rgb(225,225,225)', borderRadius: 999, padding: '6px 12px' }}>
                        €
                        <input
                          type="number"
                          value={budget || ''}
                          onChange={async (e) => {
                            const v = e.target.value === '' ? 0 : +e.target.value
                            await api.updateBudget(c.id, v)
                            await reload()
                          }}
                          style={{ border: 'none', outline: 'none', width: 64, fontSize: 13, textAlign: 'right', color: 'var(--ink)' }}
                        />
                      </span>
                    </label>
                  </div>
                )
              })}
            </div>
          </>
        )}
      </main>

      <input
        ref={fileRef}
        type="file"
        accept="image/*,application/pdf"
        style={{ display: 'none' }}
        onChange={(e) => {
          const f = e.target.files?.[0]
          e.target.value = ''
          if (f) void startImport(f)
        }}
      />
      {dragging && <div className="drag-overlay">Laat los om naar de inbox te importeren</div>}
      {toast && <div className="toast">{toast}</div>}
    </div>
  )
}
