package autoupdater

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/openpgp"
)

// AutoUpdater is the base struct for handling autoupdates
type AutoUpdater struct {
	// UpdateBaseURL is the base URL for update operations.
	// We expect a few things here:
	// 1. `signing_key.asc` - the signing key that the binary should be verified with.
	// 2. `latest.json` - a JSON blob containing information about the latest release. Fields are contained in the `Latest` struct.
	// 3. `versions/{{.Version}}.sig` - a signed file containing this version. Allows support for reverting.
	UpdateBaseURL string

	// The current version of the application.
	CurrentVersion int

	// The armored pub key that signed the binary. If blank, it is retrieved from the server.
	Signer string
}

// Latest contains information from latest.json
type Latest struct {
	// The version found
	Version int
}

// UpdateAvailable checks and returns if there is an update available
func (a *AutoUpdater) UpdateAvailable() (bool, int, error) {
	// Get the latest version from the /VERSION file
	latest, err := a.getLatest()
	if err != nil {
		return false, 0, err
	}

	// Compare the version
	if latest.Version > a.CurrentVersion {
		return true, latest.Version, nil
	}

	return false, 0, nil
}

// Update updates the binary to the version. If the version provided is older than the current one, it will revert to that version.
func (a *AutoUpdater) Update(version int) (bool, error) {
	// Load the GPG signing key
	key, err := a.getSigningKey()
	if err != nil {
		return false, err
	}

	// Load the signed version update
	reader, err := a.getVersion(version)
	if err != nil {
		return false, err
	}
	defer reader.Close()

	// Read in the signed version update
	md, err := openpgp.ReadMessage(reader, key, nil, nil)
	if err != nil {
		return false, err
	}

	// MessageDetails.IsSigned and SignedBy will be set if the signed update was correct
	if !md.IsSigned || md.SignedBy == nil {
		return false, fmt.Errorf("unable to verify update")
	}

	// The signature is validated as the unverified body is read
	body, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return false, err
	}

	// Ensure no signature error exists. If it does, there was a problem with the signed update
	if md.SignatureError != nil {
		return false, md.SignatureError
	}

	// Find the current executable
	executablePath, err := os.Executable()
	if err != nil {
		return false, err
	}

	// Create a temporary file for where the update will go
	executablePathUpdate := executablePath + ".update"
	if _, err := os.Stat(executablePathUpdate); err == nil {
		return false, err
	}

	// Write the verified body to the file
	if err := ioutil.WriteFile(executablePathUpdate, body, 0755); err != nil {
		return false, err
	}

	// Everything has been okay to this point, let's update the current binary
	if err := os.Rename(executablePathUpdate, executablePath); err != nil {
		return false, err
	}

	// Check the cli arguments to pass into the new process
	var cliArgs []string
	if len(os.Args) > 1 {
		cliArgs = os.Args[1:]
	}

	// Setup the new process, and use the old process' Std(in|err|out)
	command := exec.Command(os.Args[0], cliArgs...)
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	// Run the command, and handle if an error occurred
	if err := command.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			os.Exit(exiterr.Sys().(syscall.WaitStatus).ExitStatus())
		} else {
			return false, err
		}
	}

	os.Exit(0)

	return true, nil
}

// getPath creates the path appeneded location for UpdateBaseURL
func (a *AutoUpdater) getPath(nextPath ...string) (string, error) {
	updateBaseURL, err := url.Parse(a.UpdateBaseURL)
	if err != nil {
		return "", err
	}

	prependedPath := append([]string{updateBaseURL.Path}, nextPath...)

	updateBaseURL.Path = path.Join(prependedPath...)

	return updateBaseURL.String(), nil
}

// getVersion loads the update path and downloads the specified version
func (a *AutoUpdater) getVersion(version int) (io.ReadCloser, error) {
	versionPath := a.getVersionPath(version)

	latestURL, err := a.getPath("versions", strconv.Itoa(version), versionPath)
	if err != nil {
		return nil, err
	}

	res, err := http.Get(latestURL)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Specified remote version not available: %d", res.StatusCode)
	}
	return res.Body, nil
}

// getLatest loads the latest version from the UpdateBaseURL
func (a *AutoUpdater) getLatest() (*Latest, error) {
	latestURL, err := a.getPath("VERSION")
	if err != nil {
		return nil, err
	}

	res, err := http.Get(latestURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	version, err := strconv.Atoi(string(body))
	if err != nil {
		return nil, err
	}

	latest := &Latest{
		Version: version,
	}

	return latest, nil
}

// getSigningKey loads the signing key from locally or the update host. Ideally this is compiled with the binary
func (a *AutoUpdater) getSigningKey() (openpgp.KeyRing, error) {
	if a.Signer == "" {
		signingKey, err := a.getPath("signing_key.asc")
		if err != nil {
			return nil, err
		}

		res, err := http.Get(signingKey)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		return openpgp.ReadArmoredKeyRing(res.Body)
	}

	return openpgp.ReadArmoredKeyRing(strings.NewReader(a.Signer))
}

// getVersionPath returns the signed update file path
func (a *AutoUpdater) getVersionPath(version int) string {
	versionString := []string{strconv.Itoa(version)}
	versionString = append(versionString, runtime.GOOS, runtime.GOARCH)
	finalVersion := strings.Join(versionString, "-")

	finalVersion += ".sig"
	return finalVersion
}
