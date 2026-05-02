package supportbundle

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/brain"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/devdoctor"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/installer"
	"cactus-agentlink-rescue/internal/readiness"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/session"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion int      `json:"schemaVersion"`
	ToolVersion   string   `json:"toolVersion"`
	CreatedAt     string   `json:"createdAt"`
	Status        string   `json:"status"`
	BundlePath    string   `json:"bundlePath"`
	Files         []string `json:"files"`
	Warnings      []string `json:"warnings,omitempty"`
}

func Create(ctx context.Context, runner command.Runner, home, output string, catalog installer.Catalog) Report {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if output == "" {
		name := "agentlink-support-bundle-" + time.Now().Format("20060102-150405") + ".zip"
		output = filepath.Join(system.UserSupportBundleDir(home), name)
	}
	rep := Report{SchemaVersion: 1, ToolVersion: system.Version, CreatedAt: time.Now().Format(time.RFC3339), Status: "created", BundlePath: output}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	tmp, err := os.MkdirTemp("", "agentlink-support-bundle-*")
	if err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	defer os.RemoveAll(tmp)

	writeJSON(&rep, tmp, "facts.json", facts.Collect(ctx, runner, home, true))
	writeJSON(&rep, tmp, "brain-doctor.json", brain.Doctor(ctx, runner, home))
	writeJSON(&rep, tmp, "installer-doctor.json", installer.Doctor(ctx, runner, catalog, system.Version))
	writeJSON(&rep, tmp, "dev-essentials.json", devdoctor.Run(ctx, runner, system.Version))
	writeJSON(&rep, tmp, "offline-readiness.json", readiness.Run(ctx, runner, home, catalog))
	if sess, err := session.NewStore(home).Latest(); err == nil {
		writeJSON(&rep, tmp, "latest-session.json", sess)
		writeText(&rep, tmp, "latest-human-report.txt", session.HumanReport(sess))
		writeText(&rep, tmp, "latest-agent-dispatch.json", session.AgentDispatch(sess))
	} else {
		rep.Warnings = append(rep.Warnings, "latest session unavailable: "+safety.RedactSensitive(err.Error()))
	}
	writeText(&rep, tmp, "README.txt", "Cactus AgentLink Rescue support bundle\nAll files are redacted before writing. No private keys, browser cookies, shell history, or Wi-Fi passwords are collected.\n")
	if err := zipDir(tmp, output); err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	return rep
}

func writeJSON(rep *Report, dir, name string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		rep.Warnings = append(rep.Warnings, name+": "+err.Error())
		return
	}
	writeText(rep, dir, name, string(data))
}

func writeText(rep *Report, dir, name, text string) {
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		rep.Warnings = append(rep.Warnings, name+": "+err.Error())
		return
	}
	if err := os.WriteFile(path, []byte(safety.RedactSensitive(text)), 0644); err != nil {
		rep.Warnings = append(rep.Warnings, name+": "+err.Error())
		return
	}
	rep.Files = append(rep.Files, name)
}

func zipDir(src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if strings.Contains(rel, "..") {
			return fmt.Errorf("invalid bundle path: %s", rel)
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
}
