package pipeline

import (
	"testing"
)

func TestParseCondition(t *testing.T) {
	tests := []struct {
		name          string
		expr          string
		wantErr       bool
		wantNamespace string
		wantKey       string
		wantValue     string
		wantUncond    bool
	}{
		{
			name:       "empty string is unconditional",
			expr:       "",
			wantUncond: true,
		},
		{
			name:          "outcome=success",
			expr:          "outcome=success",
			wantNamespace: "outcome",
			wantKey:       "success",
			wantValue:     "success",
		},
		{
			name:          "outcome=failure",
			expr:          "outcome=failure",
			wantNamespace: "outcome",
			wantKey:       "failure",
			wantValue:     "failure",
		},
		{
			name:          "context key-value with boolean",
			expr:          "context.tests_passed=true",
			wantNamespace: "context",
			wantKey:       "tests_passed",
			wantValue:     "true",
		},
		{
			name:          "context key-value with number",
			expr:          "context.count=5",
			wantNamespace: "context",
			wantKey:       "count",
			wantValue:     "5",
		},
		{
			name:    "outcome with invalid value",
			expr:    "outcome=invalid",
			wantErr: true,
		},
		{
			name:    "missing equals operator",
			expr:    "invalid",
			wantErr: true,
		},
		{
			name:    "empty left-hand side",
			expr:    "=value",
			wantErr: true,
		},
		{
			name:    "empty right-hand side",
			expr:    "key=",
			wantErr: true,
		},
		{
			name:    "unknown namespace",
			expr:    "unknown=value",
			wantErr: true,
		},
		{
			name:    "empty context key",
			expr:    "context.=value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCondition(tt.expr)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseCondition(%q) expected error, got nil", tt.expr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseCondition(%q) unexpected error: %v", tt.expr, err)
			}

			if got.IsUnconditional() != tt.wantUncond {
				t.Errorf("IsUnconditional() = %v, want %v", got.IsUnconditional(), tt.wantUncond)
			}

			if tt.wantUncond {
				return
			}

			if got.Namespace != tt.wantNamespace {
				t.Errorf("Namespace = %q, want %q", got.Namespace, tt.wantNamespace)
			}
			if got.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", got.Key, tt.wantKey)
			}
			if got.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantValue)
			}
			if got.Raw != tt.expr {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.expr)
			}
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name string
		expr ConditionExpr
		ctx  *StepContext
		want bool
	}{
		{
			name: "unconditional always matches",
			expr: ConditionExpr{},
			ctx: &StepContext{
				Outcome: "failure",
			},
			want: true,
		},
		{
			name: "outcome=success matches success",
			expr: ConditionExpr{Namespace: "outcome", Key: "success", Value: "success"},
			ctx: &StepContext{
				Outcome: "success",
			},
			want: true,
		},
		{
			name: "outcome=success does not match failure",
			expr: ConditionExpr{Namespace: "outcome", Key: "success", Value: "success"},
			ctx: &StepContext{
				Outcome: "failure",
			},
			want: false,
		},
		{
			name: "context key-value matches",
			expr: ConditionExpr{Namespace: "context", Key: "tests_passed", Value: "true"},
			ctx: &StepContext{
				Context: map[string]string{"tests_passed": "true"},
			},
			want: true,
		},
		{
			name: "context key-value does not match different value",
			expr: ConditionExpr{Namespace: "context", Key: "tests_passed", Value: "true"},
			ctx: &StepContext{
				Context: map[string]string{"tests_passed": "false"},
			},
			want: false,
		},
		{
			name: "context key-value does not match missing key",
			expr: ConditionExpr{Namespace: "context", Key: "tests_passed", Value: "true"},
			ctx: &StepContext{
				Context: map[string]string{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateCondition(tt.expr, tt.ctx)
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}
