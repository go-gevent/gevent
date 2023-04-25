package utils

import "testing"

func TestAtomicBool(t *testing.T) {

	t.Run("AtomicBool", func(t *testing.T) {
		var isOk AtomicBool
		if isOk.Get() != false {
			t.Fatal("expect false")
		}

		isOk.Set(true)
		if isOk.Get() != true {
			t.Fatal("expect true")
		}

		isOk.Set(false)
		if isOk.Get() != false {
			t.Fatal("expect false")
		}
	})
}
