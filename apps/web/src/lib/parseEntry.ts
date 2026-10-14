import type { Category } from '../api/client'

const RULES: [RegExp, string][] = [
  [/salaris|loon/i, 'Salaris'],
  [/spaarrekening|sparen/i, 'Sparen'],
  [/tikkie|terugbetaling|toeslag|teruggave|refund/i, 'Overig inkomen'],
  [/thuisbezorgd|uber eats|deliveroo|restaurant|caf[eé]|starbucks|mcdonald|burger|pizza|lunch/i, 'Uit eten'],
  [/albert heijn|\bah\b|jumbo|lidl|aldi|picnic|dirk|\bplus\b|spar /i, 'Boodschappen'],
  [/huur|woonstichting|vattenfall|eneco|essent|waternet|gemeente/i, 'Wonen / huur'],
  [/netflix|spotify|disney|kpn|ziggo|odido|apple\.com|icloud|videoland|youtube/i, 'Abonnementen'],
  [/zilveren kruis|\bcz\b|vgz|menzis|verzekering|apotheek|tandarts|huisarts|centraal beheer/i, 'Zorg / verzekering'],
  [/booking\.com|klm|transavia|airbnb|easyjet|ryanair|hotel/i, 'Reizen'],
  [/\bns\b|ov-chip|shell|\bbp\b|esso|tango|uber|swapfiets|q-park|parkeren|gvb|ret /i, 'Vervoer'],
  [/zara|h&m|uniqlo|zalando|\bcos\b|weekday|nike|primark/i, 'Kleding'],
  [/path[eé]|bol\.com|basic-fit|sportschool|museum|boekhandel|steam|concert|ticketmaster/i, 'Vrije tijd'],
]

function match(desc: string) {
  for (const [re, c] of RULES) if (re.test(desc)) return c
  return null
}

export function parseEntry(text: string, categories: Category[]) {
  const t = text.trim()
  let desc = ''
  let sign = ''
  let num = ''
  let m = t.match(/^(.+?)\s+([+-]?)\s*€?\s*(\d+(?:[.,]\d{1,2})?)$/)
  if (m) {
    desc = m[1]
    sign = m[2]
    num = m[3]
  } else {
    m = t.match(/^([+-]?)\s*€?\s*(\d+(?:[.,]\d{1,2})?)\s+(.+)$/)
    if (!m) return null
    sign = m[1]
    num = m[2]
    desc = m[3]
  }
  desc = desc.trim()
  const v = +num.replace(',', '.')
  if (!desc || !(v > 0)) return null
  const incomeNames = new Set(categories.filter((c) => c.isIncome).map((c) => c.name))
  const matched = match(desc)
  const inc = sign === '+' || (sign !== '-' && matched !== null && incomeNames.has(matched))
  const amount = inc ? v : -v
  const category =
    matched ||
    (amount > 0 ? 'Overig inkomen' : 'Overig')
  return { desc, amount, category }
}
