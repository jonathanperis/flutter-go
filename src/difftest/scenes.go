// Package difftest compares flutter-go layouts with real Flutter.
//
// Each scene here has a twin with the same name in
// flutter/test/scenes_test.dart. Run scripts/difftest.sh.
package difftest

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
)

var red = ColoredBox{Color: r.RGB(0xff0000)}

func box(w, h float64) Widget { return SizedBox{Width: F(w), Height: F(h)} }

var Scenes = map[string]Widget{
	"center_padding": Center{Child: Padding{Padding: painting.NewEdgeInsetsAll(10), Child: box(100, 50)}},
	"align_factors": Align{
		Alignment: painting.AlignmentBottomRight, WidthFactor: F(2), HeightFactor: F(1.5),
		Child: box(100, 40),
	},
	"root_colored_box": ColoredBox{Color: r.RGB(0x00ff00)},
	"row_expanded": Row{Children: []Widget{
		box(100, 20),
		Expanded{Child: red},
		Expanded{Flex: 3, Child: red},
	}},
	"column_space_between_stretch": Column{
		MainAxisAlignment: r.MainSpaceBetween, CrossAxisAlignment: r.CrossStretch,
		Children: []Widget{box(50, 100), SizedBox{Height: F(100)}},
	},
	"flexible_loose": Row{
		CrossAxisAlignment: r.CrossStart,
		Children: []Widget{
			Flexible{Child: box(50, 10)},
			Expanded{Flex: 2, Child: red},
			Flexible{Child: box(1000, 10)},
		},
	},
	"center_min_column": Center{Child: Column{
		MainAxisSize: r.MainMin,
		Children: []Widget{
			box(30, 10),
			Row{MainAxisSize: r.MainMin, Children: []Widget{box(5, 5), box(7, 9)}},
		},
	}},
	"nested_alignments": Column{Children: []Widget{
		Row{
			MainAxisAlignment: r.MainSpaceEvenly, CrossAxisAlignment: r.CrossEnd,
			Children: []Widget{box(10, 30), box(20, 10), box(30, 20)},
		},
		Expanded{Child: Padding{
			Padding: painting.NewEdgeInsetsSymmetric(painting.EdgeInsetsSymmetricOptions{Horizontal: 20, Vertical: 5}),
			Child: Row{
				MainAxisAlignment: r.MainSpaceAround, CrossAxisAlignment: r.CrossStart,
				Children: []Widget{box(40, 40), Align{Alignment: painting.AlignmentTopRight, Child: box(10, 10)}},
			},
		}},
		Align{Alignment: painting.AlignmentCenterRight, Child: box(25, 25)},
		Row{
			MainAxisAlignment: r.MainCenter,
			Children:          []Widget{box(10, 10), Expanded{Child: SizedBox{Height: F(4)}}, box(10, 10)},
		},
	}},
	"space_around_end": Column{
		MainAxisAlignment: r.MainSpaceAround, CrossAxisAlignment: r.CrossEnd,
		Children: []Widget{box(10, 10), box(20, 20), box(30, 30)},
	},
	"text_center": Center{Child: Text{Text: "hello", Style: r.TextStyle{FontSize: 20}}},
	"default_text_style": DefaultTextStyle{
		Style: r.TextStyle{FontSize: 20},
		Child: Center{Child: Text{Text: "hello"}},
	},
	"text_wrap": Align{
		Alignment: painting.AlignmentTopLeft,
		Child:     SizedBox{Width: F(100), Child: Text{Text: "hello wrapping world", Style: r.TextStyle{FontSize: 10}}},
	},
}
