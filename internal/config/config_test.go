package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// testEnv points the environment at temporary directories and reports where
// each file has to be written on this platform.
//
// appConfigDir and xdgConfigHome are the same directory on Linux and diverge
// elsewhere, because the package resolves "config home" two different ways on
// purpose:
//
//   - Load() finds config.yaml through os.UserConfigDir(), which honours
//     XDG_CONFIG_HOME only on Linux. macOS uses ~/Library/Application Support
//     and Windows uses %AppData% — the locations README.md documents.
//   - xdgUserDir() reads $XDG_CONFIG_HOME directly on every platform, because
//     user-dirs.dirs is a freedesktop file that only exists where XDG does.
//
// Assuming those two coincide is what made this whole package fail on Windows
// and macOS while passing on Linux.
type testEnv struct {
	home          string // os.UserHomeDir() reports this
	appConfigDir  string // os.UserConfigDir() reports this; config.yaml goes in <this>/msd
	xdgConfigHome string // $XDG_CONFIG_HOME; user-dirs.dirs goes here
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()

	env := testEnv{home: t.TempDir(), xdgConfigHome: t.TempDir()}

	t.Setenv("HOME", env.home)
	if runtime.GOOS == "windows" {
		// os.UserHomeDir reads %USERPROFILE% on Windows, not $HOME. Setting
		// only HOME leaves it pointing at the real profile directory.
		t.Setenv("USERPROFILE", env.home)
	}
	t.Setenv("XDG_CONFIG_HOME", env.xdgConfigHome)

	switch runtime.GOOS {
	case "windows":
		// os.UserConfigDir reads %AppData%. Point it at the same directory so
		// the test layout matches Linux's.
		t.Setenv("AppData", env.xdgConfigHome)
		env.appConfigDir = env.xdgConfigHome
	case "darwin":
		// os.UserConfigDir is $HOME/Library/Application Support with no env
		// override, so it follows the temporary HOME set above.
		env.appConfigDir = filepath.Join(env.home, "Library", "Application Support")
		if err := os.MkdirAll(env.appConfigDir, 0o755); err != nil {
			t.Fatalf("create app config dir: %v", err)
		}
	default:
		env.appConfigDir = env.xdgConfigHome
	}

	return env
}

