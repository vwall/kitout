package buildinfo

import "testing"

func TestCurrentReturnsInjectedMetadata(t *testing.T) {
	oldVersion := Version
	oldCommit := Commit
	oldBuildDate := BuildDate
	t.Cleanup(func() {
		Version = oldVersion
		Commit = oldCommit
		BuildDate = oldBuildDate
	})

	Version = "1.2.3"
	Commit = "abc1234"
	BuildDate = "2026-06-03T13:14:15Z"

	got := Current()
	if got.Version != "1.2.3" {
		t.Fatalf("Version = %q, want 1.2.3", got.Version)
	}
	if got.Commit != "abc1234" {
		t.Fatalf("Commit = %q, want abc1234", got.Commit)
	}
	if got.BuildDate != "2026-06-03T13:14:15Z" {
		t.Fatalf("BuildDate = %q, want 2026-06-03T13:14:15Z", got.BuildDate)
	}
}

func TestCurrentDefaultsEmptyMetadata(t *testing.T) {
	oldVersion := Version
	oldCommit := Commit
	oldBuildDate := BuildDate
	t.Cleanup(func() {
		Version = oldVersion
		Commit = oldCommit
		BuildDate = oldBuildDate
	})

	Version = " "
	Commit = ""
	BuildDate = "\t"

	got := Current()
	if got.Version != defaultVersion {
		t.Fatalf("Version = %q, want %q", got.Version, defaultVersion)
	}
	if got.Commit != unknown {
		t.Fatalf("Commit = %q, want %q", got.Commit, unknown)
	}
	if got.BuildDate != unknown {
		t.Fatalf("BuildDate = %q, want %q", got.BuildDate, unknown)
	}
}
