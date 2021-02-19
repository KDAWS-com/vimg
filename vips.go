package vimg

/*
#cgo pkg-config: vips
#cgo CFLAGS: -g3 -O3
#cgo LDFLAGS: -lm
#include "vips.h"
*/
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// VipsVersion exposes the current libvips semantic version
const VipsVersion = string(C.VIPS_VERSION)

// VipsMajorVersion exposes the current libvips major version number
const VipsMajorVersion = int(C.VIPS_MAJOR_VERSION)

// VipsMinorVersion exposes the current libvips minor version number
const VipsMinorVersion = int(C.VIPS_MINOR_VERSION)

const (
	maxCacheMem  = 250 * 1024 * 1024
	maxCacheSize = 1000
)

type Blob int

const (
	VIPS_META_EXIF_NAME Blob = iota
	VIPS_META_XMP_NAME
	VIPS_META_IPTC_NAME
	VIPS_META_PHOTOSHOP_NAME
	VIPS_META_ICC_NAME
	VIPS_META_IMAGEDESCRIPTION
	VIPS_META_RESOLUTION_UNIT
	VIPS_META_ORIENTATION
	VIPS_META_PAGE_HEIGHT
	VIPS_META_N_PAGES
)

var blobsFromString = map[string]Blob {
	"exif-data":			VIPS_META_EXIF_NAME,
	"xmp-data": 			VIPS_META_XMP_NAME,
	"iptc-data":			VIPS_META_IPTC_NAME,
	"photoshop-data": 		VIPS_META_PHOTOSHOP_NAME,
	"icc-profile-data": 	VIPS_META_ICC_NAME,
	"image-description": 	VIPS_META_IMAGEDESCRIPTION,
	"resolution-unit":		VIPS_META_RESOLUTION_UNIT,
	"orientation":			VIPS_META_ORIENTATION,
	"page-height":			VIPS_META_PAGE_HEIGHT,
	"n-pages":				VIPS_META_N_PAGES,
}

var blobs = map[Blob]string {
	VIPS_META_EXIF_NAME: 		"exif-data",
	VIPS_META_XMP_NAME: 		"xmp-data",
	VIPS_META_IPTC_NAME: 		"iptc-data",
	VIPS_META_PHOTOSHOP_NAME: 	"photoshop-data",
	VIPS_META_ICC_NAME: 		"icc-profile-data",
	VIPS_META_IMAGEDESCRIPTION: "image-description",
	VIPS_META_RESOLUTION_UNIT: 	"resolution-unit",
	VIPS_META_ORIENTATION: 		"orientation",
	VIPS_META_PAGE_HEIGHT:		"page-height",
	VIPS_META_N_PAGES:			"n-pages",
}

var blobToCString = map[Blob]*C.char {
	VIPS_META_EXIF_NAME: 		C.CString("exif-data"),
	VIPS_META_XMP_NAME: 		C.CString("xmp-data"),
	VIPS_META_IPTC_NAME: 		C.CString("iptc-data"),
	VIPS_META_PHOTOSHOP_NAME: 	C.CString("photoshop-data"),
	VIPS_META_ICC_NAME: 		C.CString("icc-profile-data"),
	VIPS_META_IMAGEDESCRIPTION: C.CString("image-description"),
	VIPS_META_RESOLUTION_UNIT: 	C.CString("resolution-unit"),
	VIPS_META_ORIENTATION: 		C.CString("orientation"),
	VIPS_META_PAGE_HEIGHT:		C.CString("page-height"),
	VIPS_META_N_PAGES:			C.CString("n-pages"),
}

func (b Blob) String() string {
	return blobs[b]
}

func (b Blob) CString() *C.char {
	return blobToCString[b]
}

var (
	m           sync.Mutex
	initialized bool
)

// VipsMemoryInfo represents the memory stats provided by libvips.
type VipsMemoryInfo struct {
	Memory          int64
	MemoryHighwater int64
	Allocations     int64
}

// vipsSaveOptions represents the internal option used to talk with libvips.
type vipsSaveOptions struct {
	Quality        int
	Compression    int
	Type           ImageType
	Interlace      bool
	NoProfile      bool
	StripMetadata  bool
	Lossless       bool
	OutputICC      string // Absolute path to the output ICC profile
	Interpretation Interpretation
	Progressive    bool
}