// writeConfig writes config.yaml where Load() will look for it.
func (e testEnv) writeConfig(t *testing.T, body string) {
	t.Helper()
	dir := filepath.Join(e.appConfigDir, "msd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}

// writeUserDirs writes a freedesktop user-dirs.dirs where xdgUserDir() reads it.
func (e testEnv) writeUserDirs(t *testing.T, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(e.xdgConfigHome, "user-dirs.dirs"), []byte(body), 0o644); err != nil {
		t.Fatalf("write user dirs: %v", err)
	}
}

func TestDefaults(t *testing.T) {
	env := newTestEnv(t)

	cfg := defaults()
	want := filepath.Join(env.home, "Downloads")
	if cfg.DownloadDir != want {
		t.Errorf("default DownloadDir = %q, want %q", cfg.DownloadDir, want)
	}
	if cfg.Concurrency != 0 {
		t.Errorf("default Concurrency = %d, want 0", cfg.Concurrency)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("MSD_DOWNLOAD_DIR", "")
	t.Setenv("MSD_CONCURRENCY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(env.home, "Downloads")
	if cfg.DownloadDir != want {
		t.Errorf("DownloadDir = %q, want %q", cfg.DownloadDir, want)
	}
}

func TestLoad_DefaultDownloadDirUsesXDGUserDirs(t *testing.T) {
	env := newTestEnv(t)
	t.Setenv("MSD_DOWNLOAD_DIR", "")
	t.Setenv("MSD_CONCURRENCY", "")

	env.writeUserDirs(t, `XDG_DOWNLOAD_DIR="$HOME/Localized Downloads"`+"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(env.home, "Localized Downloads")
	if cfg.DownloadDir != want {
		t.Errorf("DownloadDir = %q, want %q", cfg.DownloadDir, want)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `
download_dir: $HOME/downloads
concurrency: 8
sites:
  gofile:
    account_token: file-token
`)

	t.Setenv("MSD_DOWNLOAD_DIR", "")
	t.Setenv("MSD_CONCURRENCY", "")
	t.Setenv("MSD_GOFILE_TOKEN", "")
	t.Setenv("GOFILE_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantDownloadDir := filepath.Join(env.home, "downloads")
	if cfg.DownloadDir != wantDownloadDir {
		t.Errorf("DownloadDir = %q, want %q", cfg.DownloadDir, wantDownloadDir)
	}
	if cfg.Concurrency != 8 {
		t.Errorf("Concurrency = %d, want 8", cfg.Concurrency)
	}
	if cfg.Sites.Gofile.AccountToken != "file-token" {
		t.Errorf("Gofile account token = %q, want file-token", cfg.Sites.Gofile.AccountToken)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `
download_dir: /tmp/from-file
concurrency: 4
sites:
  gofile:
    account_token: file-token
`)

	t.Setenv("MSD_DOWNLOAD_DIR", "~/from-env")
	t.Setenv("MSD_CONCURRENCY", "12")
	t.Setenv("MSD_GOFILE_TOKEN", "env-token")
	t.Setenv("GOFILE_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantDownloadDir := filepath.Join(env.home, "from-env")
	if cfg.DownloadDir != wantDownloadDir {
		t.Errorf("DownloadDir = %q, want %q", cfg.DownloadDir, wantDownloadDir)
	}
	if cfg.Concurrency != 12 {
		t.Errorf("Concurrency = %d, want 12", cfg.Concurrency)
	}
	if cfg.Sites.Gofile.AccountToken != "env-token" {
		t.Errorf("Gofile account token = %q, want env-token", cfg.Sites.Gofile.AccountToken)
	}
}

func TestLoad_GofileTokenAlternateEnv(t *testing.T) {
	newTestEnv(t)
	t.Setenv("MSD_DOWNLOAD_DIR", "")
	t.Setenv("MSD_CONCURRENCY", "")
	t.Setenv("MSD_GOFILE_TOKEN", "")
	t.Setenv("GOFILE_TOKEN", "alternate-env-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sites.Gofile.AccountToken != "alternate-env-token" {
		t.Errorf("Gofile account token = %q, want alternate-env-token", cfg.Sites.Gofile.AccountToken)
	}
}

func TestLoad_InvalidEnvConcurrency(t *testing.T) {
	newTestEnv(t)
	t.Setenv("MSD_DOWNLOAD_DIR", "")
	t.Setenv("MSD_CONCURRENCY", "notanumber")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Concurrency != 0 {
		t.Errorf("Concurrency = %d, want 0 (invalid env ignored)", cfg.Concurrency)
	}
}

func TestLoad_ConfigCanUseXDGDownloadDirVariable(t *testing.T) {
	env := newTestEnv(t)
	env.writeUserDirs(t, `XDG_DOWNLOAD_DIR="$HOME/Localized Downloads"`+"\n")
	env.writeConfig(t, `
download_dir: ${XDG_DOWNLOAD_DIR}/msd
`)

	t.Setenv("MSD_DOWNLOAD_DIR", "")
	t.Setenv("MSD_CONCURRENCY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(env.home, "Localized Downloads", "msd")
	if cfg.DownloadDir != want {
		t.Errorf("DownloadDir = %q, want %q", cfg.DownloadDir, want)
	}
}

func TestExpandPath(t *testing.T) {
	env := newTestEnv(t)

	got := ExpandPath("~/nested")
	want := filepath.Join(env.home, "nested")
	if got != want {
		t.Errorf("ExpandPath = %q, want %q", got, want)
	}
}

// A config value written with forward slashes has to come back as a native
// path. os.Expand does plain string substitution, so on Windows
// "$HOME/downloads" would otherwise yield `C:\Users\x/downloads`.
func TestExpandPath_NormalizesSeparators(t *testing.T) {
	env := newTestEnv(t)

	got := ExpandPath("$HOME/one/two")
	want := filepath.Join(env.home, "one", "two")
	if got != want {
		t.Errorf("ExpandPath = %q, want %q", got, want)
	}
}
