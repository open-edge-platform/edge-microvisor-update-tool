package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommand(cmd *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()
	return buf.String(), err
}

func TestWriteCmd(t *testing.T) {
	output, err := executeCommand(writeCmd, "path/to/image", "checksum123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "" // Replace with the actual expected output
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestApplyCmd(t *testing.T) {
	output, err := executeCommand(applyCmd)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "" // Replace with the actual expected output
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestCommitCmd(t *testing.T) {
	output, err := executeCommand(commitCmd)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "" // Replace with the actual expected output
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestRollbackCmd(t *testing.T) {
	output, err := executeCommand(rollbackCmd)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "" // Replace with the actual expected output
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestDisplayCmd(t *testing.T) {
	output, err := executeCommand(displayCmd)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := ""
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}
