package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/onboarding"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestCheckOnboarding_ManifestDirectory covers the gate when the manifest is
// not in the working directory (`wave run <p> -m cfg/wave.yaml` from the
// repository root).
func TestCheckOnboarding_ManifestDirectory(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, root string)
		manifest string
		wantErr  bool
	}{
		{
			name: "manifest in subdirectory is grandfathered",
			setup: func(t *testing.T, root string) {
				require.NoError(t, os.MkdirAll(filepath.Join(root, "cfg"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(root, "cfg", "wave.yaml"), []byte("apiVersion: v1\n"), 0o644))
			},
			manifest: "cfg/wave.yaml",
		},
		{
			name: "sentinel next to manifest in subdirectory",
			setup: func(t *testing.T, root string) {
				require.NoError(t, onboarding.MarkOnboarded(filepath.Join(root, "cfg", ".agents")))
			},
			manifest: "cfg/wave.yaml",
		},
		{
			name: "sentinel in working directory",
			setup: func(t *testing.T, root string) {
				require.NoError(t, onboarding.MarkOnboarded(filepath.Join(root, ".agents")))
			},
			manifest: "cfg/wave.yaml",
		},
		{
			name:     "missing manifest and no sentinel fails",
			setup:    func(t *testing.T, root string) {},
			manifest: "cfg/wave.yaml",
			wantErr:  true,
		},
		{
			name:     "empty manifest path defaults to wave.yaml and fails when absent",
			setup:    func(t *testing.T, root string) {},
			manifest: "",
			wantErr:  true,
		},
		{
			name: "default manifest in working directory",
			setup: func(t *testing.T, root string) {
				require.NoError(t, os.WriteFile(filepath.Join(root, "wave.yaml"), []byte("apiVersion: v1\n"), 0o644))
			},
			manifest: "wave.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)
			t.Chdir(root)

			err := checkOnboarding("impl-issue", tt.manifest)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCheckOnboarding_BootstrapPipelineBypasses(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, checkOnboarding("ops-bootstrap", "cfg/wave.yaml"))
}

// TestEnterManifestDir: `wave run <p> -m cfg/wave.yaml` from the repository
// root runs the project wave init laid out in cfg/ (pipelines, contracts and
// state are resolved relative to the working directory).
func TestEnterManifestDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cfg", ".agents", "pipelines"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cfg", "wave.yaml"), []byte("apiVersion: v1\n"), 0o644))

	t.Run("manifest in subdirectory", func(t *testing.T) {
		t.Chdir(root)
		mp := "cfg/wave.yaml"
		require.NoError(t, enterManifestDir(&mp))
		wd, err := os.Getwd()
		require.NoError(t, err)
		want, err := filepath.EvalSymlinks(filepath.Join(root, "cfg"))
		require.NoError(t, err)
		got, err := filepath.EvalSymlinks(wd)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Equal(t, "wave.yaml", mp)
		require.DirExists(t, filepath.Join(".agents", "pipelines"))
	})

	t.Run("manifest in working directory is untouched", func(t *testing.T) {
		t.Chdir(filepath.Join(root, "cfg"))
		mp := "wave.yaml"
		require.NoError(t, enterManifestDir(&mp))
		require.Equal(t, "wave.yaml", mp)
	})

	t.Run("missing manifest is left for the loader to report", func(t *testing.T) {
		t.Chdir(root)
		mp := "nope/wave.yaml"
		require.NoError(t, enterManifestDir(&mp))
		require.Equal(t, "nope/wave.yaml", mp)
		wd, err := os.Getwd()
		require.NoError(t, err)
		got, _ := filepath.EvalSymlinks(wd)
		want, _ := filepath.EvalSymlinks(root)
		require.Equal(t, want, got)
	})
}

func TestEnterManifestDirForCommand(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cfg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cfg", "wave.yaml"), []byte("apiVersion: v1\n"), 0o644))

	newCmd := func(name string) (*cobra.Command, *string) {
		var mp string
		c := &cobra.Command{Use: name}
		c.Flags().StringVarP(&mp, "manifest", "m", "wave.yaml", "")
		return c, &mp
	}

	t.Run("run-like command enters the manifest directory", func(t *testing.T) {
		t.Chdir(root)
		c, mp := newCmd("list")
		require.NoError(t, c.Flags().Set("manifest", "cfg/wave.yaml"))
		require.NoError(t, EnterManifestDirForCommand(c))
		require.Equal(t, "wave.yaml", *mp)
		require.FileExists(t, "wave.yaml")
	})

	t.Run("init keeps -m as the file to create", func(t *testing.T) {
		t.Chdir(root)
		c, mp := newCmd("init")
		require.NoError(t, c.Flags().Set("manifest", "cfg/wave.yaml"))
		require.NoError(t, EnterManifestDirForCommand(c))
		require.Equal(t, "cfg/wave.yaml", *mp)
		require.NoFileExists(t, "wave.yaml")
	})
}
