package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pokstad/nestable/orm"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)
)

type listKeyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	insertItem       key.Binding
}

func newListKeyMap() listKeyMap {
	return listKeyMap{
		insertItem: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add item"),
		),
		toggleSpinner: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle spinner"),
		),
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
	}
}

type browseModel struct {
	list         list.Model
	choices      []orm.NoteRev
	keys         listKeyMap
	delegateKeys delegateKeyMap
}

type revItem struct {
	ctx  context.Context
	nr   orm.NoteRev
	repo orm.Repo
}

func (ri revItem) FilterValue() string {
	head, err := ri.nr.Blob.GetBlobHead(ri.ctx, ri.repo, 80)
	if err != nil {
		panic(err)
	}
	return string(head)
}

func (ri revItem) Title() string {
	head, err := ri.nr.Blob.GetBlobHead(ri.ctx, ri.repo, 80)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("[%d] %s", ri.nr.ID, string(head))
}

func (ri revItem) Description() string {
	return "no description yet"
}

type delegateKeyMap struct {
	choose key.Binding
	remove key.Binding
}

func newDelegateKeyMap() delegateKeyMap {
	return delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "choose"),
		),
		remove: key.NewBinding(
			key.WithKeys("x", "backspace"),
			key.WithHelp("x", "delete"),
		),
	}
}

func newRevItemDelegate(keys delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string

		if i, ok := m.SelectedItem().(revItem); ok {
			title = i.Title()
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.choose):
				return m.NewStatusMessage("You chose " + title)

			case key.Matches(msg, keys.remove):
				index := m.Index()
				m.RemoveItem(index)
				if len(m.Items()) == 0 {
					keys.remove.SetEnabled(false)
				}
				return m.NewStatusMessage("Deleted " + title)
			}
		}

		return nil
	}

	help := []key.Binding{keys.choose, keys.remove}
	d.ShortHelpFunc = func() []key.Binding { return help }
	d.FullHelpFunc = func() [][]key.Binding { return [][]key.Binding{help} }
	return d
}

func loadBrowseModel(ctx context.Context, repo orm.Repo) (browseModel, error) {
	revs, err := repo.GetNotes(ctx)
	if err != nil {
		return browseModel{}, fmt.Errorf("getting notes for browse model: %w", err)
	}

	items := make([]list.Item, len(revs))
	for i, r := range revs {
		items[i] = revItem{
			ctx:  ctx,
			nr:   r,
			repo: repo,
		}
	}

	noteList := list.New(items, newRevItemDelegate(newDelegateKeyMap()), 0, 0)
	noteList.Title = "Notes"

	return browseModel{
		list:         noteList,
		keys:         newListKeyMap(),
		delegateKeys: newDelegateKeyMap(),
	}, nil
}

func (bm browseModel) Init() tea.Cmd { return tea.EnterAltScreen }

func (bm browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		bm.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if bm.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, bm.keys.toggleSpinner):
			cmd := bm.list.ToggleSpinner()
			return bm, cmd

		case key.Matches(msg, bm.keys.toggleTitleBar):
			v := !bm.list.ShowTitle()
			bm.list.SetShowTitle(v)
			bm.list.SetShowFilter(v)
			bm.list.SetFilteringEnabled(v)
			return bm, nil

		case key.Matches(msg, bm.keys.toggleStatusBar):
			bm.list.SetShowStatusBar(!bm.list.ShowStatusBar())
			return bm, nil

		case key.Matches(msg, bm.keys.togglePagination):
			bm.list.SetShowPagination(!bm.list.ShowPagination())
			return bm, nil

		case key.Matches(msg, bm.keys.toggleHelpMenu):
			bm.list.SetShowHelp(!bm.list.ShowHelp())
			return bm, nil

			//		case key.Matches(msg, bm.keys.insertItem):
			//			bm.delegateKeys.remove.SetEnabled(true)
			//			newItem := bm.itemGenerator.next()
			//			insCmd := bm.list.InsertItem(0, newItem)
			//			statusCmd := bm.list.NewStatusMessage(statusMessageStyle("Added " + newItem.Title()))
			//			return bm, tea.Batch(insCmd, statusCmd)
		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := bm.list.Update(msg)
	bm.list = newListModel
	cmds = append(cmds, cmd)

	return bm, tea.Batch(cmds...)
}

func (bm browseModel) View() string {
	return bm.list.View()
}

type browseCmd struct {
	repo orm.Repo
}

func newBrowseCmd(repo orm.Repo) subCmd {
	return &browseCmd{repo: repo}
}

func (_ *browseCmd) Help() string {
	return `Browse and manage notes in an interactive list.`
}

func (_ *browseCmd) Names() []string {
	return []string{"browse", "b"}
}

func (bc *browseCmd) FlagSet() *flag.FlagSet {
	return flag.NewFlagSet("browse", flag.ExitOnError)
}

func (bc *browseCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	bm, err := loadBrowseModel(ctx, bc.repo)
	if err != nil {
		return fmt.Errorf("loading browse model: %w", err)
	}

	if err := tea.NewProgram(bm).Start(); err != nil {
		return fmt.Errorf("start browse TUI: %w", err)
	}
	return nil
}
