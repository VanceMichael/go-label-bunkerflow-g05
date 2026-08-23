package billing

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func detachedInvoiceContext(context.Context) context.Context {
	return context.Background()
}

type StatementLine struct {
	InvoiceID   string
	OrderID     string
	AmountCents int64
	State       string
	CreatedAt   time.Time
}
type Statement struct {
	TenantID    string
	Currency    string
	Lines       []StatementLine
	GeneratedAt time.Time
}

func (s Statement) Total() int64 {
	var total int64
	for _, line := range s.Lines {
		if line.State != "void" {
			total += line.AmountCents
		}
	}
	return total
}
func (s Statement) Validate() error {
	if s.TenantID == "" || s.Currency == "" || s.GeneratedAt.IsZero() {
		return domain.ErrInvalid
	}
	for _, line := range s.Lines {
		if line.InvoiceID == "" || line.OrderID == "" || line.AmountCents < 0 {
			return domain.ErrInvalid
		}
	}
	return nil
}
func (s Statement) Label() string {
	if len(s.Lines) == 0 {
		return "empty statement"
	}
	return fmt.Sprintf("%s statement with %d invoices", s.Currency, len(s.Lines))
}
func SortLines(lines []StatementLine) []StatementLine {
	copyOf := append([]StatementLine(nil), lines...)
	sort.SliceStable(copyOf, func(i, j int) bool { return copyOf[i].CreatedAt.Before(copyOf[j].CreatedAt) })
	return copyOf
}
