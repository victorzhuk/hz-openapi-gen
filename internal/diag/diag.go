// Package diag collects generation diagnostics — blocking errors and
// non-blocking warnings — and reports them in a stable, deterministic order.
package diag

import (
	"fmt"
	"io"
	"sort"
)

type Level string

const (
	LevelError   Level = "error"
	LevelWarning Level = "warning"
)

// Diagnostic is a single error or warning carrying enough context to locate it
// in the source spec.
type Diagnostic struct {
	Level      Level
	Code       string
	Message    string
	Location   string
	Suggestion string
}

// Set accumulates diagnostics during extraction and generation.
type Set struct {
	items []Diagnostic
}

func (s *Set) Add(d Diagnostic) { s.items = append(s.items, d) }

func (s *Set) Errorf(code, location, format string, args ...any) {
	s.Add(Diagnostic{Level: LevelError, Code: code, Location: location, Message: fmt.Sprintf(format, args...)})
}

func (s *Set) Warnf(code, location, format string, args ...any) {
	s.Add(Diagnostic{Level: LevelWarning, Code: code, Location: location, Message: fmt.Sprintf(format, args...)})
}

func (s *Set) byLevel(level Level) []Diagnostic {
	var out []Diagnostic
	for _, d := range s.items {
		if d.Level == level {
			out = append(out, d)
		}
	}
	return out
}

func (s *Set) Errors() []Diagnostic   { return s.byLevel(LevelError) }
func (s *Set) Warnings() []Diagnostic { return s.byLevel(LevelWarning) }
func (s *Set) HasErrors() bool        { return len(s.Errors()) > 0 }

// Report writes every diagnostic to w, errors before warnings, each group
// sorted by location, then code, then message so output never depends on map
// iteration order.
func (s *Set) Report(w io.Writer) {
	report := append([]Diagnostic(nil), s.items...)
	sort.SliceStable(report, func(i, j int) bool {
		a, b := report[i], report[j]
		if a.Level != b.Level {
			return a.Level == LevelError
		}
		if a.Location != b.Location {
			return a.Location < b.Location
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	for _, d := range report {
		line := fmt.Sprintf("%s %s", d.Level, d.Code)
		if d.Location != "" {
			line += ": " + d.Location
		}
		line += ": " + d.Message
		if d.Suggestion != "" {
			line += " " + d.Suggestion
		}
		fmt.Fprintln(w, line)
	}
}
