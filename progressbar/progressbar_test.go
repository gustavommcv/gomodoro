package progressbar

import (
	"bytes"
	"testing"
)

func TestProgressBar(t *testing.T) {
	buffer := bytes.Buffer{}

	ProgressBar(&buffer, 50, 100)
	got := buffer.String()
	want := "50% [█████-----]"

	if got != want {
		t.Errorf("expect %v, got %v", want, got)
	}
}
