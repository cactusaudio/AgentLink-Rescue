package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"cactus-agentlink-rescue/internal/system"
)

type Store struct {
	Home string
}

func NewStore(home string) Store {
	return Store{Home: home}
}

func (s Store) Dir() string {
	return system.UserSessionDir(s.Home)
}

func (s Store) SessionDir(id string) string {
	return filepath.Join(s.Dir(), id)
}

func (s Store) Save(sess *Session) error {
	dir := s.SessionDir(sess.SessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if sess.HumanReportPath == "" {
		sess.HumanReportPath = filepath.Join(dir, "human-report.txt")
	}
	if sess.AgentDispatchPath == "" {
		sess.AgentDispatchPath = filepath.Join(dir, "agent-dispatch.json")
	}
	if err := os.WriteFile(sess.HumanReportPath, []byte(HumanReport(*sess)), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(sess.AgentDispatchPath, []byte(AgentDispatch(*sess)), 0644); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "session.json"), data, 0644); err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(s.Dir(), "latest"), []byte(sess.SessionID), 0644)
	return nil
}

func (s Store) Latest() (Session, error) {
	var sess Session
	if data, err := os.ReadFile(filepath.Join(s.Dir(), "latest")); err == nil {
		return s.Load(string(bytesTrimSpace(data)))
	}
	entries, err := os.ReadDir(s.Dir())
	if err != nil {
		return sess, err
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return sess, os.ErrNotExist
	}
	return s.Load(ids[len(ids)-1])
}

func (s Store) Load(id string) (Session, error) {
	var sess Session
	data, err := os.ReadFile(filepath.Join(s.SessionDir(id), "session.json"))
	if err != nil {
		return sess, err
	}
	err = json.Unmarshal(data, &sess)
	return sess, err
}

func bytesTrimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\n' || b[0] == '\t' || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 {
		c := b[len(b)-1]
		if c != ' ' && c != '\n' && c != '\t' && c != '\r' {
			break
		}
		b = b[:len(b)-1]
	}
	return b
}
