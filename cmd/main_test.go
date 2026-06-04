// Empty test file required to work around https://github.com/golang/go/issues/75031.
// Go 1.25 invokes covdata even for packages with no test files, causing
// "go: no such tool covdata" when running go test -coverprofile ./...
// Remove once the project upgrades to Go 1.27+.
package main
