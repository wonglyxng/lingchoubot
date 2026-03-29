package model

import (
	"database/sql/driver"
	"testing"
)

func TestJSON_Value(t *testing.T) {
	tests := []struct {
		name string
		j    JSON
		want string
	}{
		{"normal", JSON(`{"key":"val"}`), `{"key":"val"}`},
		{"empty", JSON(""), "{}"},
		{"null-like", JSON("{}"), "{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := tt.j.Value()
			if err != nil {
				t.Fatal(err)
			}
			b, ok := v.(driver.Value)
			if !ok {
				t.Fatalf("Value() returned %T, want driver.Value", v)
			}
			got := string(b.([]byte))
			if got != tt.want {
				t.Errorf("Value() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJSON_Scan(t *testing.T) {
	tests := []struct {
		name string
		src  interface{}
		want string
	}{
		{"bytes", []byte(`{"a":1}`), `{"a":1}`},
		{"string", `{"b":2}`, `{"b":2}`},
		{"nil", nil, "{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var j JSON
			err := j.Scan(tt.src)
			if err != nil {
				t.Fatal(err)
			}
			got := string(j)
			if got != tt.want {
				t.Errorf("Scan() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJSON_ScanUnsupported(t *testing.T) {
	var j JSON
	err := j.Scan(123)
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}
