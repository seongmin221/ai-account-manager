package execution

import (
	"strings"
	"testing"
)

func TestRenderZshQuotesValuesAndSortsWithoutSecrets(t *testing.T) {
	patch := NewEnvPatch()
	patch.SetValue("CODEX_HOME", "/tmp/user's-codex")
	patch.SetValue("GH_HOST", "github.com")
	patch.UnsetValue("GITHUB_TOKEN", "GH_TOKEN")
	output := patch.RenderZsh()
	want := "export CODEX_HOME='/tmp/user'\\''s-codex'\nexport GH_HOST='github.com'\nunset GH_TOKEN\nunset GITHUB_TOKEN\n"
	if output != want {
		t.Fatalf("RenderZsh() = %q, want %q", output, want)
	}
	if strings.Contains(output, "secret") {
		t.Fatal("shell patch contains secret material")
	}
}