type vipsWatermarkOptions struct {
	Width       C.int
	DPI         C.int
	NoReplicate C.int
	Background  [4]C.double
	Relative		C.int
	HOffset	    	C.double
	VOffset 		C.double
	HAlign			C.int
	VAlign			C.int
}

type vipsWatermarkImageOptions struct {
	Left    C.int
	Top     C.int
	Opacity C.float
	Blend	C.int
}

type vipsWatermarkTextOptions struct {
	Text *C.char
	Font *C.char
	Align C.int
}

func init() {
	Initialize()
}

// Initialize is used to explicitly start libvips in thread-safe way.
// Only call this function if you have previously turned off libvips.
func Initialize() {
	if C.VIPS_MAJOR_VERSION <= 7 && C.VIPS_MINOR_VERSION < 40 {
		panic("unsupported libvips version!")
	}

	m.Lock()
	runtime.LockOSThread()
	defer m.Unlock()
	defer runtime.UnlockOSThread()

	err := C.vips_init(C.CString("vimg"))
	if err != 0 {
		panic("unable to start vips!")
	}

	// Set libvips cache params
	C.vips_cache_set_max_mem(maxCacheMem)
	C.vips_cache_set_max(maxCacheSize)

	// Define a custom thread concurrency limit in libvips (this may generate thread-unsafe issues)
	// See: https://github.com/jcupitt/libvips/issues/261#issuecomment-92850414
	if os.Getenv("VIPS_CONCURRENCY") == "" {
		C.vips_concurrency_set(1)
	}

	// Enable libvips cache tracing
	if os.Getenv("VIPS_TRACE") != "" {
		C.vips_enable_cache_set_trace()
	}

	initialized = true
}

// Shutdown is used to shutdown libvips in a thread-safe way.
// You can call this to drop caches as well.
// If libvips was already initialized, the function is no-op
func Shutdown() {
	m.Lock()

	C.free(unsafe.Pointer(imageInterpolatorToCString[Bicubic]))
	C.free(unsafe.Pointer(imageInterpolatorToCString[Bilinear]))
	C.free(unsafe.Pointer(imageInterpolatorToCString[Nohalo]))
	C.free(unsafe.Pointer(imageInterpolatorToCString[Nearest]))

	C.free(unsafe.Pointer(blobToCString[VIPS_META_EXIF_NAME]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_XMP_NAME]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_IPTC_NAME]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_PHOTOSHOP_NAME]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_ICC_NAME]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_IMAGEDESCRIPTION]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_RESOLUTION_UNIT]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_ORIENTATION]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_PAGE_HEIGHT]))
	C.free(unsafe.Pointer(blobToCString[VIPS_META_N_PAGES]))

	if initialized {
		C.vips_shutdown()
		initialized = false
	}
	m.Unlock()
}

// VipsCacheSetMaxMem Sets the maximum amount of tracked memory allowed before the vips operation cache
// begins to drop entries.
func VipsCacheSetMaxMem(maxCacheMem int) {
	C.vips_cache_set_max_mem(C.size_t(maxCacheMem))
}

// VipsCacheSetMax sets the maximum number of operations to keep in the vips operation cache.
func VipsCacheSetMax(maxCacheSize int) {
	C.vips_cache_set_max(C.int(maxCacheSize))
}

// VipsCacheDropAll drops the vips operation cache, freeing the allocated memory.
func VipsCacheDropAll() {
	C.vips_cache_drop_all()
}

// VipsDebugInfo outputs to stdout libvips collected data. Useful for debugging.
func VipsDebugInfo() {
	C.im__print_all()
}

// VipsMemory gets memory info stats from libvips (cache size, memory allocs...)
func VipsMemory() VipsMemoryInfo {
	return VipsMemoryInfo{
		Memory:          int64(C.vips_tracked_get_mem()),
		MemoryHighwater: int64(C.vips_tracked_get_mem_highwater()),
		Allocations:     int64(C.vips_tracked_get_allocs()),
	}
}

