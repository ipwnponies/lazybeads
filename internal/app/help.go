package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazybeads/internal/config"
	"lazybeads/internal/ui"
)

type helpItemKind int

const (
	helpItemBinding helpItemKind = iota
	helpItemCustom
)

type helpItem struct {
	key     string
	desc    string
	context string
	trigger string
	kind    helpItemKind
	command config.CustomCommand
}

func (h helpItem) Title() string {
	return fmt.Sprintf("%-12s %s", h.key, h.desc)
}

func (h helpItem) Description() string {
	if h.context == "" {
		return ""
	}
	return fmt.Sprintf("(%s)", h.context)
}

func (h helpItem) FilterValue() string {
	return strings.TrimSpace(h.key + " " + h.desc + " " + h.context)
}

func newHelpList(items []helpItem) list.Model {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	l := list.New(listItems, helpDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	return l
}

func buildHelpItems(keys ui.KeyMap, customCmds []config.CustomCommand) []helpItem {
	customKeySet := make(map[string]struct{}, len(customCmds))
	for _, cmd := range customCmds {
		customKeySet[cmd.Key] = struct{}{}
	}

	var items []helpItem
	for _, group := range keys.FullHelp() {
		for _, binding := range group {
			help := binding.Help()
			if help.Key == "" || help.Desc == "" {
				continue
			}
			if _, isCustom := customKeySet[help.Key]; isCustom {
				continue
			}
			trigger := firstBindingKey(binding.Keys())
			items = append(items, helpItem{
				key:     help.Key,
				desc:    help.Desc,
				trigger: trigger,
				kind:    helpItemBinding,
			})
		}
	}

	for _, cmd := range customCmds {
		items = append(items, helpItem{
			key:     cmd.Key,
			desc:    cmd.Description,
			context: cmd.Context,
			trigger: cmd.Key,
			kind:    helpItemCustom,
			command: cmd,
		})
	}

	return items
}

func firstBindingKey(keys []string) string {
	for _, key := range keys {
		if key != "" {
			return key
		}
	}
	return ""
}

func (m *Model) selectedHelpItem() *helpItem {
	items := m.helpList.Items()
	if len(items) == 0 {
		return nil
	}
	idx := m.helpList.Index()
	if idx < 0 || idx >= len(items) {
		return nil
	}
	item, ok := items[idx].(helpItem)
	if !ok {
		return nil
	}
	return &item
}

func (m *Model) executeHelpSelection() tea.Cmd {
	item := m.selectedHelpItem()
	if item == nil {
		return nil
	}

	m.mode = ViewList
	switch item.kind {
	case helpItemCustom:
		return m.executeCustomCommand(item.command)
	case helpItemBinding:
		if item.trigger == "" {
			return nil
		}
		return m.handleKeyPress(keyMsgFromString(item.trigger))
	}
	return nil
}

func keyMsgFromString(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

func helpModalSize(width, height int) (int, int) {
	maxWidth := width - 4
	if maxWidth < 30 {
		maxWidth = max(width-2, 20)
	}
	modalWidth := min(maxWidth, 80)

	maxHeight := height - 4
	if maxHeight < 7 {
		maxHeight = max(height-2, 6)
	}
	modalHeight := min(maxHeight, 20)

	return modalWidth, modalHeight
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type helpDelegate struct{}

func (d helpDelegate) Height() int                             { return 1 }
func (d helpDelegate) Spacing() int                            { return 0 }
func (d helpDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d helpDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	helpItem, ok := item.(helpItem)
	if !ok {
		return
	}

	keyText := ui.HelpKeyStyle.Render(fmt.Sprintf("%-10s", helpItem.key))
	descText := ui.HelpDescStyle.Render(helpItem.desc)
	contextText := ""
	if helpItem.context != "" {
		contextStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
		contextText = contextStyle.Render(" (" + helpItem.context + ")")
	}
	line := keyText + " " + descText + contextText

	width := m.Width()
	if width > 0 {
		line = lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line)
	}

	if index == m.Index() {
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("#2a4a6d")).
			Bold(true)
		rawContext := ""
		if helpItem.context != "" {
			rawContext = " (" + helpItem.context + ")"
		}
		raw := fmt.Sprintf("%-10s %s%s", helpItem.key, helpItem.desc, rawContext)
		if width > 0 {
			selectedStyle = selectedStyle.Width(width)
		}
		fmt.Fprint(w, selectedStyle.Render(raw))
		return
	}

	fmt.Fprint(w, line)
}
