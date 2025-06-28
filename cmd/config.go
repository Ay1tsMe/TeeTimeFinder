/*
Sets up TeeTimeFinder config file for the user.
Collects information such as:
- Golf Course Names
- URLs
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type CourseInfo struct {
	Name        string
	URL         string
	WebsiteType string
	Blacklisted bool
}

type configModel struct {
	focusIndex int
	inputs     []textinput.Model
	cursorMode cursor.Mode
	courses    []CourseInfo
	current    CourseInfo
	done       bool
	err        error
	success    string
}

type blacklistItem struct {
	course CourseInfo
	index  int
}

type blacklistModel struct {
	list    list.Model
	courses []CourseInfo
}

type deleteItem struct {
	course CourseInfo
	index int
	selected bool
}

var (
	configPath = filepath.Join(os.Getenv("HOME"), ".config", "TeeTimeFinder", "config.txt")
	overwrite  bool
)

// Checks if the config file exists
func ConfigExists() bool {
	_, err := os.Stat(configPath)
	return err == nil
}

// Creates the .config/TeeTimeFinder directory if it doesn't exist
func CreateDir() bool {
	dir := filepath.Dir(configPath)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Printf("Failed to create directory: %s\n", err)
			return false
		}
	}
	return true
}

// Loads the existing courses from the config file into a map
func loadExistingCourses() map[string]CourseInfo {
	courses := make(map[string]CourseInfo)

	if !ConfigExists() {
		return courses
	}

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("Failed to open config file: %s\n", err)
		return courses
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 4)
		if len(parts) < 3 {
			continue
		}

		courseName := strings.TrimSpace(parts[0])
		courseURL := strings.TrimSpace(parts[1])
		websiteType := strings.TrimSpace(parts[2])

		blacklisted := false
		if len(parts) == 4 {
			bl := strings.TrimSpace(parts[3])
			blacklisted = strings.EqualFold(bl, "true")
		}

		courses[courseURL] = CourseInfo{
			Name:        courseName,
			URL:         courseURL,
			WebsiteType: websiteType,
			Blacklisted: blacklisted,
		}
	}
	return courses
}

// Appends new courses to the config file
func appendCoursesToFile(courses []CourseInfo) error {
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, course := range courses {
		line := fmt.Sprintf("%s,%s,%s,%t\n", course.Name, course.URL, course.WebsiteType, course.Blacklisted)
		_, err := file.WriteString(line)
		if err != nil {
			return err
		}
	}
	return nil
}

// Overwrites the entire config file
func overwriteCoursesToFile(courses []CourseInfo) error {
	file, err := os.OpenFile(configPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, course := range courses {
		line := fmt.Sprintf("%s,%s,%s,%t\n", course.Name, course.URL, course.WebsiteType, course.Blacklisted)
		_, err := file.WriteString(line)
		if err != nil {
			return err
		}
	}
	return nil
}

// bubbletea logic
func initialConfigModel() configModel {
	m := configModel{
		inputs: make([]textinput.Model, 3),
	}
	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.CharLimit = 512
		t.Width = 300
		t.PromptStyle = defaultStyle
		t.TextStyle = defaultStyle

		switch i {
		case 0:
			t.Placeholder = "Course Name"
			t.Focus()
		case 1:
			t.Placeholder = "Course URL"
		case 2:
			t.Placeholder = "Website Type (MiClub or Quick18)"
		}
		m.inputs[i] = t
	}
	return m
}

func (m configModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m configModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Always update the focused input
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()

		switch s {
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit

		case "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}

		case "down", "tab":
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}

		case "enter":
			if m.focusIndex == 2 {
				val := strings.ToLower(m.inputs[2].Value())
				if val != "miclub" && val != "quick18" {
					m.err = fmt.Errorf("Invalid website type")
					m.success = ""
					return m, nil
				}
				m.success = fmt.Sprintf("[SUCCESS] Added %s", m.inputs[0].Value())
				m.current.Name = m.inputs[0].Value()
				m.current.URL = m.inputs[1].Value()
				m.current.WebsiteType = val
				m.courses = append(m.courses, m.current)
				m.current = CourseInfo{}
				for i := range m.inputs {
					m.inputs[i].SetValue("")
				}
				m.focusIndex = 0
				m.err = nil
			} else {
				m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			}
		}
	}

	// Apply correct focus/blur and styles
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = hoverStyle
			m.inputs[i].TextStyle = hoverStyle
		} else {
			m.inputs[i].Blur()
			m.inputs[i].PromptStyle = defaultStyle
			m.inputs[i].TextStyle = defaultStyle
		}
	}

	return m, tea.Batch(append(cmds, cmd)...)
}

func (m configModel) View() string {
	var b strings.Builder
	b.WriteString("Enter golf course info:\n\n")
	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Courses added: %d\n", len(m.courses)))
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("[Error] %s\n", m.err)) + "\n")
	} else if m.success != "" {
		b.WriteString(successStyle.Render(m.success) + "\n")
	}

	b.WriteString("[Enter] to move next, [Esc] to quit\n")
	return b.String()
}

// bubbletea Blacklist logic
func (i blacklistItem) Title() string {
	return i.course.Name
}

func (i blacklistItem) Description() string {
	prefix := "[ ]"
	if i.course.Blacklisted {
		prefix = "[X]"
	}
	return fmt.Sprintf("%s %s (%s)", prefix, i.course.URL, i.course.WebsiteType)
}

func (i blacklistItem) FilterValue() string {
	return i.course.Name
}

type blacklistDelegate struct{}

func (d blacklistDelegate) Height() int                               { return 2 }
func (d blacklistDelegate) Spacing() int                              { return 1 }
func (d blacklistDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d blacklistDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item := listItem.(blacklistItem)

	style := defaultStyle
	if index == m.Index() {
		style = hoverStyle
	}

	prefix := "[ ]"
	if item.course.Blacklisted {
		prefix = "[X]"
	}

	title := style.Bold(true).Render(item.course.Name)
	desc := style.Render(fmt.Sprintf("%s %s (%s)", prefix, item.course.URL, item.course.WebsiteType))

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

func initialBlacklistModel(courses []CourseInfo) blacklistModel {
	items := make([]list.Item, len(courses))
	for i, c := range courses {
		items[i] = blacklistItem{course: c, index: i}
	}

	l := list.New(items, blacklistDelegate{}, 0, 0)
	l.Title = "Toggle blacklist status (press space to toggle, enter to save, esc to quit)"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return blacklistModel{
		list:    l,
		courses: courses,
	}
}

func (m blacklistModel) Init() tea.Cmd {
	return nil
}

func (m blacklistModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case " ":
			cursor := m.list.Index()
			m.courses[cursor].Blacklisted = !m.courses[cursor].Blacklisted

			// Refresh the list item's description
			m.list.SetItem(cursor, blacklistItem{
				course: m.courses[cursor],
				index:  cursor,
			})
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m blacklistModel) View() string {
	return m.list.View()
}

func (i deleteItem) Title() string {
	return i.course.Name
}

func (i deleteItem) Description() string {
	prefix := "[  ]"
	if i.selected {
		prefix = "[X]"
	}
	return fmt.Sprintf("%s %s (%s)", prefix, i.course.URL, i.course.WebsiteType)
}

func (i deleteItem) FilterValue() string {
	return i.course.Name
}

// list delegate (looks identical to blacklist’s but tracks “selected”)
type deleteDelegate struct{}

func (d deleteDelegate) Height() int                               { return 2 }
func (d deleteDelegate) Spacing() int                              { return 1 }
func (d deleteDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d deleteDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it := listItem.(deleteItem)

	style := defaultStyle
	if index == m.Index() {
		style = hoverStyle
	}
	prefix := "[ ]"
	if it.selected {
		prefix = "[X]"
	}
	title := style.Bold(true).Render(it.course.Name)
	desc := style.Render(fmt.Sprintf("%s %s (%s)", prefix, it.course.URL, it.course.WebsiteType))

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

// model that wraps the list and keeps track of chosen deletions
type deleteModel struct {
	list     list.Model
	courses  []CourseInfo     // original slice, index-aligned with list items
	selected map[int]struct{} // indices picked for removal
}

func initialDeleteModel(courses []CourseInfo) deleteModel {
	items := make([]list.Item, len(courses))
	for i, c := range courses {
		items[i] = deleteItem{course: c, index: i}
	}
	l := list.New(items, deleteDelegate{}, 0, 0)
	l.Title = "Select courses to delete (space to toggle, enter to confirm)"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return deleteModel{
		list:     l,
		courses:  courses,
		selected: map[int]struct{}{},
	}
}

func (m deleteModel) Init() tea.Cmd { return nil }

func (m deleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch k := msg.(type) {
	case tea.KeyMsg:
		switch k.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case " ":
			i := m.list.Index()
			if _, ok := m.selected[i]; ok {
				delete(m.selected, i)
			} else {
				m.selected[i] = struct{}{}
			}
			// redraw current item with new prefix
			cur := m.list.Items()[i].(deleteItem)
			cur.selected = !cur.selected
			m.list.SetItem(i, cur)
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(k.Width, k.Height)
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m deleteModel) View() string { return m.list.View() }

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure golf courses for TeeTimeFinder",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialConfigModel())
		m, err := p.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		model := m.(configModel)

		// No courses to add
		if len(model.courses) == 0 {
			fmt.Println("No courses were added.")
			return
		}

		// Ensure the directory exists
		if !CreateDir() {
			return
		}

		// Either append or overwrite the file based on the -o flag
		if overwrite {
			err = overwriteCoursesToFile(model.courses)
		} else {
			err = appendCoursesToFile(model.courses)
		}
		if err != nil {
			fmt.Printf("Failed to save to config file: %v\n", err)
			return
		}

		fmt.Println("Configuration saved!")
	},
}

// Command to show the config
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all configured golf courses",
	Run: func(cmd *cobra.Command, args []string) {
		if !ConfigExists() {
			fmt.Println("No config file found. Please add courses using `TeeTimeFinder config`.")
			return
		}

		courses := loadExistingCourses()
		if len(courses) == 0 {
			fmt.Println("No courses found in the config.")
			return
		}

		fmt.Println("Configured Golf Courses:")
		fmt.Println("   [X] indicates the course is blacklisted (skipped in ALL searches).")
		fmt.Println("   [ ] indicates the course is *not* blacklisted.")
		fmt.Println()

		i := 1

		for _, course := range courses {
			blMark := " "
			if course.Blacklisted {
				blMark = "X"
			}
			fmt.Printf("%d) [%s] %s - %s - %s\n", i, blMark, course.Name, course.URL, course.WebsiteType)
			i++
		}
	},
}

// Command to blacklist a course from the search
var configBlacklistCmd = &cobra.Command{
	Use:   "blacklist",
	Short: "Toggle blacklisted status so courses are skipped (or re-included) in 'ALL' searches",
	Run: func(cmd *cobra.Command, args []string) {
		// Load existing courses into a slice for stable ordering
		existing := loadExistingCoursesSlice()
		if len(existing) == 0 {
			fmt.Println("No courses found in config.")
			return
		}

		p := tea.NewProgram(initialBlacklistModel(existing), tea.WithAltScreen())
		m, err := p.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		model := m.(blacklistModel)

		if !CreateDir() {
			return
		}

		err = overwriteCoursesToFile(model.courses)
		if err != nil {
			fmt.Printf("Failed to save updated blacklist status: %s\n", err)
			return
		}
		fmt.Println("Blacklist updates applied successfully!")
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete courses from the config file",
	Run: func(cmd *cobra.Command, args []string) {
		existing := loadExistingCoursesSlice()
		if len(existing) == 0 {
			fmt.Println("No courses found in config.")
			return
		}

		p := tea.NewProgram(initialDeleteModel(existing), tea.WithAltScreen())
		m, err := p.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		model := m.(deleteModel)

		// Build a slice with everything *not* selected
		var remaining []CourseInfo
		for i, c := range model.courses {
			if _, remove := model.selected[i]; !remove {
				remaining = append(remaining, c)
			}
		}

		if len(remaining) == len(existing) {
			fmt.Println("No courses were deleted.")
			return
		}

		if !CreateDir() {
			return
		}
		if err := overwriteCoursesToFile(remaining); err != nil {
			fmt.Printf("Failed to save updated config: %v\n", err)
			return
		}
		fmt.Println("Selected courses deleted successfully!")
	},
}

// Helper to load into a slice instead of a map, so we can preserve an index
func loadExistingCoursesSlice() []CourseInfo {
	cMap := loadExistingCourses()
	var out []CourseInfo
	for _, c := range cMap {
		out = append(out, c)
	}
	// (Optional) sort by c.Name or something so it’s stable
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// Initialises the command and adds the -overwrite flag
func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "Overwrite the existing config")

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configBlacklistCmd)
	configCmd.AddCommand(configDeleteCmd)
}
