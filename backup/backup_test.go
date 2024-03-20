package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/require"
	"github.com/tomruk/kopyaship/backup/provider"
	"github.com/tomruk/kopyaship/config"
	"go.uber.org/zap"
)

func TestGitignoreToRestic(t *testing.T) {
	const (
		basePathRelative = "../tmp"
		repoPathRelative = "../tmp/documents-repo"
		password         = "1"
		extraArgs        = "-H test"
	)

	basePath, err := filepath.Abs(basePathRelative)
	require.NoError(t, err)

	repoPath, err := filepath.Abs(repoPathRelative)
	require.NoError(t, err)

	os.RemoveAll("../tmp/documents")
	os.RemoveAll(repoPath)

	mustCreateDir("../tmp/documents")
	mustCreateFile("../tmp/documents/1", "")
	mustCreateDir("../tmp/documents/2")

	mustCreateFile("../tmp/documents/3", "")
	mustCreateDir("../tmp/documents/4")
	mustCreateFile("../tmp/documents/4/3", "")
	mustCreateFile("../tmp/documents/4/1", "")

	mustCreateFile("../tmp/documents/5", "")
	mustCreateFile("../tmp/documents/4/5", "")

	mustCreateFile("../tmp/documents/6", "")
	mustCreateFile("../tmp/documents/4/6", "")

	// https://git-scm.com/docs/gitignore#_pattern_format
	mustCreateFile("../tmp/documents/.gitignore", `
#1
#2

3
/5
/4/6

`)

	mustHave := []string{
		"/documents",
		"/documents/.gitignore",
		"/documents/1",
		"/documents/2",
		"/documents/4",
		"/documents/4/1",
		"/documents/4/5",
		"/documents/6",
	}
	mustNotHave := []string{
		"",
		"/documents/3",
		"/documents/4/3",
		"/documents/5",
		"/documents/4/6",
	}

	restic := provider.NewRestic(context.Background(), repoPath, extraArgs, password, false, zap.NewNop())
	err = restic.Init()
	require.NoError(t, err)

	configBackups := &config.Backups{
		Run: []*config.Backup{
			{
				Name:            "test-gitignore-edge-cases",
				IfileGeneration: true,
				Restic: &config.Restic{
					Repo:      repoPath,
					ExtraArgs: extraArgs,
					Password:  password,
				},
				Base: basePath,
				Paths: []string{
					"documents",
				},
			},
		},
	}

	backups, err := FromConfig(context.Background(), configBackups, ".", zap.NewNop(), false)
	require.NoError(t, err)
	backup := backups["test-gitignore-edge-cases"]

	err = backup.Do()
	require.NoError(t, err)

	output := bytes.Buffer{}
	err = testRunRestic(repoPath, "ls -q latest", extraArgs, password, &output)
	require.NoError(t, err)

	outputString := output.String()
	outputString = strings.TrimSuffix(outputString, "\n")
	lines := strings.Split(outputString, "\n")
	require.Greater(t, len(lines), 0)

	filesDirs := make(map[string]struct{})
	for i, line := range lines {
		if line == "" {
			t.Fatal("line is empty")
		}
		line = filepath.ToSlash(line)
		if len(line) <= len(basePath) {
			continue
		}
		line = strings.TrimPrefix(line, basePath)

		fmt.Printf("line %d: '%s'\n", i, line)
		filesDirs[line] = struct{}{}
	}

	for _, have := range mustHave {
		_, ok := filesDirs[have]
		if !ok {
			t.Fatalf("not found: %s", have)
		}
	}
	for _, notHave := range mustNotHave {
		_, ok := filesDirs[notHave]
		if ok {
			t.Fatalf("should not be found: %s", notHave)
		}
	}
}

func testRunRestic(repoPath, command, extraArgs string, password string, wr io.Writer) error {
	parser := shellwords.NewParser()
	parser.ParseBacktick = true
	parser.ParseEnv = true

	command = fmt.Sprintf("restic -r '%s' %s", repoPath, command)
	if extraArgs != "" {
		command += " " + extraArgs
	}

	fmt.Printf("Running: %s\n", command)
	if password != "" {
		os.Setenv("RESTIC_PASSWORD", password)
		defer os.Unsetenv("RESTIC_PASSWORD")
	}

	w, err := parser.Parse(command)
	if err != nil {
		return err
	}
	if len(w) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(w[0], w[1:]...)
	stdout := io.MultiWriter(os.Stdout, wr)
	stderr := io.MultiWriter(os.Stderr, wr)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func mustCreateFile(path string, content string) {
	os.MkdirAll(filepath.Dir(path), 0755)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		panic(err)
	}
}

func mustCreateDir(path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		panic(err)
	}
}
