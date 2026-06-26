package image

import (
	"context"
	"fmt"

	"github.com/davidbyttow/govips/v2/vips"
)

type Processor struct {}

func Initialize() {
	vips.Startup(&vips.Config{
		ConcurrencyLevel: 4,
	})
}

func NewProcessor() Processor {
	return Processor{}
}

func (p *Processor) ResizeAndCompress(ctx context.Context, rawImage []byte, maxWidth, maxHeight int, quality int) ([]byte, error) {
	image, err := vips.LoadImageFromBuffer(rawImage, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %w", err)
	}
	defer image.Close()

	width := image.Width()
	height := image.Height()

	if width <= maxWidth && height <= maxHeight {
		maxWidth = width
		maxHeight = height

	} else {
		ratio := float64(width) / float64(height)

		if float64(maxWidth)/float64(maxHeight) > ratio {
			maxWidth = int(float64(maxHeight) * ratio)
		} else {
			maxHeight = int(float64(maxWidth) / ratio)
		}
	}

	err = image.Resize(float64(maxWidth) / float64(width), vips.KernelLanczos2)

	if err != nil {
		return nil, fmt.Errorf("failed to resize image: %w", err)
	}

	exportParams := vips.NewJpegExportParams()
	exportParams.Quality = quality

	output, _, err := image.ExportJpeg(exportParams)
	if err != nil {
		return nil, fmt.Errorf("failed to export image: %w", err)
	}

	return output, nil
}
