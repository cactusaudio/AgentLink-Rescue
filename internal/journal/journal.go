package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/system"
)

const (
	StateStarted         = "started"
	StateCheckpointed    = "checkpointed"
	StateMutating        = "mutating"
	StateVerifying       = "verifying"
	StateSucceeded       = "succeeded"
	StateFailed          = "failed"
	StateRolledBack      = "rolled_back"
	StatePartialRollback = "partial_rollback"
	StateAbandoned       = "abandoned"
)

type Transaction struct {
	SchemaVersion     int      `json:"schemaVersion"`
	TransactionID     string   `json:"transactionID"`
	StartedAt         string   `json:"startedAt"`
	EndedAt           string   `json:"endedAt,omitempty"`
	State             string   `json:"state"`
	Target            string   `json:"target,omitempty"`
	Recipe            string   `json:"recipe,omitempty"`
	RestorePoint      string   `json:"restorePoint,omitempty"`
	PackageRoot       string   `json:"packageRoot,omitempty"`
	RequiresAdmin     bool     `json:"requiresAdmin"`
	WorsenedSignals   []string `json:"worsenedSignals,omitempty"`
	RollbackAvailable bool     `json:"rollbackAvailable"`
}

type MutationEntry struct {
	Timestamp string `json:"timestamp"`
	ActionID  string `json:"actionID"`
	Stage     string `json:"stage,omitempty"`
	Command   string `json:"command,omitempty"`
	State     string `json:"state,omitempty"`
}

type StartOptions struct {
	Home          string
	Target        string
	Recipe        string
	PackageRoot   string
	RequiresAdmin bool
}

type RecoveryReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	ToolVersion   string        `json:"toolVersion"`
	Status        string        `json:"status"`
	Transactions  []Transaction `json:"transactions,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
	NextAction    string        `json:"nextAction,omitempty"`
}

type Manager struct {
	Home string
}

func New(home string) Manager {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return Manager{Home: home}
}

func (m Manager) Dir() string {
	return filepath.Join(m.Home, "Library", "Application Support", system.AppName, "journal")
}

func (m Manager) Start(opts StartOptions) (*Transaction, error) {
	if opts.Home != "" {
		m.Home = opts.Home
	}
	id := time.Now().Format("20060102-150405.000000000")
	tx := &Transaction{
		SchemaVersion: 1,
		TransactionID: id,
		StartedAt:     time.Now().Format(time.RFC3339),
		State:         StateStarted,
		Target:        opts.Target,
		Recipe:        opts.Recipe,
		PackageRoot:   opts.PackageRoot,
		RequiresAdmin: opts.RequiresAdmin,
	}
	if err := os.MkdirAll(m.txDir(id), 0700); err != nil {
		return nil, err
	}
	return tx, m.Save(tx)
}

func (m Manager) Save(tx *Transaction) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.txDir(tx.TransactionID), 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.txDir(tx.TransactionID), "transaction.json"), data, 0600)
}

func (m Manager) WriteJSON(tx *Transaction, name string, v any) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	if strings.Contains(name, "..") || filepath.IsAbs(name) {
		return errors.New("invalid journal artifact name")
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.txDir(tx.TransactionID), name), data, 0600)
}

func (m Manager) AppendMutation(tx *Transaction, entry MutationEntry) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(m.txDir(tx.TransactionID), "mutation-log.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func (m Manager) UpdateState(tx *Transaction, state string) error {
	tx.State = state
	if isTerminal(state) {
		tx.EndedAt = time.Now().Format(time.RFC3339)
	}
	return m.Save(tx)
}

func (m Manager) Load(id string) (Transaction, error) {
	id = filepath.Base(id)
	data, err := os.ReadFile(filepath.Join(m.txDir(id), "transaction.json"))
	if err != nil {
		return Transaction{}, err
	}
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return Transaction{}, err
	}
	return tx, nil
}

func (m Manager) List() ([]Transaction, error) {
	entries, err := os.ReadDir(m.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Transaction
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tx, err := m.Load(entry.Name())
		if err == nil {
			out = append(out, tx)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransactionID < out[j].TransactionID })
	return out, nil
}

func (m Manager) Incomplete() ([]Transaction, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}
	var out []Transaction
	for _, tx := range all {
		if !isTerminal(tx.State) {
			out = append(out, tx)
		}
	}
	return out, nil
}

func (m Manager) HasMutations(id string) bool {
	path := filepath.Join(m.txDir(filepath.Base(id)), "mutation-log.jsonl")
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			return true
		}
	}
	return false
}

func (m Manager) MarkAbandoned(id string) (Transaction, error) {
	tx, err := m.Load(id)
	if err != nil {
		return Transaction{}, err
	}
	tx.State = StateAbandoned
	tx.EndedAt = time.Now().Format(time.RFC3339)
	return tx, m.Save(&tx)
}

func (m Manager) Recover(dry bool) RecoveryReport {
	rep := RecoveryReport{SchemaVersion: 1, ToolVersion: system.Version, Status: "ok"}
	incomplete, err := m.Incomplete()
	if err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	if len(incomplete) == 0 {
		rep.Status = "none"
		return rep
	}
	rep.Transactions = incomplete
	rep.Status = "incomplete_transactions_found"
	for _, tx := range incomplete {
		if !m.HasMutations(tx.TransactionID) {
			if dry {
				rep.Warnings = append(rep.Warnings, "transaction has no mutation log and can be marked abandoned: "+tx.TransactionID)
				continue
			}
			if _, err := m.MarkAbandoned(tx.TransactionID); err != nil {
				rep.Status = "failed"
				rep.Warnings = append(rep.Warnings, err.Error())
			} else {
				rep.Status = "abandoned"
			}
			continue
		}
		if tx.RollbackAvailable && tx.RestorePoint != "" {
			rep.NextAction = "agentlink rollback --id " + tx.RestorePoint
			rep.Warnings = append(rep.Warnings, "transaction needs rollback: "+tx.TransactionID)
		} else {
			rep.Warnings = append(rep.Warnings, "transaction mutated state but has no rollback metadata: "+tx.TransactionID)
		}
	}
	return rep
}

func (m Manager) txDir(id string) string {
	return filepath.Join(m.Dir(), filepath.Base(id))
}

func isTerminal(state string) bool {
	switch state {
	case StateSucceeded, StateFailed, StateRolledBack, StatePartialRollback, StateAbandoned:
		return true
	default:
		return false
	}
}
