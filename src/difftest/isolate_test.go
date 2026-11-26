package difftest

import (
	"os"
	"os/exec"
	"testing"
)

// isolatedEnv names the test a child process runs on its own.
const isolatedEnv = "FLUTTER_GO_DIFFTEST_ISOLATED"

// isolated runs the calling top-level test in a process of its own, as
// `flutter test` runs each test file in its own isolate: bindings are
// process-wide singletons (BindingBase can be constructed once), so a test
// that creates its own binding can't share a process with the tests using
// flutter_test's. It returns true in the child, where the test body should
// run, and false in the parent, after reporting the child's result.
func isolated(t *testing.T) bool {
	t.Helper()
	if os.Getenv(isolatedEnv) == t.Name() {
		return true
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), isolatedEnv+"="+t.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s in its own process: %v\n%s", t.Name(), err, out)
	}
	if testing.Verbose() {
		t.Logf("%s", out)
	}
	return false
}
