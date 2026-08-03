package auth

import "testing"

func TestChildPINHashRoundTrip(t *testing.T) {
	hash, err := HashChildPIN("4826")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "4826" {
		t.Fatal("PIN stored in plaintext")
	}
	if !VerifyChildPIN(hash, "4826") {
		t.Fatal("correct PIN rejected")
	}
	if VerifyChildPIN(hash, "4827") {
		t.Fatal("wrong PIN accepted")
	}
}

func TestChildPINRejectsWeakShape(t *testing.T) {
	for _, pin := range []string{"123", "12345", "12a4", ""} {
		if _, err := HashChildPIN(pin); err == nil {
			t.Fatalf("HashChildPIN(%q) should fail", pin)
		}
	}
}
