package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

func main() {
	var (
		testsDir        string
		shortFlag       bool
		pkgParallel     int
		count           int
		integrationRun  string
		integrationPath string
		verbose         bool
	)

	flag.StringVar(&testsDir, "tests-dir", "/app/tests", "directory containing compiled test binaries")
	flag.BoolVar(&shortFlag, "short", false, "run tests with -test.short")
	flag.IntVar(&pkgParallel, "pkg-parallel", runtime.NumCPU(), "number of packages to run in parallel")
	flag.IntVar(&count, "count", 1, "pass -test.count to disable caching when set to 1")
	flag.StringVar(&integrationRun, "integration-run", "", "regex of integration test(s) to run with -test.run")
	flag.StringVar(&integrationPath, "integration-path", "", "relative package path like 'api/config' for integration run")
	flag.BoolVar(&verbose, "v", true, "add -test.v to test binaries")
	flag.Parse()

	bins, err := collectTestBinaries(testsDir)
	if err != nil {
		fatal(err)
	}
	if len(bins) == 0 {
		fatal(errors.New("no test binaries found"))
	}

	var integrationBin string
	if integrationRun != "" {
		if integrationPath == "" {
			fatal(errors.New("integration-path is required when integration-run is set"))
		}
		integrationBin = filepath.Join(testsDir, filepath.FromSlash(integrationPath) + ".test")
		if _, err := os.Stat(integrationBin); err != nil {
			fatal(fmt.Errorf("integration binary not found at %s: %w", integrationBin, err))
		}
	}

	// Exclude the integration package from the unit pass to avoid double-running.
	unitBins := make([]string, 0, len(bins))
	for _, b := range bins {
		if integrationBin != "" && sameFile(b, integrationBin) {
			continue
		}
		unitBins = append(unitBins, b)
	}

	fmt.Println("==> Running unit tests")
	if err := runBinaries(unitBins, testArgs(verbose, shortFlag, count, 0), pkgParallel); err != nil {
		fatal(err)
	}

	if integrationBin != "" {
		fmt.Printf("==> Running integration tests in %s with -test.run=%s\n", integrationPath, integrationRun)
		args := testArgs(verbose, shortFlag, count, 1) // force -test.parallel=1 for integration
		args = append(args, "-test.run", integrationRun)
		if err := runBinaries([]string{integrationBin}, args, 1); err != nil {
			fatal(err)
		}
	}

	fmt.Println("==> All tests passed")
}

func collectTestBinaries(root string) ([]string, error) {
	var bins []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() { return nil }
		if strings.HasSuffix(d.Name(), ".test") {
			bins = append(bins, path)
		}
		return nil
	})
	if err != nil { return nil, err }
	sort.Strings(bins)
	return bins, nil
}

func testArgs(verbose, short bool, count, testParallel int) []string {
	args := []string{}
	if verbose { args = append(args, "-test.v") }
	if short { args = append(args, "-test.short") }
	if count > 0 { args = append(args, fmt.Sprintf("-test.count=%d", count)) }
	if testParallel > 0 { args = append(args, fmt.Sprintf("-test.parallel=%d", testParallel)) }
	return args
}

func runBinaries(bins []string, args []string, parallel int) error {
	if len(bins) == 0 { return nil }
	if parallel < 1 { parallel = 1 }
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, b := range bins {
		b := b
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func(){ <-sem }()
			cmd := exec.Command(b, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = os.Environ()
			// Set working dir to package-like path alongside the binary if it exists.
			if wd := strings.TrimSuffix(b, ".test"); wd != b {
				if fi, err := os.Stat(wd); err == nil && fi.IsDir() {
					cmd.Dir = wd
				} else {
					cmd.Dir = "/app"
				}
			}
			fmt.Printf("[RUN] %s %s\n", b, strings.Join(args, " "))
			if err := cmd.Run(); err != nil {
				mu.Lock()
				if firstErr == nil { firstErr = fmt.Errorf("%s failed: %w", b, err) }
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func sameFile(a, b string) bool {
	ap, _ := filepath.Abs(a)
	bp, _ := filepath.Abs(b)
	return ap == bp
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
