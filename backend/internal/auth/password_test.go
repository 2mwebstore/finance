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

func TestHashPasswordRejectsAnythingButSixDigits(t *testing.T) {
	for _, pin := range []string{"", "short", "1234", "12345", "1234567", "12a456", "１２３４５６"} {
		if _, err := HashPassword(pin); err == nil {
			t.Errorf("HashPassword(%q) should fail", pin)
		}
	}
}

// PINs hashed while they were 4 digits must keep signing in until the user picks a new one.
func TestVerifyPasswordAcceptsLegacyFourDigitPIN(t *testing.T) {
	hash, err := encodeHash("2468")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyPassword("2468", hash); err != nil || !ok {
		t.Fatalf("legacy PIN should verify, got ok=%v err=%v", ok, err)
	}
	if ok, _ := VerifyPassword("246800", hash); ok {
		t.Fatal("a different PIN should not verify")
	}
}

func TestIsCurrentPIN(t *testing.T) {
	cases := map[string]bool{"246810": true, "000000": true, "2468": false, "24681": false, "2468100": false, "24681a": false}
	for pin, want := range cases {
		if got := IsCurrentPIN(pin); got != want {
			t.Errorf("IsCurrentPIN(%q) = %v, want %v", pin, got, want)
		}
	}
}
