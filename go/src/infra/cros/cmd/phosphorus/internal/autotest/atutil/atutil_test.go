package atutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/build/api"

	"infra/cros/cmd/phosphorus/internal/autotest"
	"infra/cros/cmd/phosphorus/internal/autotest/atutil"
	"infra/cros/internal/cmd"
)

func TestRunAutoserv(t *testing.T) {
	autotestDir, err := ioutil.TempDir("", "autotest-dir-*")
	if err != nil {
		t.Fatal(err)
	}

	testConfig := []atutil.SSPDeployConfig{
		{
			Source:      filepath.Join(autotestDir, "nonexistingdir"),
			Target:      "/etc/nonexistingdir",
			ForceCreate: true,
		},
	}

	testConfigJSON, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatal(err)
	}

	testConfigPath := filepath.Join(autotestDir, "ssp_deploy_shadow_config.json")
	if err := os.WriteFile(testConfigPath, testConfigJSON, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	resultsDir, err := ioutil.TempDir("", "results-dir-*")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	m := &atutil.MainJob{
		AutotestConfig: autotest.Config{
			AutotestDir: autotestDir,
		},
		ResultsDir: resultsDir,
	}

	j := &atutil.Test{
		Args:             "test args",
		ResultsDir:       resultsDir,
		RequireSSP:       true,
		SSPBaseImageName: "testsspname",
		Hosts:            []string{"host1", "host2"},
		PeerDuts:         []string{"peerdut1"},
	}

	if err = os.Mkdir(filepath.Join(resultsDir, "host_info_store"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	for _, dutName := range append(j.Hosts, j.PeerDuts...) {
		if err = os.WriteFile(
			filepath.Join(resultsDir, "host_info_store", fmt.Sprintf("%s.store", dutName)),
			[]byte{},
			os.ModePerm,
		); err != nil {
			t.Fatal(err)
		}
	}

	containerImageInfo := &api.ContainerImageInfo{
		Repository: &api.GcrRepository{
			Hostname: "gcr.io",
			Project:  "testproject",
		},
		Name:   "testimage",
		Digest: "abc",
		Tags:   []string{"build123"},
	}

	var w bytes.Buffer

	bytesWriter := bytes.NewBuffer([]byte{})
	stdWriter := stdcopy.NewStdWriter(bytesWriter, stdcopy.Stdout)

	_, err = stdWriter.Write([]byte("Some test stdout"))
	if err != nil {
		t.Fatalf("Failed to write to stdWriter: %q", err)
	}

	cmdRunner := &cmd.FakeCommandRunnerMulti{
		CommandRunners: []cmd.FakeCommandRunner{
			{
				ExpectedCmd: []string{
					"gcloud", "auth", "activate-service-account",
					"--key-file=/creds/service_accounts/skylab-drone.json",
				},
			},
			{
				ExpectedCmd: []string{
					"gcloud", "auth", "print-access-token",
				},
				Stdout: "abc123",
			},
			{
				ExpectedCmd: []string{
					"docker", "login", "-u", "oauth2accesstoken",
					"-p", "abc123", "gcr.io/testproject",
				},
			},
			{
				ExpectedCmd: []string{
					"docker", "run",
					"--user", "chromeos-test",
					"--network", "host",
					fmt.Sprintf("--mount=source=%s,target=%s,type=bind", resultsDir, resultsDir),
					fmt.Sprintf("--mount=source=%s,target=/etc/nonexistingdir,type=bind", filepath.Join(autotestDir, "nonexistingdir")),
					fmt.Sprintf(
						"--mount=source=%s,target=%s/host_info_store/host1.store,type=bind",
						filepath.Join(resultsDir, "host_info_store", "host1.store"),
						filepath.ToSlash(resultsDir),
					),
					fmt.Sprintf("--mount=source=%s,target=%s/host_info_store/host2.store,type=bind",
						filepath.Join(resultsDir, "host_info_store", "host2.store"),
						filepath.ToSlash(resultsDir),
					),
					fmt.Sprintf(
						"--mount=source=%s,target=%s/host_info_store/peerdut1.store,type=bind",
						filepath.Join(resultsDir, "host_info_store", "peerdut1.store"),
						filepath.ToSlash(resultsDir),
					),
					"gcr.io/testproject/testimage@abc",
					filepath.Join(autotestDir, "server", "autoserv"),
					"--args", "test args",
					"-s",
					"--host-info-subdir", "host_info_store",
					"-m", "host1,host2",
					"--lab", "True",
					"-n",
					"-ch", "peerdut1",
					"-r", resultsDir,
					"--verbose",
					"--verify_job_repo_url",
					"-p",
					"--local-only-host-info", "True",
					"",
				},
			},
		},
	}

	cmdRunner.CommandRunners[0].ExpectedCmd = []string{}

	result, err := atutil.RunAutoserv(
		ctx, m, j, &w, cmdRunner, containerImageInfo,
	)

	if err != nil {
		t.Fatalf("RunAutoserv returned error: %s", err)
	}

	if result.Exit != 0 {
		t.Errorf("Expected exit code 0, got %d", result.Exit)
	}
}

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
				Source:      filepath.FromSlash("/tmp/source"),
				ForceCreate: false,
			},
			err: fmt.Sprintf("source %s does not exist, and forceCreate is not set", filepath.FromSlash("/tmp/source")),
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
