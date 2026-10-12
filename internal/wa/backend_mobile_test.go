package wa

import "testing"

func TestOpenPreservesMobileMode(t *testing.T) {
	for _, mobile := range []bool{false, true} {
		b, err := Open(t.TempDir(), false, mobile)
		if err != nil {
			t.Fatalf("Open(mobile=%v): %v", mobile, err)
		}
		if b.mobile != mobile {
			t.Errorf("Open(mobile=%v) stored mobile=%v", mobile, b.mobile)
		}
		b.Close()
	}
}
