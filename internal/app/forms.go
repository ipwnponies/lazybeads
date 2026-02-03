package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

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
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			bumpTextareaHeightForNewline(&m.formDesc)
		}
		var cmd tea.Cmd
		m.formDesc, cmd = m.formDesc.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			bumpTextareaHeightForNewline(&m.formNotes)
		}
		var cmd tea.Cmd
		m.formNotes, cmd = m.formNotes.Update(msg)
		cmds = append(cmds, cmd)
	case 3:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			bumpTextareaHeightForNewline(&m.formDesign)
		}
		var cmd tea.Cmd
		m.formDesign, cmd = m.formDesign.Update(msg)
		cmds = append(cmds, cmd)
	case 4:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			bumpTextareaHeightForNewline(&m.formAcceptance)
		}
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

	m.updateFormTextAreaHeights()
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
	m.updateFormTextAreaHeights()
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

func (m *Model) applyEditorContentToForm(field editorField, content string) {
	focus, ok := formFocusForEditorField(field)
	if !ok {
		m.err = fmt.Errorf("unsupported form field for editor: %s", field)
		return
	}

	switch field {
	case editorFieldDescription:
		m.formDesc.SetValue(content)
	case editorFieldNotes:
		m.formNotes.SetValue(content)
	case editorFieldDesign:
		m.formDesign.SetValue(content)
	case editorFieldAcceptance:
		m.formAcceptance.SetValue(content)
	}

	m.formFocus = focus
	m.updateFormFocus()
	m.updateFormTextAreaHeights()
}

func formFocusForEditorField(field editorField) (int, bool) {
	switch field {
	case editorFieldDescription:
		return 1, true
	case editorFieldNotes:
		return 2, true
	case editorFieldDesign:
		return 3, true
	case editorFieldAcceptance:
		return 4, true
	default:
		return 0, false
	}
}

func (m *Model) updateFormTextAreaHeights() {
	descWidth := m.formDesc.Width()
	if descWidth < 1 {
		descWidth = 1
	}
	notesWidth := m.formNotes.Width()
	if notesWidth < 1 {
		notesWidth = 1
	}
	designWidth := m.formDesign.Width()
	if designWidth < 1 {
		designWidth = 1
	}
	acceptWidth := m.formAcceptance.Width()
	if acceptWidth < 1 {
		acceptWidth = 1
	}

	m.formDesc.SetHeight(calcTextareaHeight(m.formDesc.Value(), descWidth))
	m.formNotes.SetHeight(calcTextareaHeight(m.formNotes.Value(), notesWidth))
	m.formDesign.SetHeight(calcTextareaHeight(m.formDesign.Value(), designWidth))
	m.formAcceptance.SetHeight(calcTextareaHeight(m.formAcceptance.Value(), acceptWidth))
	m.updateFormSubmitBounds()
}

func calcTextareaHeight(value string, width int) int {
	if width < 1 {
		return 1
	}
	lines := strings.Split(value, "\n")
	height := 0
	for _, line := range lines {
		if line == "" {
			height++
			continue
		}
		normalized := strings.ReplaceAll(line, "\t", "    ")
		lineWidth := runewidth.StringWidth(normalized)
		if lineWidth == 0 {
			height++
			continue
		}
		wrapped := (lineWidth + width - 1) / width
		if lineWidth%width == 0 {
			wrapped++
		}
		height += wrapped
	}
	if height < 1 {
		return 1
	}
	return height
}

func bumpTextareaHeightForNewline(ta *textarea.Model) {
	width := ta.Width()
	if width < 1 {
		width = 1
	}
	target := calcTextareaHeight(ta.Value(), width) + 1
	if ta.Height() < target {
		ta.SetHeight(target)
	}
}

func (m *Model) updateFormSubmitBounds() {
	blocks, button, _ := m.formViewBlocks()
	y := 0
	for _, block := range blocks {
		y += lipgloss.Height(block)
	}
	m.formSubmitBounds = formBounds{
		X: 0,
		Y: y,
		W: lipgloss.Width(button),
		H: lipgloss.Height(button),
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