// VipsIsTypeSupported returns true if the given image type
// is supported by the current libvips compilation.
func VipsIsTypeSupported(t ImageType) bool {
	if t == JPEG {
		return int(C.vips_type_find_bridge(C.JPEG)) != 0
	}
	if t == WEBP {
		return int(C.vips_type_find_bridge(C.WEBP)) != 0
	}
	if t == PNG {
		return int(C.vips_type_find_bridge(C.PNG)) != 0
	}
	if t == GIF {
		return int(C.vips_type_find_bridge(C.GIF)) != 0
	}
	if t == PDF {
		return int(C.vips_type_find_bridge(C.PDF)) != 0
	}
	if t == SVG {
		return int(C.vips_type_find_bridge(C.SVG)) != 0
	}
	if t == TIFF {
		return int(C.vips_type_find_bridge(C.TIFF)) != 0
	}
	if t == MAGICK {
		return int(C.vips_type_find_bridge(C.MAGICK)) != 0
	}
	return false
}

// VipsIsTypeSupportedSave returns true if the given image type
// is supported by the current libvips compilation for the
// save operation.
func VipsIsTypeSupportedSave(t ImageType) bool {
	if t == JPEG {
		return int(C.vips_type_find_save_bridge(C.JPEG)) != 0
	}
	if t == WEBP {
		return int(C.vips_type_find_save_bridge(C.WEBP)) != 0
	}
	if t == PNG {
		return int(C.vips_type_find_save_bridge(C.PNG)) != 0
	}
	if t == TIFF {
		return int(C.vips_type_find_save_bridge(C.TIFF)) != 0
	}
	return false
}

func (img *VipsImage) vipsExifOrientation() (int, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return 0, ErrVipsImageNotValidPointer
	}
	return int(C.vips_exif_orientation(img.Image)), nil
}

func (img *VipsImage) vipsHasAlpha() (bool, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return false, ErrVipsImageNotValidPointer
	}
	return int(C.has_alpha_channel(img.Image)) > 0, nil
}

func (img *VipsImage) hasProfile() (bool, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return false, ErrVipsImageNotValidPointer
	}
	return int(C.has_profile_embed(img.Image)) > 0, nil
}

func vipsWindowSize(name string) float64 {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return float64(C.interpolator_window_size(cname))
}

func (img *VipsImage) vipsSpace() (string, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return "", ErrVipsImageNotValidPointer
	}
	return C.GoString(C.vips_enum_nick_bridge(img.Image)), nil
}

func (img *VipsImage) vipsRotate(angle Angle) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"rotate"}).Inc()

	var image *C.VipsImage
//	err := C.vips_rotate_vimg(img.Image, &image, C.double(angle))
	err := C.vips_rotate_fill(img.Image, &image, C.double(angle), C.double(img.Options.Background.R), C.double(img.Options.Background.G), C.double(img.Options.Background.B), C.double(img.Options.Background.A))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsFlip(direction Direction) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"flip"}).Inc()

	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	err := C.vips_flip_bridge(img.Image, &image, C.int(direction))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsZoom(zoom int) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"zoom"}).Inc()

	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	err := C.vips_zoom_bridge(img.Image, &image, C.int(zoom), C.int(zoom))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsWatermark(w Watermark) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"watermark_text"}).Inc()

	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	// Defaults
	noReplicate := 0
	if w.NoReplicate {
		noReplicate = 1
	}

	o := img.Options.Watermark

	text := C.CString(w.Text)
	font := C.CString(w.Font)
	background := [4]C.double{C.double(w.Background.R), C.double(w.Background.G), C.double(w.Background.B), C.double(w.Background.A)}
//fmt.Printf("Background: %+v\n", background)
	var relative int
	if w.Relative { relative = 1 } else { relative = 0 }

	textOpts := vipsWatermarkTextOptions{text, font, C.int(o.TextAlign)}
	opts := vipsWatermarkOptions{C.int(w.Width), C.int(w.DPI), C.int(noReplicate), background, C.int(relative), C.double(o.HOffset), C.double(o.VOffset), C.int(o.HAlign), C.int(o.VAlign)}
