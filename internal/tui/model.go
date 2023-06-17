// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/planta7/serve/internal/styles"
)

type item struct {
	title       string
	description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

type listKeyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
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

type Model struct {
	channel      chan item
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
}

func NewModel(info string) Model {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	delegate := newItemDelegate(delegateKeys)
	requestList := list.New([]list.Item{}, delegate, 0, 0)
	requestList.Title = info
	requestList.Styles.Title = styles.TitleStyle
	requestList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleSpinner,
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
		}
	}

	return Model{
		channel:      make(chan item),
		list:         requestList,
		keys:         listKeys,
		delegateKeys: delegateKeys,
	}
}

func (m Model) Add(title string, description string) {
	newItem := item{
		title:       title,
		description: description,
	}
	m.channel <- newItem
}

func waitForActivity(sub chan item) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}
