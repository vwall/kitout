package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const sshKeyType = "ssh_key"

var (
	errSSHPublicKeyPathSymlink   = errors.New("SSH public key path is a symlink")
	errSSHPublicKeyPathDirectory = errors.New("SSH public key path is a directory")
	errSSHPublicKeyAlreadyExists = errors.New("SSH public key already exists")
)

// SSHKeyResource ensures an SSH keypair exists.
type SSHKeyResource struct {
	path    string
	keyType string
	comment string
	runner  platform.Runner
}

var _ engine.Resource = SSHKeyResource{}

// NewSSHKey returns a resource for one SSH keypair.
func NewSSHKey(path, keyType, comment string, runner platform.Runner) SSHKeyResource {
	return SSHKeyResource{
		path:    path,
		keyType: keyType,
		comment: comment,
		runner:  runner,
	}
}

func (resource SSHKeyResource) ID() string {
	return sshKeyType + ":" + resource.path
}

func (resource SSHKeyResource) Type() string {
	return sshKeyType
}

func (resource SSHKeyResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return resource.status(engine.StateFailed, "context canceled while checking SSH key"), err
	}
	if err := resource.validate(false); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if err := resource.validatePublicKeyPath(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	privateInfo, err := os.Stat(resource.path)
	if errors.Is(err, os.ErrNotExist) {
		return resource.statusWhenPrivateKeyMissing()
	}
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect SSH private key"), err
	}
	if privateInfo.IsDir() {
		return resource.status(engine.StateFailed, "SSH private key path is a directory"), errors.New("SSH private key path is a directory")
	}

	if _, err := resource.publicKeyFileInfo(); errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateChanged, "SSH public key is missing"), nil
	} else if err != nil {
		return resource.status(engine.StateFailed, sshPublicKeyInspectMessage(err)), err
	}
	publicKey, err := resource.publicKey()
	if err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if publicKey.algorithm != publicKeyAlgorithm(resource.keyType) {
		err := fmt.Errorf("SSH public key type is %s, want %s", publicKey.algorithm, publicKeyAlgorithm(resource.keyType))
		return resource.status(engine.StateFailed, "SSH key type differs"), err
	}
	privateKey, err := resource.derivedPublicKey(ctx)
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect SSH private key"), err
	}
	if privateKey != publicKey {
		err := errors.New("SSH public key does not match private key")
		return resource.status(engine.StateFailed, err.Error()), err
	}

	return resource.status(engine.StateSatisfied, "SSH keypair exists"), nil
}

func (resource SSHKeyResource) statusWhenPrivateKeyMissing() (engine.StatusResult, error) {
	if _, err := resource.publicKeyFileInfo(); errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateMissing, "SSH private key is missing"), nil
	} else if err != nil {
		return resource.status(engine.StateFailed, sshPublicKeyInspectMessage(err)), err
	}

	err := errors.New("SSH private key is missing but public key exists")
	return resource.status(engine.StateFailed, err.Error()), err
}

func (resource SSHKeyResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	if err := resource.validate(true); err != nil {
		return resource.applyResult("fail", false, err.Error()), err
	}

	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "SSH keypair already exists"), nil
	case engine.StateMissing:
		if err := resource.ensureKeyParent(); err != nil {
			return resource.applyResult("create", false, "could not create SSH key directory"), err
		}
		if _, err := resource.runner.Run(ctx, "ssh-keygen", resource.generateArgs()...); err != nil {
			return resource.applyResult("create", false, "could not generate SSH keypair"), err
		}
		return resource.applyResult("create", true, "generated SSH keypair"), nil
	case engine.StateChanged:
		result, err := resource.runner.Run(ctx, "ssh-keygen", "-y", "-f", resource.path)
		if err != nil {
			return resource.applyResult("public_key", false, "could not recreate SSH public key"), err
		}
		if err := resource.writePublicKey(result.Stdout); err != nil {
			return resource.applyResult("public_key", false, "could not recreate SSH public key"), err
		}
		return resource.applyResult("public_key", true, "recreated SSH public key"), nil
	default:
		err := fmt.Errorf("cannot apply SSH key %s from state %s", resource.path, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource SSHKeyResource) validate(requireRunner bool) error {
	if resource.path == "" {
		return errors.New("SSH key path is required")
	}
	if resource.keyType == "" {
		return errors.New("SSH key type is required")
	}
	if resource.keyType != "ed25519" {
		return fmt.Errorf("unsupported SSH key type %q", resource.keyType)
	}
	if requireRunner && resource.runner == nil {
		return errors.New("command runner is required")
	}
	return nil
}

