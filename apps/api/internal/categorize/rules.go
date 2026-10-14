package categorize

import "regexp"

type Rule struct {
	Pattern  *regexp.Regexp
	Category string
}

var rules = []Rule{
	{regexp.MustCompile(`(?i)salaris|loon`), "Salaris"},
	{regexp.MustCompile(`(?i)spaarrekening|sparen`), "Sparen"},
	{regexp.MustCompile(`(?i)tikkie|terugbetaling|toeslag|teruggave|refund`), "Overig inkomen"},
	{regexp.MustCompile(`(?i)thuisbezorgd|uber eats|deliveroo|restaurant|caf[eé]|starbucks|mcdonald|burger|pizza|lunch`), "Uit eten"},
	{regexp.MustCompile(`(?i)albert heijn|\bah\b|jumbo|lidl|aldi|picnic|dirk|\bplus\b|spar `), "Boodschappen"},
	{regexp.MustCompile(`(?i)huur|woonstichting|vattenfall|eneco|essent|waternet|gemeente`), "Wonen / huur"},
	{regexp.MustCompile(`(?i)netflix|spotify|disney|kpn|ziggo|odido|apple\.com|icloud|videoland|youtube`), "Abonnementen"},
	{regexp.MustCompile(`(?i)zilveren kruis|\bcz\b|vgz|menzis|verzekering|apotheek|tandarts|huisarts|centraal beheer`), "Zorg / verzekering"},
	{regexp.MustCompile(`(?i)booking\.com|klm|transavia|airbnb|easyjet|ryanair|hotel`), "Reizen"},
	{regexp.MustCompile(`(?i)\bns\b|ov-chip|shell|\bbp\b|esso|tango|uber|swapfiets|q-park|parkeren|gvb|ret `), "Vervoer"},
	{regexp.MustCompile(`(?i)zara|h&m|uniqlo|zalando|\bcos\b|weekday|nike|primark`), "Kleding"},
	{regexp.MustCompile(`(?i)path[eé]|bol\.com|basic-fit|sportschool|museum|boekhandel|steam|concert|ticketmaster`), "Vrije tijd"},
}

func Match(description string) (string, bool) {
	for _, rule := range rules {
		if rule.Pattern.MatchString(description) {
			return rule.Category, true
		}
	}
	return "", false
}

func Fallback(description string, amountCents int) string {
	if name, ok := Match(description); ok {
		return name
	}
	if amountCents > 0 {
		return "Overig inkomen"
	}
	return "Overig"
}