//fmt.Printf("X,Y: %+v, %+v\n", img.Image.Xsize, img.Image.Ysize)
//fmt.Printf("Watermark: %+v\n", w)
//fmt.Printf("Watermark Text: %+v\n", textOpts)
//fmt.Printf("Watermark Options: %+v\n", opts)
	err := C.vips_watermark(img.Image, &image, (*C.WatermarkTextOptions)(unsafe.Pointer(&textOpts)), (*C.WatermarkOptions)(unsafe.Pointer(&opts)))

	C.free(unsafe.Pointer(text))
	C.free(unsafe.Pointer(font))

	if err != 0 {
//		fmt.Printf("Watermark Error: %+v\n", err)
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsRead(buf *bytes.Buffer) error {
	// No pointer check as this might be first call

	//m.Lock()
	//defer m.Unlock()

	img.Buffer = buf.Bytes()
	imageType := vipsImageType(img.Buffer)
	if imageType == UNKNOWN {
		return errors.New("Unsupported image format")
	}

	var image *C.VipsImage
	length := C.size_t(len(img.Buffer))
	imageBuf := unsafe.Pointer(&img.Buffer[0])
	err := C.vips_init_image(imageBuf, length, C.int(imageType), &image)
	defer func() {
		C.vips_thread_shutdown()
		C.vips_error_clear()
		//C.g_object_unref(C.gpointer(img.Image))
	}()

	if err != 0 {
		img.Buffer = nil
		//C.g_object_unref(C.gpointer(imageBuf))
		return catchVipsError()
	}

	if !reflect.ValueOf(img.Image).IsNil() {
		C.g_object_unref(C.gpointer(img.Image))
	}

	img.Image = image
	img.Type = imageType
//	img.Buffer = buf

	//C.g_object_unref(C.gpointer(imageBuf))

	return nil
}
/*
func vipsColourspaceIsSupportedBuffer(buf []byte) (bool, error) {
//	image, _, err := vipsRead(buf)
	image, err := NewVipsImage(buf, Options{})

	if err != nil {
		return false, err
	}
	C.g_object_unref(C.gpointer(image))
	return image.vipsColourspaceIsSupported(), nil
}
*/
func (img *VipsImage) vipsColourspaceIsSupported() (bool, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return false, ErrVipsImageNotValidPointer
	}
	return int(C.vips_colourspace_issupported_bridge(img.Image)) == 1, nil
}
/*
func vipsInterpretationBuffer(buf []byte) (Interpretation, error) {
	//image, _, err := vipsRead(buf)
	image, err := NewVipsImage(buf, Options{})
	if err != nil {
		return InterpretationError, err
	}
	C.g_object_unref(C.gpointer(image))
	return image.vipsInterpretation(), nil
}
*/
func (img *VipsImage) vipsInterpretation() (Interpretation, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return InterpretationError, ErrVipsImageNotValidPointer
	}
	return Interpretation(C.vips_image_guess_interpretation_bridge(img.Image)), nil
}

func (img *VipsImage) vipsFlattenBackground(background Color) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"flatten"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	backgroundC := [4]C.double{
		C.double(background.R),
		C.double(background.G),
		C.double(background.B),
		C.double(background.A),
	}

	if alpha, e := img.vipsHasAlpha(); alpha && e == nil {
		err := C.vips_flatten_background_brigde(img.Image, &image, backgroundC[0], backgroundC[1], backgroundC[2], backgroundC[3])
		if int(err) != 0 {
			return catchVipsError()
		}
		C.g_object_unref(C.gpointer(img.Image))
		img.Image = image
	}

	return nil
}

func (img *VipsImage) vipsBlob(name Blob) (*[]byte, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return nil, ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"blob"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	defer C.vips_error_clear()

	length := C.size_t(0)
	blobErr := C.int(0)

	/**
	 * Do not be tempted to free this ptr via:
	 *
	 * C.g_free(C.gpointer(ptr))
	 *
	 * The blob is part of the image data, you will free a hole of memory in the middle of the image.
	 * When the image is freed, you'll then end up freeing the hole of memory again, which may already
	 * be in use, causing memory corruption and ultimately a segfault and/or system instability.
	 */
	var ptr unsafe.Pointer
	/**
	 * See above warning on ptr
	 */

	blobErr = C.vips_image_get_blob_bridge(img.Image, &ptr, &length, name.CString())

	if int(blobErr) != 0 {
		return nil, catchVipsError()
	}

	buf := C.GoBytes(ptr, C.int(length))
	return &buf, nil
}

