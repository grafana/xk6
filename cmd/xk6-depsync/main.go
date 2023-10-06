// main is the main package for xk6-depsync, a script that checks and provides commands to synchronize
// common dependencies with k6 core.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	k6Core     = "go.k6.io/k6"
	k6GoModURL = "https://proxy.golang.org/" + k6Core + "/@v/%s.mod"
)

func main() {
	gomod := "go.mod"
	if len(os.Args) >= 2 {
		gomod = os.Args[1]
	}

	file, err := os.Open(gomod)
	if err != nil {
		log.Fatalf("opening local go.mod: %v", err)
	}

	ownDeps, err := dependencies(file)
	if err != nil {
		log.Fatalf("reading dependencies: %v", err)
	}

	k6Version := ownDeps[k6Core]
	if k6Version == "" {
		log.Fatalf("k6 core %q not found in %q", k6Core, gomod)
	}

	log.Printf("detected k6 core version %s", k6Version)

	k6CoreVersionedURL := fmt.Sprintf(k6GoModURL, k6Version)
	//nolint:bodyclose // Single-run script.
	response, err := http.Get(k6CoreVersionedURL)
	if err != nil {
		log.Fatalf("error fetching k6 go.mod: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		log.Fatalf("got HTTP status %d for %s", response.StatusCode, k6CoreVersionedURL)
	}

	coreDeps, err := dependencies(response.Body)
	if err != nil {
		log.Fatalf("reading k6 core dependencies: %v", err)
	}

	//nolint:prealloc // Number of mismatched deps cannot be accurately predicted.
	var mismatched []string
	for dep, version := range ownDeps {
		coreVersion, inCore := coreDeps[dep]
		if !inCore {
			continue
		}

		if version == coreVersion {
			continue
		}

		log.Printf("Mismatched versions for %s: %s (this package) -> %s (core)", dep, version, coreVersion)
		mismatched = append(mismatched, fmt.Sprintf("%s@%s", dep, coreVersion))
	}

	if len(mismatched) == 0 {
		log.Println("All deps are in sync, nothing to do.")
		return
	}

	// TODO: Use slices.Sort when we move to a go version that has it on the stdlib.
	sort.Strings(mismatched)

	//nolint:forbidigo // We are willingly writing to stdout here.
	fmt.Printf("go get %s\n", strings.Join(mismatched, " "))
}

// dependencies reads a go.mod file from an io.Reader and returns a map of dependency name to their specified versions.
func dependencies(reader io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading contents: %w", err)
	}

	gomod, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing contents: %w", err)
	}

	deps := make(map[string]string)
	for _, req := range gomod.Require {
		deps[req.Mod.Path] = req.Mod.Version
	}

	return deps, nil
}
