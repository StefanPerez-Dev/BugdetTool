package domain

import "time"

type Account struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type Category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Hue       *int   `json:"hue"`
	IsIncome  bool   `json:"isIncome"`
	IsSavings bool   `json:"isSavings"`
	Color     string `json:"color"`
}

type Budget struct {
	CategoryID   int64   `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	AmountCents  int     `json:"amountCents"`
	Amount       float64 `json:"amount"`
	Hue          *int    `json:"hue"`
	Color        string  `json:"color"`
}

type Transaction struct {
	ID           int64     `json:"id"`
	Date         string    `json:"date"`
	Description  string    `json:"description"`
	AmountCents  int       `json:"amountCents"`
	Amount       float64   `json:"amount"`
	AccountID    int64     `json:"accountId"`
	Account      string    `json:"account"`
	AccountShort string    `json:"accountShort"`
	CategoryID   int64     `json:"categoryId"`
	Category     string    `json:"category"`
	CategoryHue  *int      `json:"categoryHue"`
	CategoryColor string   `json:"categoryColor"`
	Reviewed     bool      `json:"reviewed"`
	Recurring    bool      `json:"recurring"`
	Flag         *string   `json:"flag,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateTransactionInput struct {
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	AccountID   int64   `json:"accountId"`
	CategoryID  *int64  `json:"categoryId"`
	Reviewed    *bool   `json:"reviewed"`
	Recurring   bool    `json:"recurring"`
	Flag        *string `json:"flag"`
}

type UpdateTransactionInput struct {
	Date        *string  `json:"date"`
	Description *string  `json:"description"`
	Amount      *float64 `json:"amount"`
	AccountID   *int64   `json:"accountId"`
	CategoryID  *int64   `json:"categoryId"`
	Reviewed    *bool    `json:"reviewed"`
	Recurring   *bool    `json:"recurring"`
	Flag        *string  `json:"flag"`
	ClearFlag   bool     `json:"clearFlag"`
}

type UpdateBudgetInput struct {
	Amount float64 `json:"amount"`
}

type ImportResult struct {
	Imported int           `json:"imported"`
	Message  string        `json:"message"`
	Items    []Transaction `json:"items"`
}

type Health struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Ollama   string `json:"ollama"`
	Model    string `json:"model"`
}

type MonthSummary struct {
	YearMonth   string  `json:"yearMonth"`
	Income      float64 `json:"income"`
	Expenses    float64 `json:"expenses"`
	Saved       float64 `json:"saved"`
	Net         float64 `json:"net"`
	LeftInEnvelopes float64 `json:"leftInEnvelopes"`
	RecurringCount  int     `json:"recurringCount"`
}
