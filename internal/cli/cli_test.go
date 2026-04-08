package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute_help(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Execute(nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Execute() exitCode = %d", exitCode)
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}

	if !strings.Contains(stdout.String(), "HomeStock CLI") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExecute_configShow(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Execute([]string{"config", "show", "--config", ""}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Execute() exitCode = %d, stderr = %q", exitCode, stderr.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}

	body := stdout.String()
	if !strings.Contains(body, `"port": "8888"`) {
		t.Fatalf("stdout = %q", body)
	}
	if !strings.Contains(body, `"driver": "sqlite"`) {
		t.Fatalf("stdout = %q", body)
	}
}

func TestExecute_unknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Execute([]string{"missing"}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("Execute() exitCode = %d", exitCode)
	}

	if !strings.Contains(stderr.String(), `unknown command "missing"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecute_configShowRejectsPositionalArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Execute([]string{"config", "show", "extra"}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("Execute() exitCode = %d", exitCode)
	}

	if !strings.Contains(stderr.String(), "does not accept positional arguments") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
