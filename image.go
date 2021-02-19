package vimg

import (
	"bytes"
	"errors"
	"github.com/karlaustin/refcount"
	"github.com/prometheus/client_golang/prometheus"
)

// Image provides a simple method DSL to transform a given image as byte buffer.
type Image struct {
	refcount.ReferenceCounter
	VipsImage *VipsImage
}

// NewImage creates a new Image struct with method DSL.
func NewImage(buf *bytes.Buffer, o Options) (*Image, error) {
	vimgImageBuffer.With(prometheus.Labels{"action":"request", "type":"image"}).Inc()
	var err error
	ret := AquireImage()
	ret.VipsImage, err = NewVipsImage(buf, o)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func ResetImage(i interface{}) error {
	img, ok := i.(*Image)
	if !ok {
		return errors.New("illegal object sent to ResetVipsImage")
	}
	img.Reset()
	return nil
}

func AquireImage() *Image {
	return ImagePool.Get().(*Image)
}

var ImagePool = refcount.NewReferenceCountedPool(
	func(counter refcount.ReferenceCounter) refcount.ReferenceCountable {
		vimgImageBuffer.With(prometheus.Labels{"action":"new", "type":"image"}).Inc()
		i := new(Image)
		i.ReferenceCounter = counter
		return i
	}, ResetImage)

func (img *Image) Reset() {
	img.VipsImage.DecrementReferenceCount()
	img.VipsImage = nil
	//img.VipsImage.Reset()
}

func (i *Image) SetOptions(o Options)  {
	i.VipsImage.Options = o
}

// Resize resizes the image to fixed width and height.
func (i *Image) Resize(width, height int) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Embed = true

	return i.Process()
}

// ForceResize resizes with custom size (aspect ratio won't be maintained).
func (i *Image) ForceResize(width, height int) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Force = true

	return i.Process()
}

// ResizeAndCrop resizes the image to fixed width and height with additional crop transformation.
func (i *Image) ResizeAndCrop(width, height int) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Embed = true
	i.VipsImage.Options.Crop = true

	return i.Process()
}

// SmartCrop produces a thumbnail aiming at focus on the interesting part.
func (i *Image) SmartCrop(width, height int) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Gravity = GravitySmart
	i.VipsImage.Options.Crop = true

	return i.Process()
}

// Extract area from the by X/Y axis in the current image.
func (i *Image) Extract(top, left, width, height int) error {
	i.VipsImage.Options.Extract.Width = float32(width)
	i.VipsImage.Options.Extract.Height = float32(height)
	if top == 0 && left == 0 {
		i.VipsImage.Options.Extract.Top = -1
	} else {
		i.VipsImage.Options.Extract.Top = float32(top)
	}
	i.VipsImage.Options.Extract.Left = float32(left)
	
	return i.Process()
}

// Enlarge enlarges the image by width and height. Aspect ratio is maintained.
func (i *Image) Enlarge(width, height int) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Enlarge = true

	return i.Process()
}

// EnlargeAndCrop enlarges the image by width and height with additional crop transformation.
func (i *Image) EnlargeAndCrop(width, height int) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Force = true
	i.VipsImage.Options.Crop = true
	
	return i.Process()
}

// Crop crops the image to the exact size specified.
func (i *Image) Crop(width, height int, gravity Gravity) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Crop = true
	i.VipsImage.Options.Gravity = gravity

	return i.Process()
}

// CropByWidth crops an image by width only param (auto height).
func (i *Image) CropByWidth(width int) error {
	i.VipsImage.Options.Width = width
	i.VipsImage.Options.Crop = true

	return i.Process()
}

// CropByHeight crops an image by height (auto width).
func (i *Image) CropByHeight(height int) error {
	i.VipsImage.Options.Height = height
	i.VipsImage.Options.Crop = true

	return i.Process()
}

// Thumbnail creates a thumbnail of the image by the a given width by aspect ratio 4:4.
func (i *Image) Thumbnail(pixels int) error {
	i.VipsImage.Options.Width = pixels
	i.VipsImage.Options.Height = pixels
	i.VipsImage.Options.Crop = true
	i.VipsImage.Options.Quality = 95

	return i.Process()
}

