package vimg

/*
#cgo pkg-config: vips
#include "vips/vips.h"
*/
import "C"

import (
	"bytes"
	"errors"
	"github.com/karlaustin/refcount"
	"github.com/prometheus/client_golang/prometheus"
	"math"
)

type VipsImage struct {
	refcount.ReferenceCounter
	Buffer 		[]byte
	Image 		*C.VipsImage
	Type    	ImageType
	Options		Options
}

func NewVipsImage(buf *bytes.Buffer, opt Options) (*VipsImage, error) {
	vimgImageBuffer.With(prometheus.Labels{"action":"request", "type":"vips"}).Inc()
	ret := AquireVipsImage()
	if err := ret.Load(buf); err != nil {
		return nil, err
	}
	ret.Options = opt
	return ret, nil
}

var (
	ErrExtractAreaParamsRequired = errors.New("extract area width/height params are required")
	ErrVipsImageNotValidPointer = errors.New("Image is not a valid pointer to *C.VipsImage")
)

func ResetVipsImage(i interface{}) error {
	img, ok := i.(*VipsImage)
	if !ok {
		return errors.New("illegal object sent to ResetVipsImage")
	}
	img.Reset()
	return nil
}

func AquireVipsImage() *VipsImage {
	return vipsImagePool.Get().(*VipsImage)
}

var vipsImagePool = refcount.NewReferenceCountedPool(
		func(counter refcount.ReferenceCounter) refcount.ReferenceCountable {
			vimgImageBuffer.With(prometheus.Labels{"action":"new", "type":"vips"}).Inc()
			vi := new(VipsImage)
			vi.Buffer = make([]byte, 1024 * 2048)
			vi.ReferenceCounter = counter
			return vi
		}, ResetVipsImage)

func (img *VipsImage) Load(buf *bytes.Buffer) error {
	if buf.Len() == 0 {
		return errors.New("Image buffer is empty")
	}

	var err error
	err = img.vipsRead(buf)
	if err != nil {
		return err
	}

	return nil
}

func (img *VipsImage) Reset() {
	img.Buffer = nil
	img.Type = UNKNOWN
	img.Options = Options{}
	img.Image = nil
}

/**
 * All the heavy work happens here, Process() looks at the Options and works out what needs doing to the image
 */
func (img *VipsImage) Process() error {
	// Make sure defaults are applied sensibly
	img.applyDefaults()

	// Can we work with this image?
	if !IsTypeSupported(img.Options.Type) {
		return errors.New("Unsupported image output type")
	}

	/**
	 * Rotate early, so the output image is the correct size requested
	 */
	rotated, err := img.rotateAndFlipImage(true)
	if err != nil {
		return err
	}

	/**
	 * If the image has been rotated retrieve the buffer, otherwise the rotation will not manifest
	 */
	if rotated {//} && !img.Options.NoAutoRotate {
		img.Buffer, err = img.getImageBuffer()
		if err != nil {
			return err
		}
	}

	// Infer the required operation based on the in/out image sizes for a coherent transformation
	img.normalizeOperation()

	inWidth := int(img.Image.Xsize)
	inHeight := int(img.Image.Ysize)

	// Do not enlarge the output if the input width or height
	// are already less than the required dimensions
	if !img.Options.Enlarge && !img.Options.Force &&
	(inWidth < img.Options.Width && inHeight < img.Options.Height) {
			img.Options.Width = inWidth
			img.Options.Height = inHeight
	}

	factor := img.ScaleFactor()
	shrink := img.calculateShrink()
	residual := img.calculateResidual()

	// Try to use libjpeg/libwebp shrink-on-load
	supportsShrinkOnLoad := img.Type == WEBP && VipsMajorVersion >= 8 && VipsMinorVersion >= 3
	supportsShrinkOnLoad = supportsShrinkOnLoad || img.Type == JPEG
	if supportsShrinkOnLoad && shrink >= 2 {
		factor, err = img.shrinkOnLoad()
		if err != nil {
			return err
		}

		factor = math.Max(factor, 1.0)
		shrink = int(math.Floor(factor))
		residual = float64(shrink) / factor
	}

	// Zoom image, if necessary
	err = img.zoomImage()
	if err != nil {
		return err
	}

	// Transform image, if necessary
	if img.shouldTransformImage() {
		err = img.transformImage(shrink, residual)
		if err != nil {
			return err
		}
	}

	// Apply effects, if necessary
	if img.shouldApplyEffects() {
		err = img.applyEffects()
		if err != nil {
			return err
		}
	}

	// Add watermark, if necessary
	err = img.watermarkWithText()
	if err != nil {
		return err
	}

	// Add watermark, if necessary
	err = img.watermarkWithImage()
	if err != nil {
		return err
	}

	// Flatten image on a background, if necessary
	err = img.Flatten()
	if err != nil {
		return err
	}

	// Apply Gamma filter, if necessary
	err = img.applyGamma()
	if err != nil {
		return err
	}

	return nil
}

