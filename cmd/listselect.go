package cmd

import (
	"fmt"
	"io"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

var (
    itemStyle         = defaultStyle.Copy().PaddingLeft(2)        // dull row
    selectedItemStyle = hoverStyle.Copy().PaddingLeft(0)          // highlighted row
)

// list item
type item string

func (i item) Title() string       { return string(i) } // not used in delegate
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }


// custom delegate
type simpleDelegate struct{}

func (d simpleDelegate) Height() int                             { return 1 }
func (d simpleDelegate) Spacing() int                            { return 0 }
func (d simpleDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d simpleDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(item)
	if !ok {
		return
	}

	// prefix:  "> 1." for selected, "  1." otherwise
	prefix := fmt.Sprintf("%d.", index+1)
	if index == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render("> "+prefix+" "+string(it)))
	} else {
		fmt.Fprint(w, itemStyle.Render(prefix+" "+string(it)))
	}
}


// keymap
var defaultKeyMap = list.KeyMap{
	CursorUp:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	CursorDown: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	AcceptWhileFiltering: key.NewBinding(key.WithKeys("enter", "l", "right"), key.WithHelp("↵/l", "choose")),
	Quit: key.NewBinding(key.WithKeys("esc", "ctrl+c", "h"), key.WithHelp("esc/h", "cancel")),
}


// selector model
type selectorModel struct {
	list     list.Model
	choice   string
	cancel   bool
	finished bool
}

func newSelector(title string, options []string) selectorModel {
	items := make([]list.Item, len(options))
	for i, o := range options { items[i] = item(o) }

	l := list.New(items, simpleDelegate{}, 0, 0)
	l.Title = title
	l.KeyMap = defaultKeyMap
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = controlStyle

	return selectorModel{list: l}
}


// tea plumbing
func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeyMap.AcceptWhileFiltering):
			if i, ok := m.list.SelectedItem().(item); ok {
				m.choice, m.finished = string(i), true
				return m, tea.Quit
			}
		case key.Matches(msg, defaultKeyMap.Quit):
			m.cancel, m.finished = true, true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-4)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectorModel) View() string { return m.list.View() }

// exported helper
func selectFromList(title string, options []string) (string, bool, error) {
	if len(options) == 0 { return "", false, fmt.Errorf("no options to choose from") }

	p := tea.NewProgram(newSelector(title, options), tea.WithAltScreen())
	res, err := p.Run()
	if err != nil { return "", false, err }

	m := res.(selectorModel)
	if m.cancel { return "", false, nil }
	return m.choice, true, nil
}