func (resource SSHKeyResource) generateArgs() []string {
	args := []string{"-t", resource.keyType}
	if resource.comment != "" {
		args = append(args, "-C", resource.comment)
	}
	return append(args, "-f", resource.path, "-N", "")
}

func (resource SSHKeyResource) publicPath() string {
	return resource.path + ".pub"
}

func (resource SSHKeyResource) ensureKeyParent() error {
	if err := resource.validatePublicKeyPath(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resource.path), 0o700); err != nil {
		return err
	}
	return resource.validatePublicKeyPath()
}

func (resource SSHKeyResource) validatePublicKeyPath() error {
	for _, ancestor := range pathAncestors(resource.publicPath()) {
		info, err := os.Lstat(ancestor)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("could not inspect SSH public key ancestor %s: %w", ancestor, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if isAllowedDarwinSystemSymlink(ancestor) {
				continue
			}
			return fmt.Errorf("SSH public key ancestor %s must not be a symlink", ancestor)
		}
		if !info.IsDir() {
			return fmt.Errorf("SSH public key ancestor %s must be a directory", ancestor)
		}
	}
	return nil
}

func (resource SSHKeyResource) publicKeyFileInfo() (os.FileInfo, error) {
	if err := resource.validatePublicKeyPath(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(resource.publicPath())
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errSSHPublicKeyPathSymlink
	}
	if info.IsDir() {
		return nil, errSSHPublicKeyPathDirectory
	}
	return info, nil
}

func sshPublicKeyInspectMessage(err error) string {
	if errors.Is(err, errSSHPublicKeyPathSymlink) || errors.Is(err, errSSHPublicKeyPathDirectory) {
		return err.Error()
	}
	return "could not inspect SSH public key"
}

func (resource SSHKeyResource) publicKey() (sshPublicKey, error) {
	contents, err := os.ReadFile(resource.publicPath())
	if err != nil {
		return sshPublicKey{}, fmt.Errorf("could not inspect SSH public key: %w", err)
	}
	return parseSSHPublicKey(string(contents))
}

func (resource SSHKeyResource) derivedPublicKey(ctx context.Context) (sshPublicKey, error) {
	if resource.runner == nil {
		return sshPublicKey{}, errors.New("command runner is required")
	}
	result, err := resource.runner.Run(ctx, "ssh-keygen", "-y", "-f", resource.path)
	if err != nil {
		return sshPublicKey{}, fmt.Errorf("could not inspect SSH private key: %w", err)
	}
	key, err := parseSSHPublicKey(result.Stdout)
	if err != nil {
		return sshPublicKey{}, fmt.Errorf("could not parse SSH private key public output: %w", err)
	}
	return key, nil
}

func (resource SSHKeyResource) writePublicKey(contents string) error {
	if err := resource.validatePublicKeyPath(); err != nil {
		return err
	}
	// Exclusive no-follow creation rejects a final-path symlink inserted after Status.
	file, err := os.OpenFile(resource.publicPath(), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			if _, inspectErr := resource.publicKeyFileInfo(); inspectErr != nil {
				return inspectErr
			}
			return errSSHPublicKeyAlreadyExists
		}
		return err
	}
	if _, err := file.WriteString(contents); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

type sshPublicKey struct {
	algorithm string
	body      string
}

func parseSSHPublicKey(contents string) (sshPublicKey, error) {
	fields := strings.Fields(contents)
	if len(fields) < 2 {
		return sshPublicKey{}, errors.New("could not parse SSH public key")
	}
	return sshPublicKey{
		algorithm: fields[0],
		body:      fields[1],
	}, nil
}

func publicKeyAlgorithm(keyType string) string {
	switch keyType {
	case "ed25519":
		return "ssh-ed25519"
	default:
		return ""
	}
}

func (resource SSHKeyResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource SSHKeyResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource SSHKeyResource) details() map[string]string {
	details := map[string]string{
		"path":        resource.path,
		"public_path": resource.publicPath(),
		"type":        resource.keyType,
	}
	if resource.comment != "" {
		details["comment"] = resource.comment
	}
	return details
}