func (img *VipsImage) applyDefaults() {
	o := &img.Options
	if o.Quality == 0 {
		o.Quality = Quality
	}
	if o.Compression == 0 {
		o.Compression = 6
	}
	if o.Type == 0 {
		o.Type = img.Type
	}
	if o.Interpretation == 0 {
		o.Interpretation = InterpretationSRGB
	}
}

func (img *VipsImage) Save() error {
	o := &img.Options
	saveOptions := vipsSaveOptions{
		Quality:        o.Quality,
		Type:           o.Type,
		Compression:    o.Compression,
		Interlace:      o.Interlace,
		NoProfile:      o.NoProfile,
		Interpretation: o.Interpretation,
		OutputICC:      o.OutputICC,
		StripMetadata:  o.StripMetadata,
		Lossless:       o.Lossless,
	}

	err := img.vipsSave(saveOptions)
	if err != nil {
		return err
	}

	return nil
}

func (img *VipsImage) GetICCProfile() ([]byte, error) {
	vimgOperations.With(prometheus.Labels{"type":"geticc"}).Inc()
	hasProfile, err := img.hasProfile()
	if err != nil {
		return nil, err
	}
	if !hasProfile {
		return nil, errors.New("No Profile")
	}
	blob, err := img.vipsBlob(VIPS_META_ICC_NAME)
	if err != nil {
		return nil, err
	}
	return *blob, nil
}

func (img *VipsImage) normalizeOperation() {
	o := &img.Options
	if !o.MaintainAspect && !o.Force && !o.Crop && !o.Embed && !o.Enlarge && o.Rotate == 0 && (o.Width > 0 || o.Height > 0) {
		o.Force = true
	}
}

func (img *VipsImage) shouldTransformImage() bool {
	o := &img.Options
	inWidth := int(img.Image.Xsize)
	inHeight := int(img.Image.Ysize)

	/**
	 * As we've modified things so o.Force isn't set unless o.MaintainAspect is false (or it's explicitly set to enlarge)
	 * we're not doing a basic resize if the width and height are more than the original, we just return the original.
	 * So we can skip the transport for basic resize.
	*/
	return o.Force || (o.Width > 0 && o.Width < inWidth) ||
		(o.Height > 0 && o.Height < inHeight) || o.Extract.Width > 0 || o.Extract.Height > 0 ||
		(o.Height > 0 && o.Height > inHeight && o.Enlarge) || (o.Width > 0 && o.Width > inWidth && o.Enlarge) ||
		o.Trim
}

func (img *VipsImage) shouldApplyEffects() bool {
	o := &img.Options
	return o.GaussianBlur.Sigma > 0 || o.GaussianBlur.MinAmpl > 0 || o.Sharpen.Sigma > 0 && o.Sharpen.Y2 > 0 || o.Sharpen.Y3 > 0
}

