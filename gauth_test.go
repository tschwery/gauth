package main

import (
	"testing"
)

func TestNormalizeSecret(t *testing.T) {
	var norm string

	norm = normalizeSecret("A1B2C3")
	if norm != "A1B2C3==" {
		t.Error(norm)
	}

	norm = normalizeSecret("dGVzdAo")
	if norm != "DGVZDAO=" {
		t.Error(norm)
	}

	norm = normalizeSecret("dG Vz dA oO")
	if norm != "DGVZDAOO" {
		t.Error(norm)
	}
}

func TestAuthCodeFixedTime(t *testing.T) {
	var code string
	var err error

	code, err = AuthCode("DGVZDAO=", 0, "TOTP")
	if err != nil {
		t.Error(err)
	}
	if code != "117080" {
		t.Error(code)
	}

	code, err = AuthCode("DGVZDAO=", 1000, "TOTP")
	if err != nil {
		t.Error(err)
	}
	if code != "046449" {
		t.Error(code)
	}
}
