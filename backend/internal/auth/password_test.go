package auth

import (
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := "713952"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !ok {
		t.Fatal("password should verify")
	}

	ok, err = VerifyPassword("000000", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if ok {
		t.Fatal("wrong password should not verify")
	}
}

func TestHashPasswordAcceptsFourToSixDigits(t *testing.T) {
	for _, pin := range []string{"2468", "24681", "246810"} {
		if _, err := HashPassword(pin); err != nil {
			t.Errorf("HashPassword(%q) failed: %v", pin, err)
		}
	}
	for _, pin := range []string{"", "short", "123", "1234567", "12a4", "１２３４"} {
		if _, err := HashPassword(pin); err == nil {
			t.Errorf("HashPassword(%q) should fail", pin)
		}
	}
}

func TestVerifyPasswordFourDigitPIN(t *testing.T) {
	hash, err := HashPassword("2468")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyPassword("2468", hash); err != nil || !ok {
		t.Fatalf("PIN should verify, got ok=%v err=%v", ok, err)
	}
	if ok, _ := VerifyPassword("246800", hash); ok {
		t.Fatal("a different PIN should not verify")
	}
}

func TestIsValidPIN(t *testing.T) {
	cases := map[string]bool{"246810": true, "2468": true, "24681": true, "246": false, "2468100": false, "24681a": false}
	for pin, want := range cases {
		if got := IsValidPIN(pin); got != want {
			t.Errorf("IsValidPIN(%q) = %v, want %v", pin, got, want)
		}
	}
}
