autoupdater
===========

A package for handling quick and secure auto updates.

How it works?
-------------
The autoupdater works by existing as a separate Golang package that can be used in any Golang project. The server for the project is 100% static, and currently uses [GCS](https://cloud.google.com/storage/) to host the necessary assets. The required assets are as follows:

1. `/VERSION` - A file ONLY containing the "newest" version
2. `versions/<vid>/<vid>-<GOOS>-<GOARCH>(.sig)` - A Golang binary and/or GPG signed Golang binary
3. `signing_key.asc` - The public key of the signing key to verify the signed Golang binary

We automatically build the demo binary from `main.go` using [Google Cloud Build](https://cloud.google.com/cloud-build/). The build steps necessary are shown in `cloudbuild.yaml`. The general steps are as follows:

1. Load the decryption key for `signing_key.asc.enc` from [KMS](https://cloud.google.com/kms/)
2. Build the binary for all of the neccessary platforms/architectures. Include the version and the signing key public key.
3. Sign the binaries with the decrypted `signing_key.asc`
4. Copy these artifacts, and upload to GCS.
5. Do it all in containers and provide a container that can be pulled and ran with the current version.

Once done, the general update flow is as follows:

1. Pull the `/VERSION` file.
2. Check if an update needs to happen.
3. Ask for the update, pull the file, verify the signature, and then replace the file.
4. Start the new process and pipe it's output through the old process.

Assumptions?
------------
The assumptions of this project are 1st off, you're using Golang. This will of course not work for other languages, but is fairly trivial to implement elsewhere. 

You also need to have a public server that can be used to serve the update assets, and there is a easy way to ensure these assets are located in the correct place as needed.

The updater assumes it has access to the same directory as the parent binary.

Final assumption is the assets on the remote server are placed in a specific place.

Tradeoffs?
----------
By using GPG, we are offloading hash checking, source verification, and other security limiations to the OpenPGP library for Golang. One may be interested in writing their own system for verfying all of these options, but as such this was the easiest way to do it. By allowing for a remote signing key to also be available, there is the possiblility of a malicious update being deployed if an attacker assumes control of the remote update host. By including the public key within the binary, this makes it less likely for this to occur.

License
-------
Copyright (c) 2019 Antonio Mika