func (img *VipsImage) vipsPreSave(o *vipsSaveOptions) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage
	// Remove ICC profile metadata
	if o.NoProfile {
		C.remove_profile(img.Image)
	}

	// Use a default interpretation and cast it to C type
	if o.Interpretation == 0 {
		o.Interpretation = InterpretationSRGB
	}
	interpretation := C.VipsInterpretation(o.Interpretation)
	// Apply the proper colour space
	space, err := img.vipsColourspaceIsSupported()
	if err != nil {
		return err
	}

	if space {
		err := C.vips_colourspace_bridge(img.Image, &image, interpretation)
		if int(err) != 0 {
			return catchVipsError()
		}
		C.g_object_unref(C.gpointer(img.Image))
		img.Image = image
	}

	hasProfile, err := img.hasProfile()
	if err != nil {
		return err
	}

	if o.OutputICC != "" && hasProfile {
		outputIccPath := C.CString(o.OutputICC)
		defer C.free(unsafe.Pointer(outputIccPath))
		err := C.vips_icc_transform_bridge(img.Image, &image, outputIccPath)
		if int(err) != 0 {
			return catchVipsError()
		}
		C.g_object_unref(C.gpointer(img.Image))
		img.Image = image
	}
	return nil
}

func (img *VipsImage) vipsSave(o vipsSaveOptions) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"save"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var ptr unsafe.Pointer

	err := img.vipsPreSave(&o)
	if err != nil {
		return err
	}

	// When an image has an unsupported color space, vipsPreSave
	// returns the pointer of the image passed to it unmodified.
	// When this occurs, we must take care to not dereference the
	// original image a second time; we may otherwise erroneously
	// free the object twice.
/*	if tmpImage != img {
		defer C.g_object_unref(C.gpointer(tmpImage))
	}*/

	length := C.size_t(0)
	saveErr := C.int(0)
	interlace := C.int(boolToInt(o.Interlace))
	quality := C.int(o.Quality)
	strip := C.int(boolToInt(o.StripMetadata))
	lossless := C.int(boolToInt(o.Lossless))

	if o.Type != 0 && !IsTypeSupportedSave(o.Type) {
		return fmt.Errorf("VIPS cannot save to %#v", ImageTypes[o.Type])
	}
/*
	switch o.Type {
	case WEBP:
		saveErr = C.vips_webpsave_bridge(img.Image, &ptr, &length, strip, quality, lossless)
	case PNG:
		saveErr = C.vips_pngsave_bridge(img.Image, &ptr, &length, strip, C.int(o.Compression), quality, interlace)
	case TIFF:
		saveErr = C.vips_tiffsave_bridge(img.Image, &ptr, &length)
	default:
		saveErr = C.vips_jpegsave_bridge(img.Image, &ptr, &length, strip, quality, interlace)
	}

	if int(saveErr) != 0 {
		C.g_free(C.gpointer(ptr))
		return catchVipsError()
	}

	buf := C.GoBytes(ptr, C.int(length))
	err = img.vipsRead(buf)
	if err != nil {
		C.g_free(C.gpointer(ptr))
		C.vips_error_clear()
		return err
	}

	C.g_free(C.gpointer(ptr))*/

	switch o.Type {
	case WEBP:
		saveErr = C.vips_webpsave_bridge(img.Image, &ptr, &length, strip, quality, lossless)
	case PNG:
		saveErr = C.vips_pngsave_bridge(img.Image, &ptr, &length, strip, C.int(o.Compression), quality, interlace)
	case TIFF:
		saveErr = C.vips_tiffsave_bridge(img.Image, &ptr, &length)
	default:
		saveErr = C.vips_jpegsave_bridge(img.Image, &ptr, &length, strip, quality, interlace)
	}
	if int(saveErr) != 0 {
		C.g_free(C.gpointer(ptr))
		return catchVipsError()
	}

	buf := C.GoBytes(ptr, C.int(length))
	img.Buffer = buf
	C.g_object_unref(C.gpointer(img.Image))
	C.g_free(C.gpointer(ptr))

	return nil
}

