package jvm

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// RuntimeDirName is the folder under %AppData%\STALART\updates\ that
// contains bin\javaw.exe. Update when the launcher ships a new JDK build.
const RuntimeDirName = "java25-windows-x86-64"

func referenceBin(name string) (string, error) {
	roaming, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(roaming, "STALART", "updates", RuntimeDirName, "bin", name), nil
}

func sha256File(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// MatchRuntime reports whether invokedExe is the STALART-bundled java.exe or
// javaw.exe by comparing SHA-256 against the reference image under
// %AppData%\STALART\updates\<RuntimeDirName>\bin\. Returns (false, err) when
// the reference file is missing so callers can log once.
func MatchRuntime(invokedExe string) (bool, error) {
	base := filepath.Base(invokedExe)
	var refPath string
	var refErr error
	switch {
	case strings.EqualFold(base, "javaw.exe"):
		refPath, refErr = referenceBin("javaw.exe")
	case strings.EqualFold(base, "java.exe"):
		refPath, refErr = referenceBin("java.exe")
	default:
		return false, nil
	}
	if refErr != nil {
		return false, refErr
	}
	if _, err := os.Stat(refPath); err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("reference JVM image missing: %s", refPath)
		}
		return false, err
	}
	want, err := sha256File(refPath)
	if err != nil {
		return false, fmt.Errorf("hash reference image: %w", err)
	}
	got, err := sha256File(invokedExe)
	if err != nil {
		return false, fmt.Errorf("hash invoked image: %w", err)
	}
	return subtle.ConstantTimeCompare(want, got) == 1, nil
}
