package main

import "testing"

func TestLuhn(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		isValid := IsValid("4561261212345467")
		if isValid != true {
			t.Errorf("IsValid(valid_card) != true")
		}
	})
	t.Run("invalid", func(t *testing.T) {
		isValid := IsValid("4561261212345464")
		if isValid != false {
			t.Errorf("IsValid(valid_card) != false")
		}
	})
}
