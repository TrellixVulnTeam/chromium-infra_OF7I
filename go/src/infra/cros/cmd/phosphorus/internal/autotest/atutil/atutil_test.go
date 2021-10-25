package atutil_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/google/go-cmp/cmp"

	"infra/cros/cmd/phosphorus/internal/autotest"
	"infra/cros/cmd/phosphorus/internal/autotest/atutil"
)

func TestParseSSPDeployShadowConfig(t *testing.T) {
	ctx := context.Background()
	testDir := t.TempDir()

	filesToCreate := []string{
		filepath.Join(testDir, "file1"), filepath.Join(testDir, "relfile1.txt"),
	}

	dirsToCreate := []string{
		filepath.Join(testDir, "existingdir"),
	}

	for _, f := range filesToCreate {
		fd, err := os.Create(f)
		if err != nil {
			t.Fatal(err)
		}

		if err := fd.Close(); err != nil {
			t.Fatal(err)
		}
	}

	for _, d := range dirsToCreate {
		if err := os.MkdirAll(d, os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}

	testConfig := []atutil.SSPDeployConfig{
		{
			Source: filepath.Join(testDir, "file1"),
			Target: "/tmp/file1",
			Append: true,
		},
		{
			Source: "relfile1.txt",
			Target: "/tmp/a/relfile1.txt",
			Append: true,
		},
		{
			Source:      filepath.Join(testDir, "existingdir"),
			Target:      "/etc/existingdir",
			ForceCreate: false,
			Readonly:    true,
		},
		{
			Source:      filepath.Join(testDir, "nonexistingdir"),
			Target:      "/etc/nonexistingdir",
			ForceCreate: true,
		},
	}

	testConfigJSON, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatal(err)
	}

	testConfigPath := filepath.Join(testDir, "ssp_deploy_shadow_config.json")
	if err := os.WriteFile(testConfigPath, testConfigJSON, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	mounts, err := atutil.ParseSSPDeployShadowConfig(ctx, autotest.Config{AutotestDir: testDir}, testConfigPath)
	if err != nil {
		t.Fatalf("ParseSSPDeployShadowConfig returned error: %s", err)
	}

	expectedMounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: filepath.Join(testDir, "file1"),
			Target: "/tmp/file1",
		},
		{
			Type:   mount.TypeBind,
			Source: filepath.Join(testDir, "relfile1.txt"),
			Target: "/tmp/a/relfile1.txt",
		},
		{
			Type:     mount.TypeBind,
			Source:   filepath.Join(testDir, "existingdir"),
			Target:   "/etc/existingdir",
			ReadOnly: true,
		},
		{
			Type:   mount.TypeBind,
			Source: filepath.Join(testDir, "nonexistingdir"),
			Target: "/etc/nonexistingdir",
		},
	}

	if diff := cmp.Diff(expectedMounts, mounts); diff != "" {
		t.Errorf("ParseSSPDeployShadowConfig() returned unexpected results (-want +got):\n%s", diff)
	}

	_, err = os.Stat(filepath.Join(testDir, "nonexistingdir"))
	if err != nil {
		t.Errorf("Expected nonexisting source path to be created, Stat failed: %s", err)
	}
}

func TestParseSSPDeployShadowConfigErrors(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name   string
		config atutil.SSPDeployConfig
		err    string
	}{
		{
			name: "non-absolute target",
			config: atutil.SSPDeployConfig{
				Target: "nonabsolute/txt1",
			},
			err: "target path must be absolute (nonabsolute/txt1)",
		},
		{
			name: "source path with ~",
			config: atutil.SSPDeployConfig{
				Target: "/tmp/target",
				Source: "~/txt1",
				Append: true,
			},
			err: "source path may not have '~' (~/txt1)",
		},
		{
			name: "source non-existent",
			config: atutil.SSPDeployConfig{
				Target:      "/tmp/target",
				Source:      "/tmp/source",
				ForceCreate: false,
			},
			err: "source /tmp/source does not exist, and forceCreate is not set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bytes, err := json.Marshal([]atutil.SSPDeployConfig{tc.config})
			if err != nil {
				t.Fatal(err)
			}

			configFile := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(configFile, bytes, os.ModePerm); err != nil {
				t.Fatal(err)
			}

			_, err = atutil.ParseSSPDeployShadowConfig(ctx, autotest.Config{}, configFile)
			if err == nil {
				t.Error("expected ParseSSPDeployShadowConfig to return error")
			}

			if err.Error() != tc.err {
				t.Errorf("expected error %q, got %q", tc.err, err.Error())
			}
		})
	}
}