func (img *VipsImage) getImageBuffer() ([]byte, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return nil, ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"getbuffer"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var ptr unsafe.Pointer
	length := C.size_t(0)
	interlace := C.int(0)
	quality := C.int(100)

	err := C.int(0)
	switch img.Type {
	case WEBP:
		err = C.vips_webpsave_bridge(img.Image, &ptr, &length, 0, quality, 1)
	case PNG:
		err = C.vips_pngsave_bridge(img.Image, &ptr, &length, 0, 0, quality, interlace)
	case TIFF:
		err = C.vips_tiffsave_bridge(img.Image, &ptr, &length)
	default:
		err = C.vips_jpegsave_bridge(img.Image, &ptr, &length, 0, quality, interlace)
	}
	if int(err) != 0 {
		C.g_free(C.gpointer(ptr))
		return nil, catchVipsError()
	}

	buf := C.GoBytes(ptr, C.int(length))

	img.Buffer = buf
	C.g_free(C.gpointer(ptr))
	return buf, nil
}

func (img *VipsImage) vipsExtract(left, top, width, height float32) (*VipsImage, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return nil, ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"extract"}).Inc()
	//m.Lock()
	//defer m.Unlock()
	var image *C.VipsImage

	if width > MaxSize || height > MaxSize {
		return nil, errors.New("Maximum image size exceeded")
	}

	srcX := float32(img.Image.Xsize)
	srcY := float32(img.Image.Ysize)

	if img.Options.Extract.Relative {
		top = srcY * (top/100)
		left = srcX * (left/100)
		height = srcY * (height/100)
		width = srcX * (width/100)
	}

	err := C.vips_extract_area_bridge(img.Image, &image, C.int(left), C.int(top), C.int(width), C.int(height))
	if err != 0 {
		return nil, catchVipsError()
	}

	var e error
	i := AquireVipsImage()
	i.Image = image
	i.Type = JPEG
	i.Options = Options{}
	i.Buffer, e = i.getImageBuffer()
	if e != nil {
		return nil, e
	}
	return i, nil
}

func (img *VipsImage) vipsSmartCrop(width, height int) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"smartcrop"}).Inc()
	//m.Lock()
	//defer m.Unlock()
	var image *C.VipsImage

	if width > MaxSize || height > MaxSize {
		return errors.New("Maximum image size exceeded")
	}

	err := C.vips_smartcrop_bridge(img.Image, &image, C.int(width), C.int(height))
	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsTrim(background Color, threshold float64) (int, int, int, int, error) {
	if reflect.ValueOf(img.Image).IsNil() {
		return 0, 0, 0, 0,ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"trim"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	top := C.int(0)
	left := C.int(0)
	width := C.int(0)
	height := C.int(0)

	err := C.vips_find_trim_bridge(img.Image,
		&top, &left, &width, &height,
		C.double(background.R), C.double(background.G), C.double(background.B),
		C.double(threshold))
	if err != 0 {
		return 0, 0, 0, 0, catchVipsError()
	}

	return int(top), int(left), int(width), int(height), nil
}

func (img *VipsImage) vipsShrinkJpeg(shrink int) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"shrink_jpeg"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	var ptr = unsafe.Pointer(&img.Buffer[0])

	err := C.vips_jpegload_buffer_shrink(ptr, C.size_t(len(img.Buffer)), &image, C.int(shrink))
	if err != 0 {
		//C.g_free(C.gpointer(ptr))
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsShrinkWebp(shrink int) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"shrink_webp"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage
	var ptr = unsafe.Pointer(&img.Buffer[0])

	err := C.vips_webpload_buffer_shrink(ptr, C.size_t(len(img.Buffer)), &image, C.int(shrink))
	if err != 0 {
		//C.g_free(C.gpointer(ptr))
		return catchVipsError()
	}

	//C.g_free(C.gpointer(ptr))
	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsShrink(shrink int) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"shrink"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	err := C.vips_shrink_bridge(img.Image, &image, C.double(float64(shrink)), C.double(float64(shrink)))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsResize(scale float64, i Interpolator) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"resize"}).Inc()
	//m.Lock()
	//defer m.Unlock()
	var image *C.VipsImage

	interpolator := C.vips_interpolate_new(i.CString())

	err := C.vips_resize_bridge(img.Image, &image, C.double(scale), interpolator)

	C.g_object_unref(C.gpointer(interpolator))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image
	return nil
}

