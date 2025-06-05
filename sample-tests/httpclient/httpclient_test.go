package httpclient_test

import (
	"testing"
	"time"
)

func TestHttpClient(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		// pass
	})

	t.Run("Post", func(t *testing.T) {
		t.Error("Failed to send post request")
	})

	t.Run("Delete", func(t *testing.T) {
		// pass
	})

	t.Run("Timeouts", func(t *testing.T) {
		t.Run("Short timeout", func(t *testing.T) {
			time.Sleep(time.Second * 3)
			t.Errorf("Timeout error after 3 seconds")
		})
		t.Run("Long timeout", func(t *testing.T) {
			// pass
		})
	})
}
