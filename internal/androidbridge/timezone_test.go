package androidbridge

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func TestAndroidTimezoneUsesSystemZoneName(t *testing.T) {
	before := time.Local
	defer func() { time.Local = before }()
	if err := setLocalTimezone("America/Tegucigalpa"); err != nil {
		t.Fatal(err)
	}
	utc := time.Date(2026, 10, 13, 14, 52, 0, 0, time.UTC)
	if got := utc.In(time.Local).Format("15:04"); got != "08:52" {
		t.Fatalf("local clock=%s", got)
	}
	if err := setLocalTimezone("invalid/zone"); err == nil {
		t.Fatal("invalid zone accepted")
	}
}
