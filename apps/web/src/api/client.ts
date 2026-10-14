const BASE = import.meta.env.VITE_API_BASE_URL ?? ''

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      ...(init?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
      ...init?.headers,
    },
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

export type Account = { id: number; name: string; shortName: string }
export type Category = {
  id: number
  name: string
  hue: number | null
  isIncome: boolean
  isSavings: boolean
  color: string
}
export type Budget = {
  categoryId: number
  categoryName: string
  amountCents: number
  amount: number
  hue: number | null
  color: string
}
export type Transaction = {
  id: number
  date: string
  description: string
  amountCents: number
  amount: number
  accountId: number
  account: string
  accountShort: string
  categoryId: number
  category: string
  categoryHue: number | null
  categoryColor: string
  reviewed: boolean
  recurring: boolean
  flag?: string | null
}

export type ImportResult = {
  imported: number
  message: string
  items: Transaction[]
}

export type Health = {
  status: string
  database: string
  ollama: string
  model: string
}

export const api = {
  health: () => request<Health>('/api/health'),
  meta: () => request<{ demo: boolean }>('/api/meta'),
  accounts: () => request<Account[]>('/api/accounts'),
  categories: () => request<Category[]>('/api/categories'),
  budgets: () => request<Budget[]>('/api/budgets'),
  updateBudget: (categoryId: number, amount: number) =>
    request('/api/budgets/' + categoryId, {
      method: 'PUT',
      body: JSON.stringify({ amount }),
    }),
  transactions: (params: { month?: string; reviewed?: boolean; q?: string } = {}) => {
    const sp = new URLSearchParams()
    if (params.month) sp.set('month', params.month)
    if (params.reviewed !== undefined) sp.set('reviewed', String(params.reviewed))
    if (params.q) sp.set('q', params.q)
    const qs = sp.toString()
    return request<Transaction[]>('/api/transactions' + (qs ? `?${qs}` : ''))
  },
  createTransaction: (body: Record<string, unknown>) =>
    request<Transaction>('/api/transactions', { method: 'POST', body: JSON.stringify(body) }),
  updateTransaction: (id: number, body: Record<string, unknown>) =>
    request<Transaction>(`/api/transactions/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteTransaction: (id: number) =>
    request(`/api/transactions/${id}`, { method: 'DELETE' }),
  confirmAll: () => request<{ confirmed: number }>('/api/inbox/confirm-all', { method: 'POST' }),
  demoImport: () => request<ImportResult>('/api/demo/import', { method: 'POST' }),
  importFile: async (file: File, accountId?: number) => {
    const fd = new FormData()
    fd.append('file', file)
    if (accountId) fd.append('accountId', String(accountId))
    return request<ImportResult>('/api/imports', { method: 'POST', body: fd })
  },
  clearData: () => request('/api/data', { method: 'DELETE' }),
}
