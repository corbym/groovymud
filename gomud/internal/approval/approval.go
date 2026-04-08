// Package approval provides a minimal approval-testing helper.
//
// Each call to Verify compares actual output against a committed
// "approved" file.  When they differ a ".received.txt" file is written
// beside the approved file and the test is failed, giving reviewers a
// human-readable diff they can inspect or copy into the approved file.
//
// On the very first run (no approved file yet) the file is created
// automatically so bootstrapping new tests requires no manual step.
package approval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Verify checks actual against the file at approvedPath.
//
//   - If the approved file does not exist it is created from actual (first-run
//     bootstrap) and the test passes.
//   - If they match the test passes.
//   - If they differ a "<name>.received.txt" file is written alongside the
//     approved file and the test is failed with a clear message.
func Verify(t *testing.T, approvedPath, actual string) {
	t.Helper()

	// Normalise line endings so CR LF (telnet) == LF (file on disk).
	actual = strings.ReplaceAll(actual, "\r\n", "\n")
	actual = strings.ReplaceAll(actual, "\r", "\n")

	approved, err := os.ReadFile(approvedPath)
	if os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(approvedPath), 0o755); mkErr != nil {
			t.Fatalf("approval: mkdir %s: %v", filepath.Dir(approvedPath), mkErr)
		}
		if wErr := os.WriteFile(approvedPath, []byte(actual), 0o644); wErr != nil {
			t.Fatalf("approval: write approved %s: %v", approvedPath, wErr)
		}
		t.Logf("approval: created new approved file %s", approvedPath)
		return
	}
	if err != nil {
		t.Fatalf("approval: read %s: %v", approvedPath, err)
	}

	approvedStr := strings.ReplaceAll(string(approved), "\r\n", "\n")
	approvedStr = strings.ReplaceAll(approvedStr, "\r", "\n")

	if actual == approvedStr {
		return
	}

	// Write the received file so the CI workflow can diff it.
	receivedPath := receivedPathFor(approvedPath)
	if wErr := os.WriteFile(receivedPath, []byte(actual), 0o644); wErr != nil {
		t.Logf("approval: could not write received file %s: %v", receivedPath, wErr)
	}
	t.Errorf("approval mismatch:\n  approved : %s\n  received : %s\nUpdate the approved file to accept the new output.",
		approvedPath, receivedPath)
}

// receivedPathFor returns the path of the ".received.txt" file that sits
// beside the approved file.
func receivedPathFor(approvedPath string) string {
	base := strings.TrimSuffix(approvedPath, ".approved.txt")
	return base + ".received.txt"
}
