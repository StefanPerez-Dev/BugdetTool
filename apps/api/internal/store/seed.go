package store

import (
	"context"
	"fmt"
	"math"
)

func (s *Store) SeedIfEmpty(ctx context.Context) error {
	n, err := s.CountTransactions(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureReferenceData(ctx); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return s.seedDemoTransactions(ctx)
}

func (s *Store) ensureReferenceData(ctx context.Context) error {
	accounts := [][2]string{
		{"ABN AMRO Betaalrekening", "ABN AMRO"},
		{"ABN AMRO Creditcard", "ABN Creditcard"},
		{"ING Betaalrekening", "ING"},
	}
	for _, a := range accounts {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO accounts(name, short_name) VALUES ($1, $2)
			ON CONFLICT (name) DO NOTHING`, a[0], a[1])
		if err != nil {
			return err
		}
	}

	type cat struct {
		name              string
		hue               *int
		income, savings   bool
		defaultBudgetEuro float64
	}
	h := func(v int) *int { return &v }
	cats := []cat{
		{"Boodschappen", h(145), false, false, 450},
		{"Wonen / huur", h(255), false, false, 1300},
		{"Vervoer", h(215), false, false, 120},
		{"Uit eten", h(40), false, false, 150},
		{"Abonnementen", h(300), false, false, 80},
		{"Zorg / verzekering", h(185), false, false, 160},
		{"Kleding", h(345), false, false, 100},
		{"Vrije tijd", h(85), false, false, 120},
		{"Reizen", h(230), false, false, 200},
		{"Sparen", h(275), false, true, 0},
		{"Salaris", h(150), true, false, 0},
		{"Overig inkomen", h(120), true, false, 0},
		{"Overig", nil, false, false, 100},
	}
	for _, c := range cats {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO categories(name, hue, is_income, is_savings)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO NOTHING`, c.name, c.hue, c.income, c.savings)
		if err != nil {
			return err
		}
		if c.defaultBudgetEuro > 0 {
			_, err = s.pool.Exec(ctx, `
				INSERT INTO budgets(category_id, amount_cents)
				SELECT id, $2 FROM categories WHERE name = $1
				ON CONFLICT (category_id) DO NOTHING`, c.name, int(math.Round(c.defaultBudgetEuro*100)))
			if err != nil {
				return err
			}
		}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_meta(key, value) VALUES ('demo', 'true')
		ON CONFLICT (key) DO NOTHING`)
	return err
}

func (s *Store) seedDemoTransactions(ctx context.Context) error {
	accID := map[string]int64{}
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM accounts`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		accID[name] = id
	}
	rows.Close()

	catID := map[string]int64{}
	rows, err = s.pool.Query(ctx, `SELECT id, name FROM categories`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		catID[name] = id
	}
	rows.Close()

	type row struct {
		date, desc, account, category string
		amount                        float64
		reviewed, recurring           bool
		flag                          *string
	}
	flag := func(s string) *string { return &s }
	A, C, I := "ABN AMRO Betaalrekening", "ABN AMRO Creditcard", "ING Betaalrekening"
	var items []row

	add := func(y, m, d int, desc string, amount float64, acc, cat string, reviewed, rec bool, fl *string) {
		items = append(items, row{
			date: fmt.Sprintf("%04d-%02d-%02d", y, m, d), desc: desc, amount: amount,
			account: acc, category: cat, reviewed: reviewed, recurring: rec, flag: fl,
		})
	}

	for m := 5; m <= 9; m++ {
		add(2026, m, 1, "Huur Woonstichting De Key", -1150, A, "Wonen / huur", true, true, nil)
		add(2026, m, 2, "Zilveren Kruis zorgverzekering", -142.5, A, "Zorg / verzekering", true, true, nil)
		add(2026, m, 3, "Vattenfall energie", -89, I, "Wonen / huur", true, true, nil)
		add(2026, m, 4, "Basic-Fit abonnement", -29.99, I, "Vrije tijd", true, true, nil)
		add(2026, m, 5, "KPN internet", -35, I, "Abonnementen", true, true, nil)
		add(2026, m, 6, "Netflix", -13.99, C, "Abonnementen", true, true, nil)
		add(2026, m, 8, "Spotify", -11.99, C, "Abonnementen", true, true, nil)
		add(2026, m, 24, "Salaris Studio Noord BV", 3250, A, "Salaris", true, true, nil)
		add(2026, m, 25, "Overboeking spaarrekening", -300, A, "Sparen", true, true, nil)
		add(2026, m, 10, "Albert Heijn", -42.15, A, "Boodschappen", true, false, nil)
		add(2026, m, 12, "Jumbo", -28.4, I, "Boodschappen", true, false, nil)
		add(2026, m, 14, "Thuisbezorgd.nl", -24.5, C, "Uit eten", true, false, nil)
		add(2026, m, 16, "NS Reizen", -14.2, A, "Vervoer", true, false, nil)
		add(2026, m, 18, "Shell Amstelveen", -55, I, "Vervoer", true, false, nil)
	}
	add(2026, 6, 12, "Booking.com hotel Lissabon", -420, C, "Reizen", true, false, nil)
	add(2026, 6, 10, "Transavia", -268, C, "Reizen", true, false, nil)
	add(2026, 10, 11, "Albert Heijn 1403 Utrecht", -47.82, A, "Boodschappen", false, false, nil)
	add(2026, 10, 10, "Tikkie J. de Vries", -22.5, A, "Overig", false, false, flag("Onbekend"))
	add(2026, 10, 9, "Ziggo B.V.", -54.95, A, "Abonnementen", false, true, nil)

	for _, it := range items {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO transactions(booked_on, description, amount_cents, account_id, category_id, reviewed, recurring, flag)
			VALUES ($1::date, $2, $3, $4, $5, $6, $7, $8)`,
			it.date, it.desc, int(math.Round(it.amount*100)), accID[it.account], catID[it.category],
			it.reviewed, it.recurring, it.flag,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
