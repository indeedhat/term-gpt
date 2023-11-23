package gpt

import "github.com/charmbracelet/bubbles/list"

type listItem struct {
	title string
	date  string
}

// FilterValue implements list.Item.
func (i listItem) FilterValue() string {
	return i.title
}

// Title returns the private title member
func (i listItem) Title() string {
	return i.title
}

// Description returns the private date member as the list item description
func (i listItem) Description() string {
	return i.date
}

var _ list.Item = (*listItem)(nil)

// testItems is used for setup debug until i implement the sqlite store
// TODO: replace this with sqlite store
func testItems() []list.Item {
	return []list.Item{
		listItem{title: "test 1", date: "02/01/05"},
		listItem{title: "test 2", date: "03/01/05"},
		listItem{title: "test 3", date: "04/01/05"},
	}
}
