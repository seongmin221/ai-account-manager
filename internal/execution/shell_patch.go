package execution

import (
	"sort"
	"strings"
)

type EnvPatch struct {
	Set   map[string]string
	Unset []string
}

func NewEnvPatch() EnvPatch {
	return EnvPatch{Set: make(map[string]string)}
}

func (p *EnvPatch) SetValue(name, value string) {
	if p.Set == nil {
		p.Set = make(map[string]string)
	}
	p.Set[name] = value
}

func (p *EnvPatch) UnsetValue(names ...string) {
	p.Unset = append(p.Unset, names...)
}

// RenderZsh emits only exports/unsets; it never includes credential values.
func (p EnvPatch) RenderZsh() string {
	var output strings.Builder
	keys := make([]string, 0, len(p.Set))
	for key := range p.Set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		output.WriteString("export ")
		output.WriteString(key)
		output.WriteString("=")
		output.WriteString(shellQuote(p.Set[key]))
		output.WriteString("\n")
	}
	unset := append([]string(nil), p.Unset...)
	sort.Strings(unset)
	for _, key := range unset {
		output.WriteString("unset ")
		output.WriteString(key)
		output.WriteString("\n")
	}
	return output.String()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
