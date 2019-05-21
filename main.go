package main

import (
	"fmt"
	"strconv"

	"github.com/antoniomika/autoupdater/autoupdater"
)

var version = "0"
var signer = ""

func main() {
	realVersion, err := strconv.Atoi(version)
	if err != nil {
		fmt.Println("Unable to parse version.", err)
	}

	updater := &autoupdater.AutoUpdater{
		UpdateBaseURL:  "https://storage.googleapis.com/autoupdater-artifacts",
		CurrentVersion: realVersion,
		Signer:         signer,
	}

	updateAvailable, updateVersion, err := updater.UpdateAvailable()
	if err != nil {
		fmt.Println("Unable to do automatic updates.", err)
	}

	if updateAvailable {
		fmt.Println("Update available, attempting update.")

		status, err := updater.Update(updateVersion)
		if err != nil || !status {
			fmt.Println("Unable to do automatic updates.", err)
		}
	}

	fmt.Println("Hello world! From version:", realVersion)
}
