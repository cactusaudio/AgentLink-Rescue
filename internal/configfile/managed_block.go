package configfile

import (
	"os"
	"path/filepath"
	"strings"
)

type PatchResult struct {
	Path    string `json:"path"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
}

func StartMarker(marker string) string {
	return "# >>> AGENTLINK_" + marker + " >>>"
}

func EndMarker(marker string) string {
	return "# <<< AGENTLINK_" + marker + " <<<"
}

func HasManagedBlock(path, marker string) bool {
	return CountManagedBlocks(path, marker) > 0
}

func CountManagedBlocks(path, marker string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(data), StartMarker(marker))
}

func AppendManagedBlockIfMissing(path, marker, content string) PatchResult {
	if CountManagedBlocks(path, marker) > 0 {
		return PatchResult{Path: path, Changed: false, Message: "managed block already present"}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return PatchResult{Path: path, Message: err.Error()}
	}
	existing, _ := os.ReadFile(path)
	text := string(existing)
	var b strings.Builder
	b.WriteString(text)
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		b.WriteString("\n")
	}
	if len(text) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(StartMarker(marker))
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(content, "\n"))
	b.WriteString("\n")
	b.WriteString(EndMarker(marker))
	b.WriteString("\n")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return PatchResult{Path: path, Message: err.Error()}
	}
	return PatchResult{Path: path, Changed: true, Message: "managed block appended"}
}

func RemoveManagedBlock(path, marker string) PatchResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return PatchResult{Path: path, Message: err.Error()}
	}
	start := StartMarker(marker)
	end := EndMarker(marker)
	text := string(data)
	startIdx := strings.Index(text, start)
	if startIdx < 0 {
		return PatchResult{Path: path, Changed: false, Message: "managed block absent"}
	}
	endIdx := strings.Index(text[startIdx:], end)
	if endIdx < 0 {
		return PatchResult{Path: path, Message: "managed block end marker missing"}
	}
	endIdx = startIdx + endIdx + len(end)
	for endIdx < len(text) && (text[endIdx] == '\n' || text[endIdx] == '\r') {
		endIdx++
	}
	updated := strings.TrimRight(text[:startIdx], "\n") + "\n" + text[endIdx:]
	if strings.TrimSpace(updated) == "" {
		updated = ""
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return PatchResult{Path: path, Message: err.Error()}
	}
	return PatchResult{Path: path, Changed: true, Message: "managed block removed"}
}
