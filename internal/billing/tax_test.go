package billing

import "testing"

func TestTaxRulesApplyAndFind(t *testing.T) {
	rule, ok := FindTax("cn")
	if !ok || rule.Validate() != nil {
		t.Fatal("CN rule missing")
	}
	if rule.Apply(100) != 106 {
		t.Fatalf("tax=%d", rule.Apply(100))
	}
	exempt, ok := FindTax("PA")
	if !ok || exempt.Apply(100) != 100 {
		t.Fatal("exempt tax wrong")
	}
	if _, ok := FindTax("XX"); ok {
		t.Fatal("unknown tax found")
	}
}
