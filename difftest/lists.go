package difftest

import (
	r "github.com/jonathanperis/flutter-go/rendering"
	. "github.com/jonathanperis/flutter-go/widgets"
)

// ListScene is the twin of ListScene in flutter/test/lists_test.dart.
type ListScene struct {
	Count    int
	Extent   float64
	JumpTo   float64
	InColumn bool
}

var ListScenes = map[string]ListScene{
	"list_top":        {100, 50, 0, false},
	"list_scrolled":   {100, 50, 1234, false},
	"list_end":        {100, 50, 1e9, false},
	"list_in_column":  {1000, 37, 777, true},
	"list_short":      {5, 50, 100, false},
	"list_fractional": {10000, 33.3, 1000.5, false},
}

func (s ListScene) Widget(c *ScrollController) Widget {
	var w Widget = ListViewBuilder{
		ItemCount: s.Count, ItemExtent: s.Extent, Controller: c,
		ItemBuilder: func(_ *Element, i int) Widget { return ColoredBox{Color: r.RGB(0xeeeeee)} },
	}
	if s.InColumn {
		w = Column{Children: []Widget{SizedBox{Height: F(100)}, Expanded{Child: w}}}
	}
	return w
}
