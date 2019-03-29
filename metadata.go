package vimg

/*
#cgo pkg-config: vips
#include "vips/vips.h"
*/
import "C"

// ImageSize represents the image width and height values
type ImageSize struct {
	Width  int
	Height int
}

// ImageMetadata represents the basic metadata fields
type ImageMetadata struct {
	Orientation int
	Channels    int
	Alpha       bool
	Profile     bool
	Type        string
	Space       string
	Colourspace string
	Size        ImageSize
}

// Metadata returns the image metadata (size, type, alpha channel, profile, EXIF orientation...).
func (img *VipsImage) Metadata() (ImageMetadata, error) {

	size := ImageSize{
		Width:  int(img.Image.Xsize),
		Height: int(img.Image.Ysize),
	}

	o, err := img.vipsExifOrientation()
	if err != nil { return ImageMetadata{}, err }

	a, err := img.vipsHasAlpha()
	if err != nil { return ImageMetadata{}, err }

	p, err := img.hasProfile()
	if err != nil { return ImageMetadata{}, err }

	s, err := img.vipsSpace()
	if err != nil { return ImageMetadata{}, err }

	b := img.Buffer
	metadata := ImageMetadata{
		Size:        size,
		Channels:    int(img.Image.Bands),
		Orientation: o,
		Alpha:       a,
		Profile:     p,
		Space:       s,
		Type:        ImageTypeName(vipsImageType(b)),
	}

	return metadata, nil
}
