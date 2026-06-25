package resources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestSSHKeyStatusSatisfiedWhenKeypairExists(t *testing.T) {
	path := writeSSHKeypair(t)
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("ssh-keygen", []string{"-y", "-f", path}, testEd25519DerivedPublicKey+"\n")},
	}}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateSatisfied, "SSH keypair exists")
	if result.Details["path"] != path {
		t.Fatalf("Details[path] = %q, want %q", result.Details["path"], path)
	}
	expectCalls(t, runner.calls, []commandCall{{name: "ssh-keygen", args: []string{"-y", "-f", path}}})
}

func TestSSHKeyStatusFailsWhenPublicKeyTypeDiffers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(path+".pub", []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC user@example.com\n"), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	resource := NewSSHKey(path, "ed25519", "user@example.com", &fakeRunner{})

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want key type mismatch error")
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateFailed, "SSH key type differs")
	if !containsError(err, "want ssh-ed25519") {
		t.Fatalf("error = %q, want expected public key type", err.Error())
	}
}

func TestSSHKeyStatusFailsWhenPublicKeyDoesNotMatchPrivateKey(t *testing.T) {
	path := writeSSHKeypair(t)
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("ssh-keygen", []string{"-y", "-f", path}, otherEd25519DerivedPublicKey+"\n")},
	}}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want key mismatch error")
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateFailed, "SSH public key does not match private key")
	expectCalls(t, runner.calls, []commandCall{{name: "ssh-keygen", args: []string{"-y", "-f", path}}})
}

func TestSSHKeyStatusFailsWhenPrivateKeyCannotBeDerived(t *testing.T) {
	path := writeSSHKeypair(t)
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("ssh-keygen", []string{"-y", "-f", path}, 1)},
	}}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want ssh-keygen failure")
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateFailed, "could not inspect SSH private key")
	expectCalls(t, runner.calls, []commandCall{{name: "ssh-keygen", args: []string{"-y", "-f", path}}})
}

func TestSSHKeyStatusMissingWhenPrivateKeyIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "id_ed25519")
	runner := &fakeRunner{}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateMissing, "SSH private key is missing")
	expectCalls(t, runner.calls, nil)
}

func TestSSHKeyStatusFailsWhenPrivateKeyIsMissingButPublicKeyExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path+".pub", []byte("existing-public\n"), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	runner := &fakeRunner{}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want unsafe public key error")
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateFailed, "SSH private key is missing but public key exists")
	expectCalls(t, runner.calls, nil)
}

func TestSSHKeyStatusChangedWhenPublicKeyIsMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	runner := &fakeRunner{}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateChanged, "SSH public key is missing")
	expectCalls(t, runner.calls, nil)
}

func TestSSHKeyApplyGeneratesMissingKeypair(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "id_ed25519")
	runner := &fakeRunner{}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), sshKeyType, "create", true, "generated SSH keypair")
	expectCalls(t, runner.calls, []commandCall{
		{name: "ssh-keygen", args: []string{"-t", "ed25519", "-C", "user@example.com", "-f", path, "-N", ""}},
	})
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Fatalf("SSH directory stat = (%v, %v), want directory", info, err)
	}
}

func TestSSHKeyApplyDoesNotOverwriteExistingPublicKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	originalPublicKey := []byte("existing-public\n")
	if err := os.WriteFile(path+".pub", originalPublicKey, 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	runner := &fakeRunner{}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want unsafe public key error")
	}

	expectApply(t, result, resource.ID(), sshKeyType, "fail", false, "SSH private key is missing but public key exists")
	expectCalls(t, runner.calls, nil)
	contents, err := os.ReadFile(path + ".pub")
	if err != nil {
		t.Fatalf("read public key: %v", err)
	}
	if string(contents) != string(originalPublicKey) {
		t.Fatalf("public key contents = %q, want %q", string(contents), string(originalPublicKey))
	}
}

func TestSSHKeyApplyReportsGenerateFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "id_ed25519")
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("ssh-keygen", []string{"-t", "ed25519", "-f", path, "-N", ""}, 1)},
	}}
	resource := NewSSHKey(path, "ed25519", "", runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want ssh-keygen failure")
	}

	expectApply(t, result, resource.ID(), sshKeyType, "create", false, "could not generate SSH keypair")
}

func TestSSHKeyApplyRecreatesMissingPublicKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	runner := &fakeRunner{}
	resource := NewSSHKey(path, "ed25519", "", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), sshKeyType, "public_key", true, "recreated SSH public key")
	expectCalls(t, runner.calls, []commandCall{
		{name: "sh", args: []string{"-c", "ssh-keygen -y -f \"$1\" > \"$2\"", "kitout", path, path + ".pub"}},
	})
}

func TestSSHKeyDryRunDoesNotGenerateKeypair(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "id_ed25519")
	runner := &fakeRunner{}
	resource := NewSSHKey(path, "ed25519", "user@example.com", runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, nil)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want key to remain missing", path, err)
	}
}

func TestSSHKeyStatusFailsForUnsupportedType(t *testing.T) {
	resource := NewSSHKey(filepath.Join(t.TempDir(), "id_rsa"), "rsa", "", &fakeRunner{})

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want unsupported type error")
	}

	expectStatus(t, result, resource.ID(), sshKeyType, engine.StateFailed, "unsupported SSH key type \"rsa\"")
}

const (
	testEd25519PublicKeyBody     = "AAAAC3NzaC1lZDI1NTE5AAAAIKitoutTestKey"
	otherEd25519PublicKeyBody    = "AAAAC3NzaC1lZDI1NTE5AAAAIOtherKitoutTestKey"
	testEd25519DerivedPublicKey  = "ssh-ed25519 " + testEd25519PublicKeyBody
	otherEd25519DerivedPublicKey = "ssh-ed25519 " + otherEd25519PublicKeyBody
	testEd25519PublicKey         = testEd25519DerivedPublicKey + " user@example.com"
)

func writeSSHKeypair(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(path+".pub", []byte(testEd25519PublicKey+"\n"), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	return path
}
