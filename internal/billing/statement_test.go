package billing

import (
	"testing"
	"time"
)

func TestStatementTotalsNonVoidedLines(t *testing.T) {
	statement := Statement{TenantID: "tenant-zj", Currency: "USD", GeneratedAt: time.Now(), Lines: []StatementLine{{InvoiceID: "i1", OrderID: "o1", AmountCents: 100, State: "paid"}, {InvoiceID: "i2", OrderID: "o2", AmountCents: 50, State: "void"}}}
	if err := statement.Validate(); err != nil {
		t.Fatal(err)
	}
	if statement.Total() != 100 || statement.Label() == "empty statement" {
		t.Fatalf("statement=%+v", statement)
	}
}
