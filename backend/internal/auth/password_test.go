package auth

import (
	"strings"
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

func TestHashPasswordAcceptsLettersAndSigns(t *testing.T) {
	ok := []string{"2468", "24681", "246810", "short", "1234567", "12a4", "Ab-+*_/:;()$&@\"x", "១២៣៤", "pass word", strings.Repeat("x", MaxPasswordLength)}
	for _, pw := range ok {
		if _, err := HashPassword(pw); err != nil {
			t.Errorf("HashPassword(%q) failed: %v", pw, err)
		}
	}
	for _, pw := range []string{"", "123", "ab\tcd", "ab\ncd", strings.Repeat("x", MaxPasswordLength+1), "ab\xffcd"} {
		if _, err := HashPassword(pw); err == nil {
			t.Errorf("HashPassword(%q) should fail", pw)
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

func TestIsValidPassword(t *testing.T) {
	// Lengths are in characters: four Khmer digits are 12 bytes but 4 characters.
	cases := map[string]bool{"246810": true, "2468": true, "24681a": true, "-+*_": true, "១២៣៤": true, "246": false, "១២៣": false, "a\u0000bcd": false}
	for pw, want := range cases {
		if got := IsValidPassword(pw); got != want {
			t.Errorf("IsValidPassword(%q) = %v, want %v", pw, got, want)
		}
	}
}

func TestVerifyPasswordWithSigns(t *testing.T) {
	hash, err := HashPassword("Ab-+*12")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyPassword("Ab-+*12", hash); err != nil || !ok {
		t.Fatalf("password should verify, got ok=%v err=%v", ok, err)
	}
	if ok, _ := VerifyPassword("ab-+*12", hash); ok {
		t.Fatal("letters' case counts")
	}
}
