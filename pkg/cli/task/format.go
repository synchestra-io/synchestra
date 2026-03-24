package task

// Features implemented: cli/task
// Features depended on:  state-store

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/synchestra-io/synchestra/pkg/state"
	"gopkg.in/yaml.v3"
)

// taskOutput is the serialisable view of a task for CLI output.
type taskOutput struct {
	Path           string `json:"path" yaml:"path"`
	Status         string `json:"status" yaml:"status"`
	Title          string `json:"title" yaml:"title"`
	Run            string `json:"run,omitempty" yaml:"run,omitempty"`
	Model          string `json:"model,omitempty" yaml:"model,omitempty"`
	Requester      string `json:"requester,omitempty" yaml:"requester,omitempty"`
	DependsOn      string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Reason         string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Summary        string `json:"summary,omitempty" yaml:"summary,omitempty"`
	ClaimedAt      string `json:"claimed_at,omitempty" yaml:"claimed_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	AbortRequested string `json:"abort_requested,omitempty" yaml:"abort_requested,omitempty"`
}

func toTaskOutput(t state.Task) taskOutput {
	o := taskOutput{
		Path:      t.Slug,
		Status:    string(t.Status),
		Title:     t.Title,
		Run:       t.Run,
		Model:     t.Model,
		Requester: t.Requester,
		Reason:    t.Reason,
		Summary:   t.Summary,
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if len(t.DependsOn) > 0 {
		o.DependsOn = strings.Join(t.DependsOn, ",")
	}
	if t.ClaimedAt != nil {
		o.ClaimedAt = t.ClaimedAt.Format("2006-01-02T15:04:05Z")
	}
	return o
}

func writeTaskList(w io.Writer, format string, tasks []state.Task) error {
	outputs := make([]taskOutput, len(tasks))
	for i, t := range tasks {
		outputs[i] = toTaskOutput(t)
	}
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(outputs)
	case "yaml":
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		err := enc.Encode(outputs)
		if err != nil {
			return err
		}
		return enc.Close()
	case "csv":
		return writeCSV(w, outputs)
	case "md":
		return writeMarkdown(w, outputs)
	default:
		return writeYAML(w, outputs)
	}
}

func writeTask(w io.Writer, format string, t state.Task) error {
	o := toTaskOutput(t)
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(o)
	case "yaml":
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		err := enc.Encode(o)
		if err != nil {
			return err
		}
		return enc.Close()
	default:
		return writeYAML(w, o)
	}
}

func writeYAML(w io.Writer, v any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	err := enc.Encode(v)
	if err != nil {
		return err
	}
	return enc.Close()
}

func writeCSV(w io.Writer, tasks []taskOutput) error {
	cw := csv.NewWriter(w)
	header := []string{"path", "status", "title", "run", "model", "requester", "depends_on", "claimed_at", "updated_at"}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, t := range tasks {
		row := []string{t.Path, t.Status, t.Title, t.Run, t.Model, t.Requester, t.DependsOn, t.ClaimedAt, t.UpdatedAt}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeMarkdown(w io.Writer, tasks []taskOutput) error {
	_, err := fmt.Fprintln(w, "| Path | Status | Title | Run | Model | Updated |")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, "|---|---|---|---|---|---|")
	if err != nil {
		return err
	}
	for _, t := range tasks {
		_, err = fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s |\n", t.Path, t.Status, t.Title, t.Run, t.Model, t.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}
