package voicerec

import (
	"bytes"
	"testing"
)

type captureEncoder struct {
	bytes.Buffer
	closed bool
}

func (e *captureEncoder) Close() error { e.closed = true; return nil }
func TestFinishPCMStopsCaptureBeforeDraining(t *testing.T) {
	q := make(chan []byte, 2)
	q <- []byte{1, 2}
	e := new(captureEncoder)
	stopped := false
	err := finishPCM(func() { stopped = true; q <- []byte{3, 4} }, q, e)
	if err != nil || !stopped || !e.closed || !bytes.Equal(e.Bytes(), []byte{1, 2, 3, 4}) {
		t.Fatalf("capture finalization: stopped=%v closed=%v PCM=%v err=%v", stopped, e.closed, e.Bytes(), err)
	}
}
