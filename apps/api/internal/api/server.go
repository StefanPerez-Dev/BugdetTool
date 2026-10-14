package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/categorize"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/config"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/domain"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/money"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/ollama"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/store"
)

type Server struct {
	cfg    config.Config
	store  *store.Store
	ollama *ollama.Client
	mux    *http.ServeMux
}

func New(cfg config.Config, st *store.Store, ol *ollama.Client) *Server {
	s := &Server{cfg: cfg, store: st, ollama: ol, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.withCORS(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/accounts", s.handleListAccounts)
	s.mux.HandleFunc("GET /api/categories", s.handleListCategories)
	s.mux.HandleFunc("GET /api/budgets", s.handleListBudgets)
	s.mux.HandleFunc("PUT /api/budgets/{categoryId}", s.handleUpdateBudget)
	s.mux.HandleFunc("GET /api/transactions", s.handleListTransactions)
	s.mux.HandleFunc("POST /api/transactions", s.handleCreateTransaction)
	s.mux.HandleFunc("PATCH /api/transactions/{id}", s.handleUpdateTransaction)
	s.mux.HandleFunc("DELETE /api/transactions/{id}", s.handleDeleteTransaction)
	s.mux.HandleFunc("POST /api/inbox/confirm-all", s.handleConfirmAll)
	s.mux.HandleFunc("POST /api/demo/import", s.handleDemoImport)
	s.mux.HandleFunc("POST /api/imports", s.handleImport)
	s.mux.HandleFunc("DELETE /api/data", s.handleClearData)
	s.mux.HandleFunc("GET /api/meta", s.handleMeta)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.cfg.CORSOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	h := domain.Health{Status: "ok", Database: "ok", Ollama: "down", Model: s.cfg.OllamaModel}
	{
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		if err := s.store.Ping(ctx); err != nil {
			h.Status = "degraded"
			h.Database = "down"
		}
		cancel()
	}
	{
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		if err := s.ollama.Ping(ctx); err == nil {
			h.Ollama = "ok"
		} else if h.Status == "ok" {
			h.Status = "degraded"
		}
		cancel()
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	demo, err := s.store.DemoEnabled(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"demo": demo})
}

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListAccounts(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListCategories(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListBudgets(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleUpdateBudget(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("categoryId"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var in domain.UpdateBudgetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.store.UpdateBudget(r.Context(), id, money.ToCents(in.Amount)); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.ListTxFilter{Query: q.Get("q")}
	if ym := q.Get("month"); ym != "" {
		f.YearMonth = &ym
	}
	if rev := q.Get("reviewed"); rev != "" {
		v := rev == "true" || rev == "1"
		f.Reviewed = &v
	}
	items, err := s.store.ListTransactions(r.Context(), f)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []domain.Transaction{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) resolveCategory(ctx context.Context, description string, amountCents int, categoryID *int64) (int64, string, error) {
	if categoryID != nil {
		return *categoryID, "", nil
	}
	name := categorize.Fallback(description, amountCents)
	if _, ok := categorize.Match(description); !ok {
		cats, err := s.store.ListCategories(ctx)
		if err == nil {
			names := make([]string, 0, len(cats))
			for _, c := range cats {
				names = append(names, c.Name)
			}
			if llmName, err := s.ollama.Categorize(ctx, description, amountCents, names); err == nil {
				name = llmName
			}
		}
	}
	cat, err := s.store.CategoryByName(ctx, name)
	if err != nil {
		return 0, "", err
	}
	if cat == nil {
		cat, err = s.store.CategoryByName(ctx, "Overig")
		if err != nil || cat == nil {
			return 0, "", errors.New("category not found")
		}
	}
	return cat.ID, name, nil
}

func (s *Server) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if in.Date == "" || strings.TrimSpace(in.Description) == "" || in.AccountID == 0 {
		writeErr(w, http.StatusBadRequest, "date, description and accountId are required")
		return
	}
	amountCents := money.ToCents(in.Amount)
	catID, _, err := s.resolveCategory(r.Context(), in.Description, amountCents, in.CategoryID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	reviewed := true
	if in.Reviewed != nil {
		reviewed = *in.Reviewed
	}
	tx, err := s.store.CreateTransaction(r.Context(), in, amountCents, catID, reviewed)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tx)
}

func (s *Server) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in domain.UpdateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	tx, err := s.store.UpdateTransaction(r.Context(), id, in)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tx == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (s *Server) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteTransaction(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleConfirmAll(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.ConfirmAllInbox(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"confirmed": n})
}

func (s *Server) handleClearData(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ClearAll(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

var demoImportRows = []struct {
	date, desc string
	amount     float64
}{
	{"2026-10-10", "AH to go Amsterdam Zuid", -8.45},
	{"2026-10-09", "Uber Eats", -27.8},
	{"2026-10-09", "Apple.com/bill", -2.99},
	{"2026-10-08", "Zalando SE", -89.95},
	{"2026-10-07", "NS Reizen Utrecht", -14.2},
	{"2026-10-06", "PAYPAL *MARKTPLAATS", -35},
	{"2026-10-06", "Netflix", -13.99},
}

func (s *Server) importRows(ctx context.Context, accountID int64, rows []struct {
	date, desc string
	amount     float64
}, source string) (domain.ImportResult, error) {
	var created []domain.Transaction
	for _, row := range rows {
		exists, err := s.store.ExistsSimilar(ctx, row.date, money.ToCents(row.amount), row.desc)
		if err != nil {
			return domain.ImportResult{}, err
		}
		if exists {
			continue
		}
		amountCents := money.ToCents(row.amount)
		catID, _, err := s.resolveCategory(ctx, row.desc, amountCents, nil)
		if err != nil {
			return domain.ImportResult{}, err
		}
		var flag *string
		dup, _ := s.store.ExistsSameDateAmount(ctx, row.date, amountCents)
		if dup {
			v := "Mogelijk dubbel"
			flag = &v
		} else if _, ok := categorize.Match(row.desc); !ok {
			v := "Onbekend"
			flag = &v
		}
		reviewed := false
		tx, err := s.store.CreateTransaction(ctx, domain.CreateTransactionInput{
			Date: row.date, Description: row.desc, Amount: row.amount,
			AccountID: accountID, Recurring: false, Flag: flag, Reviewed: &reviewed,
		}, amountCents, catID, false)
		if err != nil {
			return domain.ImportResult{}, err
		}
		created = append(created, *tx)
	}
	msg := "Geen nieuwe transacties gevonden"
	if len(created) > 0 {
		msg = strconv.Itoa(len(created)) + " nieuwe transacties in je inbox"
	}
	_ = source
	return domain.ImportResult{Imported: len(created), Message: msg, Items: created}, nil
}

func (s *Server) handleDemoImport(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.store.ListAccounts(r.Context())
	if err != nil || len(accounts) < 2 {
		writeErr(w, http.StatusInternalServerError, "accounts missing")
		return
	}
	res, err := s.importRows(r.Context(), accounts[1].ID, demoImportRows, "demo")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "expected multipart file upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	accounts, err := s.store.ListAccounts(r.Context())
	if err != nil || len(accounts) == 0 {
		writeErr(w, http.StatusInternalServerError, "accounts missing")
		return
	}
	accountID := accounts[0].ID
	if idStr := r.FormValue("accountId"); idStr != "" {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			accountID = id
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	var extracted []ollama.ExtractedTx
	ct := header.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "image/") || strings.HasSuffix(strings.ToLower(header.Filename), ".png") ||
		strings.HasSuffix(strings.ToLower(header.Filename), ".jpg") || strings.HasSuffix(strings.ToLower(header.Filename), ".jpeg") {
		b64 := base64.StdEncoding.EncodeToString(raw)
		extracted, err = s.ollama.ExtractFromImage(ctx, b64)
		if err != nil {
			// Fall back to demo rows so local UX still works without a vision model.
			res, ierr := s.importRows(ctx, accountID, demoImportRows, header.Filename)
			if ierr != nil {
				writeErr(w, http.StatusInternalServerError, err.Error()+"; fallback failed: "+ierr.Error())
				return
			}
			res.Message = res.Message + " (vision model unavailable — demo extract gebruikt)"
			writeJSON(w, http.StatusOK, res)
			return
		}
	} else {
		// PDF / unknown: try vision if possible, else demo fallback
		res, ierr := s.importRows(ctx, accountID, demoImportRows, header.Filename)
		if ierr != nil {
			writeErr(w, http.StatusInternalServerError, ierr.Error())
			return
		}
		res.Message = res.Message + " (PDF parsing via Ollama vision volgt later — demo extract gebruikt)"
		writeJSON(w, http.StatusOK, res)
		return
	}

	rows := make([]struct {
		date, desc string
		amount     float64
	}, 0, len(extracted))
	for _, e := range extracted {
		if e.Date == "" || e.Description == "" {
			continue
		}
		rows = append(rows, struct {
			date, desc string
			amount     float64
		}{e.Date, e.Description, e.Amount})
	}
	res, err := s.importRows(ctx, accountID, rows, header.Filename)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