func (img *VipsImage) transformImage(shrink int, residual float64) error {
	var err error
	// Use vips_shrink with the integral reduction
	if shrink > 1 {
		residual, err = img.shrinkImage(img.Options, residual, shrink)
		if err != nil {
			return err
		}
	}

	if img.Options.Force || residual != 0 {
		err = img.vipsResize( residual, img.Options.Interpolator )
		if err != nil {
			return err
		}
	}

	if img.Options.Force {
		img.Options.Crop = false
		img.Options.Embed = false
	}

	i, err := img.extractOrEmbedImage(img.Options)
	if err != nil {
		return err
	}
	/**
	 * We've probably ended up with a new image because we've taken part of the old out or otherwise transformed it
	 * SmartCrop is the anomaly here as it does modify the source image directly.
	 * TODO: Looking at making smartcrop return the image portion as well
	 */
	if i != nil && len(i.Buffer) > 0 {
//		fmt.Println("New Image Buffer from transform")
		C.g_object_unref(C.gpointer(img.Image))
		img.Image = i.Image
		img.Buffer = i.Buffer
		// This may go wrong if re unref it to soon?
		defer i.DecrementReferenceCount()
	}

	return nil
}

func (img *VipsImage) applyEffects() error {
	var err error

	if img.Options.GaussianBlur.Sigma > 0 || img.Options.GaussianBlur.MinAmpl > 0 {
		err = img.vipsGaussianBlur(img.Options.GaussianBlur)
		if err != nil {
			return err
		}
	}

	if img.Options.Sharpen.Sigma > 0 && img.Options.Sharpen.Y2 > 0 || img.Options.Sharpen.Y3 > 0 {
		err = img.vipsSharpen(img.Options.Sharpen)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
 * TODO: Make crop & embed work with relative numbers
 */
func (img *VipsImage) extractOrEmbedImage(o Options) (*VipsImage, error) {
	var err error = nil
	inWidth := int(img.Image.Xsize)
	inHeight := int(img.Image.Ysize)

	var image *VipsImage = nil

	switch {
	case o.Gravity == GravitySmart, o.SmartCrop:
		err = img.vipsSmartCrop(o.Width, o.Height)
		break
	case o.Crop:
		width := int(math.Min(float64(inWidth), float64(o.Width)))
		height := int(math.Min(float64(inHeight), float64(o.Height)))
		left, top := calculateCrop(inWidth, inHeight, o.Width, o.Height, o.Gravity)
		left, top = int(math.Max(float64(left), 0)), int(math.Max(float64(top), 0))
		image, err = img.vipsExtract(float32(left), float32(top), float32(width), float32(height))
		break
	case o.Embed:
		left, top := (o.Width-inWidth)/2, (o.Height-inHeight)/2
		err = img.vipsEmbed(left, top, o.Width, o.Height, o.Extend, o.Background)
		image = nil
		break
	case o.Trim:
		left, top, width, height, err := img.vipsTrim(o.Background, o.Threshold)
		if err == nil {
			image, err = img.vipsExtract(float32(left), float32(top), float32(width), float32(height))
		}
		break
	case o.Extract.Top != 0 || o.Extract.Left != 0 || o.Extract.Width != 0 || o.Extract.Height != 0:
		if o.Extract.Width == 0 {
			o.Extract.Width = float32(o.Width)
		}
		if o.Extract.Height == 0 {
			o.Extract.Height = float32(o.Height)
		}
		if o.Extract.Width == 0 || o.Extract.Height == 0 {
			return nil, errors.New("extract area width/height params are required")
		}
		image, err = img.vipsExtract(o.Extract.Left, o.Extract.Top, o.Extract.Width, o.Extract.Height)
		break
	}

	return image, err
}

func (img *VipsImage) rotateAndFlipImage(additive bool) (bool, error) {
	var err error
	var rotated bool

	rotation, flip, err := img.calculateRotationAndFlip(additive)
	if err != nil { return false, err }

	if img.Options.NoAutoRotate == false {
		// Cancel out the EXIF flip if user has requested flip
		if flip && img.Options.Flip {
			img.Options.Flip = false
		}
		img.Options.Rotate = rotation
	}

	if img.Options.Rotate > 0 {
		rotated = true
		//err = img.vipsRotate(getAngle(img.Options.Rotate))
		err = img.vipsRotate(img.Options.Rotate)
	}

	if img.Options.Flip {
		rotated = true
		err = img.vipsFlip(Vertical)
	}

	if img.Options.Flop {
		rotated = true
		err = img.vipsFlip(Horizontal)
	}
	return rotated, err
}

func (img *VipsImage) watermarkWithText() error {
	w := img.Options.Watermark
	if w.Text == "" {
		return nil
	}

	// Defaults
	if w.Font == "" {
		w.Font = WatermarkFont
	}
	if w.Width == 0 {
		w.Width = int(math.Floor(float64(img.Image.Xsize / 6)))
	}
	if w.DPI == 0 {
		w.DPI = 150
	}
	if w.Margin == 0 {
		w.Margin = w.Width
	}

	var err error
	err = img.vipsWatermark(w)
	if err != nil {
		return err
	}

	return nil
}

func (img *VipsImage) watermarkWithImage() error {
	w := img.Options.WatermarkImage

	if len(w.Buf) == 0 {
		return nil
	}

	if w.Opacity == 0.0 {
		w.Opacity = 1.0
	}

	var err error
	err = img.vipsDrawWatermark(w)

	if err != nil {
		return err
	}

	return nil
}

func (img *VipsImage) Flatten() error {
	var err error
	// Only PNG images are supported for now
	if img.Type != PNG || img.Options.Background == ColorBlack {
		return nil
	}
	err = img.vipsFlattenBackground(img.Options.Background)
	return err
}

func (img *VipsImage) zoomImage() error {
	if img.Options.Zoom != 0 {
		var err error
		err = img.vipsZoom(img.Options.Zoom+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func (img *VipsImage) shrinkImage(o Options, residual float64, shrink int) (float64, error) {
	// Use vips_shrink with the integral reduction
	err := img.vipsShrink(shrink)
	if err != nil {
		return 0, err
	}

	// Recalculate residual float based on dimensions of required vs shrunk images
	residualx := float64(o.Width) / float64(img.Image.Xsize)
	residualy := float64(o.Height) / float64(img.Image.Ysize)

	if o.Crop {
		residual = math.Max(residualx, residualy)
	} else {
		residual = math.Min(residualx, residualy)
	}

	return residual, nil
}

func (img *VipsImage) shrinkOnLoad() (float64, error) {
	var err error

	shrink := img.calculateShrink()
	factor := img.ScaleFactor()

	// Reload input using shrink-on-load
	if img.Type == JPEG && shrink >= 2 {
		shrinkOnLoad := 1
		// Recalculate integral shrink and double residual
		switch {
		case shrink >= 8:
			factor = factor / 8
			shrinkOnLoad = 8
		case shrink >= 4:
			factor = factor / 4
			shrinkOnLoad = 4
		case shrink >= 2:
			factor = factor / 2
			shrinkOnLoad = 2
		}

		err = img.vipsShrinkJpeg(shrinkOnLoad)
	} else if img.Type == WEBP && shrink >= 2 {
		err = img.vipsShrinkWebp(shrink)
	}

	return factor, err
}

func (img *VipsImage) ScaleFactor() float64 {

	if !img.Options.Enlarge && !img.Options.Force && int(img.Image.Xsize) < img.Options.Width && int(img.Image.Ysize) < img.Options.Height {
		return 1.0
	}

	o := &img.Options
	inWidth := int(img.Image.Xsize)
	inHeight := int(img.Image.Ysize)

	factor := 1.0
	xfactor := float64(inWidth) / float64(o.Width)
	yfactor := float64(inHeight) / float64(o.Height)

	switch {
	// Fixed width and height
	case o.Width > 0 && o.Height > 0:
		if o.Crop {
			factor = math.Min(xfactor, yfactor)
		} else {
			factor = math.Max(xfactor, yfactor)
		}
	// Fixed width, auto height
	case o.Width > 0:
		if o.Crop {
			o.Height = inHeight
		} else {
			factor = xfactor
			o.Height = roundFloat(float64(inHeight) / factor)
		}
	// Fixed height, auto width
	case o.Height > 0:
		if o.Crop {
			o.Width = inWidth
		} else {
			factor = yfactor
			o.Width = roundFloat(float64(inWidth) / factor)
		}
	// Identity transform
	default:
		o.Width = inWidth
		o.Height = inHeight
		break
	}

	return factor
}

func roundFloat(f float64) int {
	if f < 0 {
		return int(math.Ceil(f - 0.5))
	}
	return int(math.Floor(f + 0.5))
}

func calculateCrop(inWidth, inHeight, outWidth, outHeight int, gravity Gravity) (int, int) {
	left, top := 0, 0

	switch gravity {
	case GravityNorth:
		left = (inWidth - outWidth + 1) / 2
	case GravityEast:
		left = inWidth - outWidth
		top = (inHeight - outHeight + 1) / 2
	case GravitySouth:
		left = (inWidth - outWidth + 1) / 2
		top = inHeight - outHeight
	case GravityWest:
		top = (inHeight - outHeight + 1) / 2
	default:
		left = (inWidth - outWidth + 1) / 2
		top = (inHeight - outHeight + 1) / 2
	}

	return left, top
}

// calculateRotationAndFlip works out the angles and flips needed to get an image to look "normal" based on EXIF
// metadata for orientation.
// If an angle is specified in the image Options then it will use that.
// If additive is true, it will add the EXIF auto rotation and specified angle together, which is what end users would
// probably expect to happen.
func (img *VipsImage) calculateRotationAndFlip(additive bool) (Angle, bool, error) {
	angle := img.Options.Rotate
	rotate := D0
	flip := false

	if angle > 0 && !additive {
		return rotate, flip, nil
	}

	o, err := img.vipsExifOrientation()
	if err != nil { return D0, false, err }

	switch o {
	case 6:
		rotate = D90
		break
	case 3:
		rotate = D180
		break
	case 8:
		rotate = D270
		break
	case 2:
		flip = true
		break // flip 1
	case 7:
		flip = true
		rotate = D270
		break // flip 6
	case 4:
		flip = true
		rotate = D180
		break // flip 3
	case 5:
		flip = true
		rotate = D90
		break // flip 8
	}

	if additive { rotate+= angle }

	return rotate, flip, nil
}

func (img *VipsImage) calculateShrink() int {

	if !img.Options.Enlarge && !img.Options.Force && int(img.Image.Xsize) < img.Options.Width && int(img.Image.Ysize) < img.Options.Height {
		return 1
	}

	var shrink float64
	factor := img.ScaleFactor()

	// Calculate integral box shrink
	windowSize := vipsWindowSize(img.Options.Interpolator.String())
	if factor >= 2 && windowSize > 3 {
		// Shrink less, affine more with interpolators that use at least 4x4 pixel window, e.g. bicubic
		shrink = float64(math.Floor(factor * 3.0 / windowSize))
	} else {
		shrink = math.Floor(factor)
	}

	return int(math.Max(shrink, 1))
}

func (img *VipsImage) calculateResidual() float64 {
	if !img.Options.Enlarge && !img.Options.Force && int(img.Image.Xsize) < img.Options.Width && int(img.Image.Ysize) < img.Options.Height {
		return 0
	}

	return float64(img.calculateShrink()) / img.ScaleFactor()
}
/*
func getAngle(angle Angle) Angle {
	divisor := angle % 90
	if divisor != 0 {
		angle = angle - divisor
	}
	return Angle(math.Min(float64(angle), 270))
}
*/

func (img *VipsImage) applyGamma() error {
	var err error
	if o.Gamma > 0 {
		err = img.vipsGamma(img.Options.Gamma)
		if err != nil {
			return err
		}
	}
	return nil
}