package autoupdater

import (
	"testing"
)

// Here's our test file. Include methods to help test the autoupdater package

func TestUpdateAvailable(t *testing.T) {
	updater := &AutoUpdater{}

	_, _, err := updater.UpdateAvailable()

	if err.Error() != "Get VERSION: unsupported protocol scheme \"\"" {
		t.Fail()
	}
}
