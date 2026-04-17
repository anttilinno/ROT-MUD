package golden

import (
	"bytes"
	"flag"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"rotmud/pkg/combat"
)

// updateGolden, when true, rewrites testdata/entities.golden from the
// current fixture output instead of diffing against it. Default off; CI
// must NEVER set this flag (CI runs `go test ./...` without flags so the
// default-path diff executes — any mismatch is a parity regression).
//
// Pass via:
//
//	go test ./pkg/golden/ -run TestGolden -update
var updateGolden = flag.Bool("update", false, "regenerate testdata/entities.golden")

// goldenSeed is pinned at the constant 42. Every committed byte of
// testdata/entities.golden is a function of this seed; changing it
// invalidates the whole snapshot.
const goldenSeed = 42

// TestGolden is the parity gate. Run under the default path it diffs
// the freshly-computed fixture output against testdata/entities.golden
// and fails the build on any byte-level mismatch. Run with -update it
// rewrites that file.
func TestGolden(t *testing.T) {
	// Install deterministic RNG for the whole combat/magic/skills/ai call
	// tree (Plan 01 wired SetRand into the package-scope defaultRand).
	restore := combat.SetRand(rand.New(rand.NewSource(goldenSeed)))
	t.Cleanup(restore)

	var buf bytes.Buffer
	runFixture(&buf)

	path := filepath.Join("testdata", "entities.golden")
	got := buf.Bytes()

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden updated: %s (%d bytes)", path, len(got))
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v (run `go test ./pkg/golden/ -run TestGolden -update` to create)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf(
			"golden mismatch at %s\n"+
				"run `go test ./pkg/golden/ -run TestGolden -update` if this behavior change is intentional\n\n"+
				"--- want (%d bytes) ---\n%s\n"+
				"--- got (%d bytes) ---\n%s",
			path,
			len(want), want,
			len(got), got,
		)
	}
}
