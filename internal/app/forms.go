package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/beads"
)

func (m *Model) updateForm(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch m.formFocus {
	case 0:
		var cmd tea.Cmd
		m.formTitle, cmd = m.formTitle.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		var cmd tea.Cmd
		m.formDesc, cmd = m.formDesc.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		var cmd tea.Cmd
		m.formNotes, cmd = m.formNotes.Update(msg)
		cmds = append(cmds, cmd)
	case 3:
		var cmd tea.Cmd
		m.formDesign, cmd = m.formDesign.Update(msg)
		cmds = append(cmds, cmd)
	case 4:
		var cmd tea.Cmd
		m.formAcceptance, cmd = m.formAcceptance.Update(msg)
		cmds = append(cmds, cmd)
	case 5:
		// Priority selection
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "left", "h":
				if m.formPriority > 0 {
					m.formPriority--
				}
			case "right", "l":
				if m.formPriority < 4 {
					m.formPriority++
				}
			}
		}
	case 6:
		// Type selection
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			types := []string{"task", "bug", "feature", "epic", "chore"}
			idx := 0
			for i, t := range types {
				if t == m.formType {
					idx = i
					break
				}
			}
			switch keyMsg.String() {
			case "left", "h":
				idx = (idx - 1 + len(types)) % len(types)
			case "right", "l":
				idx = (idx + 1) % len(types)
			}
			m.formType = types[idx]
		}
	}

	return tea.Batch(cmds...)
}

func (m *Model) resetForm() {
	m.formTitle.SetValue("")
	m.formDesc.SetValue("")
	m.formNotes.SetValue("")
	m.formDesign.SetValue("")
	m.formAcceptance.SetValue("")
	m.formPriority = 2
	m.formType = "feature"
	m.formFocus = 0
	m.updateFormFocus()
}

func (m *Model) updateFormFocus() {
	m.formTitle.Blur()
	m.formDesc.Blur()
	m.formNotes.Blur()
	m.formDesign.Blur()
	m.formAcceptance.Blur()
	switch m.formFocus {
	case 0:
		m.formTitle.Focus()
	case 1:
		m.formDesc.Focus()
	case 2:
		m.formNotes.Focus()
	case 3:
		m.formDesign.Focus()
	case 4:
		m.formAcceptance.Focus()
	}
}

func (m *Model) submitForm() tea.Cmd {
	title := strings.TrimSpace(m.formTitle.Value())
	if title == "" {
		m.err = fmt.Errorf("title is required")
		return nil
	}

	if m.editing {
		return func() tea.Msg {
			err := m.client.Update(m.editingID, beads.UpdateOptions{
				Title:    title,
				Priority: &m.formPriority,
			})
			return taskUpdatedMsg{err: err}
		}
	}

	return func() tea.Msg {
		task, err := m.client.Create(beads.CreateOptions{
			Title:              title,
			Description:        m.formDesc.Value(),
			Notes:              m.formNotes.Value(),
			Design:             m.formDesign.Value(),
			AcceptanceCriteria: m.formAcceptance.Value(),
			Type:               m.formType,
			Priority:           m.formPriority,
		})
		return taskCreatedMsg{task: task, err: err}
	}
}
