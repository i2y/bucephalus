package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTool(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	tool := MustRead()

	t.Run("read all", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"path": "`+testFile+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(ReadOutput)
		if out.Lines != 5 {
			t.Errorf("expected 5 lines, got %d", out.Lines)
		}
		if out.Truncated {
			t.Error("expected not truncated")
		}
	})

	t.Run("read with limit", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"path": "`+testFile+`", "limit": 2}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(ReadOutput)
		if out.Lines != 2 {
			t.Errorf("expected 2 lines, got %d", out.Lines)
		}
		if !out.Truncated {
			t.Error("expected truncated")
		}
	})

	t.Run("read with offset", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"path": "`+testFile+`", "offset": 2, "limit": 2}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(ReadOutput)
		if out.Lines != 2 {
			t.Errorf("expected 2 lines, got %d", out.Lines)
		}
		if !strings.Contains(out.Content, "line3") {
			t.Errorf("expected content to contain 'line3', got %s", out.Content)
		}
	})
}

func TestWriteTool(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	tool := MustWrite()

	t.Run("write file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "write_test.txt")
		content := "hello world"
		result, err := tool.Execute(ctx, []byte(`{"path": "`+testFile+`", "content": "`+content+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(WriteOutput)
		if !out.Success {
			t.Error("expected success")
		}
		if out.Bytes != len(content) {
			t.Errorf("expected %d bytes, got %d", len(content), out.Bytes)
		}

		// Verify file content
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}
	})

	t.Run("write with nested directory", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "nested", "dir", "test.txt")
		result, err := tool.Execute(ctx, []byte(`{"path": "`+testFile+`", "content": "test"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(WriteOutput)
		if !out.Success {
			t.Error("expected success")
		}
	})
}

func TestGlobTool(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "test1.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub", "test3.go"), []byte(""), 0644)

	ctx := context.Background()
	tool := MustGlob()

	t.Run("simple glob", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"pattern": "*.go", "path": "`+tmpDir+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(GlobOutput)
		if out.Count != 2 {
			t.Errorf("expected 2 files, got %d: %v", out.Count, out.Files)
		}
	})

	t.Run("recursive glob", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"pattern": "**/*.go", "path": "`+tmpDir+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(GlobOutput)
		if out.Count != 3 {
			t.Errorf("expected 3 files, got %d: %v", out.Count, out.Files)
		}
	})
}

func TestGrepTool(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "test1.go"), []byte("func main() {}\nfunc helper() {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.go"), []byte("type Foo struct{}\nfunc Bar() {}"), 0644)

	ctx := context.Background()
	tool := MustGrep()

	t.Run("search pattern", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"pattern": "func", "path": "`+tmpDir+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(GrepOutput)
		if out.Count != 3 {
			t.Errorf("expected 3 matches, got %d", out.Count)
		}
	})

	t.Run("search with glob filter", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"pattern": "func", "path": "`+tmpDir+`", "glob": "test1.go"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(GrepOutput)
		if out.Count != 2 {
			t.Errorf("expected 2 matches, got %d", out.Count)
		}
	})

	t.Run("max matches", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"pattern": "func", "path": "`+tmpDir+`", "max_matches": 1}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(GrepOutput)
		if out.Count != 1 {
			t.Errorf("expected 1 match, got %d", out.Count)
		}
	})
}

func TestBashTool(t *testing.T) {
	ctx := context.Background()
	tool := MustBash()

	t.Run("simple command", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"command": "echo hello"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(BashOutput)
		if out.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", out.ExitCode)
		}
		if strings.TrimSpace(out.Stdout) != "hello" {
			t.Errorf("expected 'hello', got %q", out.Stdout)
		}
	})

	t.Run("failing command", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"command": "exit 1"}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(BashOutput)
		if out.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", out.ExitCode)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		result, err := tool.Execute(ctx, []byte(`{"command": "sleep 5", "timeout": 1}`))
		if err != nil {
			t.Fatal(err)
		}
		out := result.(BashOutput)
		if out.ExitCode != -1 {
			t.Errorf("expected exit code -1 for timeout, got %d", out.ExitCode)
		}
	})
}

func TestRegistryFunctions(t *testing.T) {
	t.Run("AllTools", func(t *testing.T) {
		tools := AllTools()
		if len(tools) != 8 {
			t.Errorf("expected 8 tools, got %d", len(tools))
		}
	})

	t.Run("FileTools", func(t *testing.T) {
		tools := FileTools()
		if len(tools) != 4 {
			t.Errorf("expected 4 tools, got %d", len(tools))
		}
	})

	t.Run("WebTools", func(t *testing.T) {
		tools := WebTools()
		if len(tools) != 3 {
			t.Errorf("expected 3 tools, got %d", len(tools))
		}
	})

	t.Run("KnowledgeTools", func(t *testing.T) {
		tools := KnowledgeTools()
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	t.Run("ReadOnlyTools", func(t *testing.T) {
		tools := ReadOnlyTools()
		if len(tools) != 6 {
			t.Errorf("expected 6 tools, got %d", len(tools))
		}
	})

	t.Run("SystemTools", func(t *testing.T) {
		tools := SystemTools()
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})
}

func TestToolMetadata(t *testing.T) {
	tools := AllTools()

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			if tool.Name() == "" {
				t.Error("tool name should not be empty")
			}
			if tool.Description() == "" {
				t.Error("tool description should not be empty")
			}
			if tool.Parameters() == nil {
				t.Error("tool parameters should not be nil")
			}
		})
	}
}
