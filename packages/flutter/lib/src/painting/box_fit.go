package painting

// Port of box_fit.dart (Flutter 3.50.1).

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// BoxFit says how a box should be inscribed into another box.
type BoxFit int

const (
	// BoxFitFill fills the target box by distorting the source's aspect ratio.
	BoxFitFill BoxFit = iota
	// BoxFitContain is as large as possible while still containing the
	// source entirely within the target box.
	BoxFitContain
	// BoxFitCover is as small as possible while still covering the entire
	// target box.
	BoxFitCover
	// BoxFitFitWidth makes sure the full width of the source is shown.
	BoxFitFitWidth
	// BoxFitFitHeight makes sure the full height of the source is shown.
	BoxFitFitHeight
	// BoxFitNone aligns the source within the target box and discards any
	// portion outside it, without scaling.
	BoxFitNone
	// BoxFitScaleDown is BoxFitNone, scaled down like BoxFitContain when
	// the source is larger than the target.
	BoxFitScaleDown
)

// BoxFitValues is BoxFit.values.
var BoxFitValues = []BoxFit{BoxFitFill, BoxFitContain, BoxFitCover, BoxFitFitWidth,
	BoxFitFitHeight, BoxFitNone, BoxFitScaleDown}

// Name is the Dart enum value's name.
func (f BoxFit) Name() string {
	return [...]string{"fill", "contain", "cover", "fitWidth", "fitHeight", "none", "scaleDown"}[f]
}

func (f BoxFit) String() string { return "BoxFit." + f.Name() }

// FittedSizes are the pair of sizes ApplyBoxFit returns.
type FittedSizes struct {
	// Source is the size of the part of the input to show on the output.
	Source ui.Size
	// Destination is the size of the part of the output on which to show
	// the input.
	Destination ui.Size
}

// NewFittedSizes is FittedSizes(source, destination).
func NewFittedSizes(source, destination ui.Size) FittedSizes { return FittedSizes{source, destination} }

// ApplyBoxFit applies a BoxFit value: how much of inputSize to show
// (Source), and at what size on outputSize (Destination).
func ApplyBoxFit(fit BoxFit, inputSize, outputSize ui.Size) FittedSizes {
	if inputSize.Height <= 0.0 || inputSize.Width <= 0.0 || outputSize.Height <= 0.0 || outputSize.Width <= 0.0 {
		return FittedSizes{ui.SizeZero, ui.SizeZero}
	}
	var sourceSize, destinationSize ui.Size
	switch fit {
	case BoxFitFill:
		sourceSize = inputSize
		destinationSize = outputSize
	case BoxFitContain:
		sourceSize = inputSize
		if outputSize.Width/outputSize.Height > sourceSize.Width/sourceSize.Height {
			destinationSize = ui.NewSize(sourceSize.Width*outputSize.Height/sourceSize.Height, outputSize.Height)
		} else {
			destinationSize = ui.NewSize(outputSize.Width, sourceSize.Height*outputSize.Width/sourceSize.Width)
		}
	case BoxFitCover:
		if outputSize.Width/outputSize.Height > inputSize.Width/inputSize.Height {
			sourceSize = ui.NewSize(inputSize.Width, inputSize.Width*outputSize.Height/outputSize.Width)
		} else {
			sourceSize = ui.NewSize(inputSize.Height*outputSize.Width/outputSize.Height, inputSize.Height)
		}
		destinationSize = outputSize
	case BoxFitFitWidth:
		if outputSize.Width/outputSize.Height > inputSize.Width/inputSize.Height {
			sourceSize = ui.NewSize(inputSize.Width, inputSize.Width*outputSize.Height/outputSize.Width)
			destinationSize = outputSize
		} else {
			sourceSize = inputSize
			destinationSize = ui.NewSize(outputSize.Width, sourceSize.Height*outputSize.Width/sourceSize.Width)
		}
	case BoxFitFitHeight:
		if outputSize.Width/outputSize.Height > inputSize.Width/inputSize.Height {
			sourceSize = inputSize
			destinationSize = ui.NewSize(sourceSize.Width*outputSize.Height/sourceSize.Height, outputSize.Height)
		} else {
			sourceSize = ui.NewSize(inputSize.Height*outputSize.Width/outputSize.Height, inputSize.Height)
			destinationSize = outputSize
		}
	case BoxFitNone:
		sourceSize = ui.NewSize(math.Min(inputSize.Width, outputSize.Width), math.Min(inputSize.Height, outputSize.Height))
		destinationSize = sourceSize
	case BoxFitScaleDown:
		sourceSize = inputSize
		destinationSize = inputSize
		aspectRatio := inputSize.Width / inputSize.Height
		if destinationSize.Height > outputSize.Height {
			destinationSize = ui.NewSize(outputSize.Height*aspectRatio, outputSize.Height)
		}
		if destinationSize.Width > outputSize.Width {
			destinationSize = ui.NewSize(outputSize.Width, outputSize.Width/aspectRatio)
		}
	}
	return FittedSizes{sourceSize, destinationSize}
}
