package calculator

import "testing"

func TestEvaluateArithmetic(t *testing.T) {
	cases := map[string]float64{
		"1+2*3":       7,
		"(1 + 2) * 3": 9,
		"8 / 2 - 1":   3,
		"-2 + 5":      3,
	}

	for expr, want := range cases {
		t.Run(expr, func(t *testing.T) {
			got, err := Evaluate(expr)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("Evaluate(%q) = %v, want %v", expr, got, want)
			}
		})
	}
}

func TestEvaluateRejectsInvalidExpressions(t *testing.T) {
	cases := []string{"", "1+", "1/0", "abc", "(1+2"}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			if _, err := Evaluate(expr); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestQueryUsesEqualsPrefix(t *testing.T) {
	result, ok := Query("= 21 * 2")
	if !ok {
		t.Fatal("expected calculator result")
	}
	if result.Title != "42" {
		t.Fatalf("Title = %q", result.Title)
	}
	if result.Action.Text != "42" {
		t.Fatalf("Action.Text = %q", result.Action.Text)
	}
}

func TestQueryIgnoresPlainWords(t *testing.T) {
	if _, ok := Query("firefox"); ok {
		t.Fatal("expected no calculator result")
	}
}
