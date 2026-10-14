const NF = new Intl.NumberFormat('nl-NL', { style: 'currency', currency: 'EUR' })
const MONTHS = [
  'januari', 'februari', 'maart', 'april', 'mei', 'juni',
  'juli', 'augustus', 'september', 'oktober', 'november', 'december',
]
const MS = ['jan', 'feb', 'mrt', 'apr', 'mei', 'jun', 'jul', 'aug', 'sep', 'okt', 'nov', 'dec']
const WD = ['zo', 'ma', 'di', 'wo', 'do', 'vr', 'za']

export function eur(n: number) {
  return NF.format(Math.abs(n) < 0.005 ? 0 : n)
}

export function signed(n: number) {
  return (n < 0 ? '− ' : '+ ') + NF.format(Math.abs(n))
}

export function neg(n: number) {
  return (n < -0.005 ? '− ' : '') + NF.format(Math.abs(n))
}

export function monthLabel(y: number, m: number) {
  return `${MONTHS[m]} ${y}`
}

export function ymKey(y: number, m: number) {
  return `${y}-${String(m + 1).padStart(2, '0')}`
}

export function dShort(s: string) {
  const [, m, d] = s.split('-')
  return `${+d} ${MS[+m - 1]}`
}

export function dDay(s: string) {
  const [y, m, d] = s.split('-').map(Number)
  return `${WD[new Date(y, m - 1, d).getDay()]} ${d} ${MS[m - 1]}`
}

export function todayIso() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export { MONTHS }
