package database

import (
	"strings"
	"testing"
)

func TestWithDisablePreparedStatements_AppendsWhenMissing(t *testing.T) {
	dsn := "postgres://user:pass@host/db"
	out := withDisablePreparedStatements(dsn)
	if !strings.Contains(out, "disable_prepared_statements=true") {
		t.Fatalf("expected disable_prepared_statements=true appended, got %s", out)
	}
	if !strings.Contains(strings.ToLower(out), "binary_parameters=yes") {
		t.Fatalf("expected binary_parameters=yes appended, got %s", out)
	}
}

func TestWithDisablePreparedStatements_NoDuplicateWhenPresent(t *testing.T) {
	dsn := "postgres://user:pass@host/db?disable_prepared_statements=true&binary_parameters=yes"
	out := withDisablePreparedStatements(dsn)
	if strings.Count(out, "disable_prepared_statements=") != 1 {
		t.Fatalf("expected single disable_prepared_statements, got %s", out)
	}
	if strings.Count(strings.ToLower(out), "binary_parameters=") != 1 {
		t.Fatalf("expected single binary_parameters, got %s", out)
	}
}

func TestWithDisablePreparedStatements_RespectsPreferSimpleProtocol(t *testing.T) {
	dsn := "postgres://u:p@h/db?prefer_simple_protocol=true"
	out := withDisablePreparedStatements(dsn)
	if out != dsn {
		t.Fatalf("expected unchanged DSN when prefer_simple_protocol present, got %s", out)
	}
}