// Watermark adds text as watermark on the given image.
func (i *Image) Watermark(w Watermark) error {
	i.VipsImage.Options.Watermark = w

	return i.Process()
}

// WatermarkImage adds image as watermark on the given image.
func (i *Image) WatermarkImage(w WatermarkImage) error {
	i.VipsImage.Options.WatermarkImage = w
	return i.Process()
}

// Zoom zooms the image by the given factor.
// You should probably call Extract() before.
func (i *Image) Zoom(factor int) error {
	i.VipsImage.Options.Zoom = factor
	return i.Process()
}

// Rotate rotates the image by given angle degrees (0, 90, 180 or 270).
func (i *Image) Rotate(a Angle) error {
	i.VipsImage.Options.Rotate = a
	return i.Process()
}

// Flip flips the image about the vertical Y axis.
func (i *Image) Flip() error {
	i.VipsImage.Options.Flip = true
	return i.Process()
}

// Flop flops the image about the horizontal X axis.
func (i *Image) Flop() error {
	i.VipsImage.Options.Flop = true
	return i.Process()
}

/**
 * Go listen to The Crag Rats: https://soundcloud.com/thecragrats
 */
func (i *Image) Fly() error {
	return errors.New("Don't care if I die")
}

// Convert converts image to another format.
func (i *Image) Convert(t ImageType) error {
	i.VipsImage.Options.Type = t
	return i.Process()
}

// Colourspace performs a color space conversion bsaed on the given interpretation.
func (i *Image) Colourspace(c Interpretation) error {
	i.VipsImage.Options.Interpretation = c
	return i.Process()
}

// Trim removes the background from the picture. It can result in a 0x0 output
// if the image is all background.
func (i *Image) Trim() error {
	i.VipsImage.Options.Trim = true
	return i.Process()
}

func (i *Image) GetICCProfile() ([]byte, error) {
	ret, err := i.VipsImage.GetICCProfile()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Process processes the image based on the given transformation options,
// talking with libvips bindings accordingly and returning the resultant
// image buffer.
func (i *Image) Process() error {
	err := i.VipsImage.Process()
	if err != nil {
		return err
	}
	return nil
}

func (i *Image) Save() (*[]byte, error) {
	err := i.VipsImage.Save()
	if err != nil {
		return nil, err
	}
	return i.GetBuffer(), nil
}

func (i *Image) GetBuffer() *[]byte {
	return &i.VipsImage.Buffer
}

// Metadata returns the image metadata (size, alpha channel, profile, EXIF rotation).
func (i *Image) Metadata() (ImageMetadata, error) {
	return i.Metadata()
}

// Interpretation gets the image interpretation type.
// See: https://jcupitt.github.io/libvips/API/current/VipsImage.html#VipsInterpretation
func (i *Image) Interpretation() (Interpretation, error) {
	return i.VipsImage.vipsInterpretation()
}

// ColourspaceIsSupported checks if the current image
// color space is supported.
func (i *Image) ColourspaceIsSupported() (bool, error) {
	return i.VipsImage.vipsColourspaceIsSupported()
	//return ColourspaceIsSupported(*i.GetBuffer())
}

// Type returns the image type format (jpeg, png, webp, tiff).
func (i *Image) Type() string {
	return DetermineImageTypeName(*i.GetBuffer())
}

// Size returns the image size as form of width and height pixels.
func (i *Image) Size() (ImageSize, error) {
	m, err := i.Metadata()
	return m.Size, err
}

// Image returns the current resultant image buffer.
func (i *Image) Image() *[]byte {
	return i.GetBuffer()
}

// Length returns the size in bytes of the image buffer.
func (i *Image) Length() int {
	return len(*i.GetBuffer())
}

// Gamma returns the gamma filtered image buffer.
func (i *Image) Gamma(exponent float64) error {
	i.VipsImage.Options.Gamma = exponent
	return i.Process()
}