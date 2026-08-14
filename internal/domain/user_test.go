package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewUserInputValidate(t *testing.T) {
	valid := NewUserInput{
		Name:     "Ada Lovelace",
		Email:    "ada@example.com",
		Password: "s3cret-pass",
	}

	tests := []struct {
		name      string
		mutate    func(*NewUserInput)
		wantField string
	}{
		{"valid", func(*NewUserInput) {}, ""},
		{"missing name", func(in *NewUserInput) { in.Name = "  " }, "name"},
		{"missing email", func(in *NewUserInput) { in.Email = "" }, "email"},
		{"invalid email", func(in *NewUserInput) { in.Email = "abc" }, "email"},
		{"email without tld", func(in *NewUserInput) { in.Email = "a@b" }, "email"},
		{"missing password", func(in *NewUserInput) { in.Password = "" }, "password"},
		{"short password", func(in *NewUserInput) { in.Password = "short" }, "password"},
		{"long password", func(in *NewUserInput) { in.Password = strings.Repeat("x", 73) }, "password"},
		{"name too long", func(in *NewUserInput) { in.Name = strings.Repeat("n", 101) }, "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.mutate(&in)

			err := in.Validate()
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			var verr ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			for _, fe := range verr {
				if fe.Field == tt.wantField {
					return
				}
			}
			t.Fatalf("expected error on field %q, got %v", tt.wantField, verr)
		})
	}
}

func TestValidationErrorError(t *testing.T) {
	verr := ValidationError{
		{Field: "name", Message: "is required"},
		{Field: "email", Message: "is required"},
	}
	msg := verr.Error()
	if !strings.Contains(msg, "name") || !strings.Contains(msg, "email") {
		t.Fatalf("unexpected message: %s", msg)
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"665f1c2d3e4f5a6b7c8d9e0f", true},    // valid ObjectID hex
		{"665F1C2D3E4F5A6B7C8D9E0F", true},    // uppercase accepted
		{"665f1c2d3e4f5a6b7c8d9e0", false},    // too short
		{"665f1c2d3e4f5a6b7c8d9e0faa", false}, // too long
		{"zzzf1c2d3e4f5a6b7c8d9e0f", false},   // non-hex
		{"", false},
	}
	for _, tt := range tests {
		if got := IsValidID(tt.id); got != tt.want {
			t.Errorf("IsValidID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}