func (img *VipsImage) vipsReduce(xshrink float64, yshrink float64) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"reduce"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	err := C.vips_reduce_bridge(img.Image, &image, C.double(xshrink), C.double(yshrink))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsEmbed(left, top, width, height int, extend Extend, background Color) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"embed"}).Inc()
	//m.Lock()
	//defer m.Unlock()


	var image *C.VipsImage

	// Max extend value, see: https://jcupitt.github.io/libvips/API/current/libvips-conversion.html#VipsExtend
	if extend > 5 {
		extend = ExtendBackground
	}

	err := C.vips_embed_bridge(img.Image, &image, C.int(left), C.int(top), C.int(width),
		C.int(height), C.int(extend), C.double(background.R), C.double(background.G), C.double(background.B))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsAffine(residualx, residualy float64, i Interpolator) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"affine"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage
	interpolator := C.vips_interpolate_new(i.CString())

	err := C.vips_affine_interpolator(img.Image, &image, C.double(residualx), 0, 0, C.double(residualy), interpolator)

	C.g_object_unref(C.gpointer(interpolator))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func vipsImageType(buf []byte) ImageType {
	if len(buf) < 12 {
		return UNKNOWN
	}
	if buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return JPEG
	}
	if IsTypeSupported(GIF) && buf[0] == 0x47 && buf[1] == 0x49 && buf[2] == 0x46 {
		return GIF
	}
	if buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		return PNG
	}
	if IsTypeSupported(TIFF) &&
		((buf[0] == 0x49 && buf[1] == 0x49 && buf[2] == 0x2A && buf[3] == 0x0) ||
			(buf[0] == 0x4D && buf[1] == 0x4D && buf[2] == 0x0 && buf[3] == 0x2A)) {
		return TIFF
	}
	if IsTypeSupported(PDF) && buf[0] == 0x25 && buf[1] == 0x50 && buf[2] == 0x44 && buf[3] == 0x46 {
		return PDF
	}
	if IsTypeSupported(WEBP) && buf[8] == 0x57 && buf[9] == 0x45 && buf[10] == 0x42 && buf[11] == 0x50 {
		return WEBP
	}
	if IsTypeSupported(SVG) && IsSVGImage(buf) {
		return SVG
	}
	if IsTypeSupported(MAGICK) && strings.HasSuffix(readImageType(buf), "MagickBuffer") {
		return MAGICK
	}

	return UNKNOWN
}

func readImageType(buf []byte) string {
	length := C.size_t(len(buf))
	imageBuf := unsafe.Pointer(&buf[0])
	load := C.vips_foreign_find_load_buffer(imageBuf, length)
	return C.GoString(load)
}

