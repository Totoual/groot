package toolchains

import "testing"

func TestLatestPythonPatchVersion(t *testing.T) {
	listing := `
	<html>
	  <a href="3.13.7/">3.13.7/</a>
	  <a href="3.14.0/">3.14.0/</a>
	  <a href="3.14.2/">3.14.2/</a>
	  <a href="3.14.1/">3.14.1/</a>
	  <a href="3.14.2/">3.14.2/</a>
	  <a href="3.15.0/">3.15.0/</a>
	</html>
	`

	got, err := latestPythonPatchVersion(listing, "3.14")
	if err != nil {
		t.Fatalf("latestPythonPatchVersion returned error: %v", err)
	}

	if got != "3.14.2" {
		t.Fatalf("latestPythonPatchVersion = %q, want %q", got, "3.14.2")
	}
}

func TestPythonVersionHelpers(t *testing.T) {
	if !isPythonMinorSeries("3.14") {
		t.Fatal("expected 3.14 to be treated as a Python minor series")
	}
	if isPythonMinorSeries("3.14.0") {
		t.Fatal("expected 3.14.0 not to be treated as a Python minor series")
	}
	if !isExactPythonVersion("3.14.0") {
		t.Fatal("expected 3.14.0 to be treated as an exact Python release")
	}
	if isExactPythonVersion("3.14") {
		t.Fatal("expected 3.14 not to be treated as an exact Python release")
	}
}