func catchVipsError() error {
	s := C.GoString(C.vips_error_buffer())
	C.vips_error_clear()
	C.vips_thread_shutdown()
	return errors.New(s)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (img *VipsImage) vipsGaussianBlur(o GaussianBlur) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"blur"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	if o.Sigma == 0 { o.Sigma = 1 }

	var image *C.VipsImage

	err := C.vips_gaussblur_bridge(img.Image, &image, C.double(o.Sigma), C.double(o.MinAmpl))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsSharpen(o Sharpen) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"sharpen"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage

	err := C.vips_sharpen_bridge(img.Image, &image, C.double(o.Sigma), C.double(o.X1), C.double(o.Y2), C.double(o.Y3), C.double(o.M1), C.double(o.M2))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func max(x int) int {
	return int(math.Max(float64(x), 0))
}

func (img *VipsImage) vipsDrawWatermark(o WatermarkImage) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"watermark_image"}).Inc()

	srcX := float32(img.Image.Xsize)
	srcY := float32(img.Image.Ysize)

	var wmOptions Options
	var left float32
	var top float32
	if o.Relative {
		wmOptions.Width = int(srcX * (o.Width / 100))
		wmOptions.Force = false
	} else {
		wmOptions.Width = int(o.Width)
		wmOptions.Force = false
	}
	wmOptions.MaintainAspect = true
	wmOptions.Enlarge = false
	wmOptions.Background = ColorBlack
	var image *C.VipsImage
	br := bytes.NewBuffer(o.Buf)
	watermark, e := NewVipsImage(br, wmOptions)
	if watermark != nil {
		defer watermark.DecrementReferenceCount()
	}
	if e != nil {
		return e
	}
	e = watermark.Process()
	if e != nil {
		return e
	}

	wmX := float32(watermark.Image.Xsize)
	wmY := float32(watermark.Image.Ysize)

	if o.Relative {
		switch o.HAlign {
		case PositionLeft:
			left = (o.HOffset / 100) * srcX
			break
		case PositionRight:
			hOffset := (o.HOffset / 100) * srcX
			left = srcX - hOffset - wmX
			break
		case PositionCentre:
			left = (srcX - wmX) / 2
		}
		switch o.VAlign {
		case PositionTop:
			top = (o.VOffset / 100) * srcY
			break
		case PositionBottom:
			vOffset := (o.VOffset / 100) * srcY
			top = srcY - vOffset - wmY
			break
		case PositionCentre:
			top = (srcY - wmY) / 2
		}
	} else {
		switch o.HAlign {
		case PositionLeft:
			left = o.HOffset
			break
		case PositionRight:
			left = srcX - o.HOffset - wmX
			break
		case PositionCentre:
			left = (srcX - wmX) / 2
		}
		switch o.VAlign {
		case PositionTop:
			top = o.VOffset
			break
		case PositionBottom:
			top = srcY - o.VOffset - wmY
			break
		case PositionCentre:
			top = (srcY - wmY) / 2
		}
	}

	opts := vipsWatermarkImageOptions{C.int(left), C.int(top), C.float(o.Opacity), C.int(o.BlendMode)}

	err := C.vips_watermark_image(img.Image, watermark.Image, &image, (*C.WatermarkImageOptions)(unsafe.Pointer(&opts)))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}
/*
func (img *VipsImage) vipsDrawWatermark(o WatermarkImage) error {
	if reflect.ValueOf(img.Image).IsNil() {
		return ErrVipsImageNotValidPointer
	}
	vimgOperations.With(prometheus.Labels{"type":"watermark_image"}).Inc()
	//m.Lock()
	//defer m.Unlock()

	var image *C.VipsImage
	br := bytes.NewBuffer(o.Buf)
	watermark, e := NewVipsImage(br, Options{})
	if watermark != nil {
		defer watermark.DecrementReferenceCount()
	}

	if e != nil {
		return e
	}

	opts := vipsWatermarkImageOptions{C.int(o.Left), C.int(o.Top), C.float(o.Opacity)}

	err := C.vips_watermark_image(img.Image, watermark.Image, &image, (*C.WatermarkImageOptions)(unsafe.Pointer(&opts)))

	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}
*/

func (img *VipsImage) vipsGamma(Gamma float64) error {
	defer C.g_object_unref(C.gpointer(img.Image))

	var image *C.VipsImage

	err := C.vips_gamma_bridge(img.Image, &image, C.double(Gamma))
	if err != 0 {
		return catchVipsError()
	}

	C.g_object_unref(C.gpointer(img.Image))
	img.Image = image

	return nil
}

func (img *VipsImage) vipsExifStringTag(tag string) string {
	return vipsExifShort(C.GoString(C.vips_exif_tag(img.Image, C.CString(tag))))
}

func (img *VipsImage) vipsExifIntTag(tag string) int {
	return int(C.vips_exif_tag_to_int(img.Image, C.CString(tag)))
}

func vipsExifShort(s string) string {
	if strings.Contains(s, " (") {
		return s[:strings.Index(s, "(")-1]
	}
	return s
}