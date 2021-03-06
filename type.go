package vimg

import "C"
import (
	"encoding/json"
	"regexp"
	"sync"
	"unicode/utf8"
)

const (
	// UNKNOWN represents an unknow image type value.
	UNKNOWN ImageType = iota
	// JPEG represents the JPEG image type.
	JPEG
	// WEBP represents the WEBP image type.
	WEBP
	// PNG represents the PNG image type.
	PNG
	// TIFF represents the TIFF image type.
	TIFF
	// GIF represents the GIF image type.
	GIF
	// PDF represents the PDF type.
	PDF
	// SVG represents the SVG image type.
	SVG
	// MAGICK represents the libmagick compatible genetic image type.
	MAGICK
)

// ImageType represents an image type value.
type ImageType int

/**
 * TODO: If they change the enum order in libvips, this breaks. Need to get the enum correctly from C
 */
type BlendMode int
const (
	BlendClear BlendMode = iota
	BlendSource
	BlendOver
	BlendIn
	BlendOut
	BlendAtop
	BlendDest
	BlendDestOver
	BlendDestIn
	BlendDestOut
	BlendDestAtop
	BlendXor
	BlendAdd
	BlendSaturate
	BlendMultiply
	BlendScreen
	BlendOverlay
	BlendDarken
	BlendLighten
	BlendDodge
	BlendBurn
	BlendHard
	BlendSoft
	BlendDifference
	BlendExclusion
	BlendLast
)

var blendModeToID = map[string]BlendMode {
	"clear": BlendClear,
	"source": BlendSource,
	"over": BlendOver,
	"in": BlendIn,
	"out": BlendOut,
	"atop": BlendAtop,
	"dest": BlendDest,
	"dest_over": BlendDestOver,
	"dest_in": BlendDestIn,
	"dest_out": BlendDestOut,
	"dest_atop": BlendDestAtop,
	"xor": BlendXor,
	"add": BlendAdd,
	"saturate": BlendSaturate,
	"multiply": BlendMultiply,
	"screen": BlendScreen,
	"overlay": BlendOverlay,
	"darken": BlendDarken,
	"lighten": BlendLighten,
	"dodge": BlendDodge,
	"burn": BlendBurn,
	"soft": BlendSoft,
	"hard": BlendHard,
	"difference": BlendDifference,
	"exclusion": BlendExclusion,
	"last": BlendLast,
}

func (b *BlendMode) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*b = blendModeToID[s]
	return nil
}

var (
	htmlCommentRegex = regexp.MustCompile("(?i)<!--([\\s\\S]*?)-->")
	svgRegex         = regexp.MustCompile(`(?i)^\s*(?:<\?xml[^>]*>\s*)?(?:<!doctype svg[^>]*>\s*)?<svg[^>]*>[^*]*<\/svg>\s*$`)
)

// ImageTypes stores as pairs of image types supported and its alias names.
var ImageTypes = map[ImageType]string{
	JPEG:   "jpeg",
	PNG:    "png",
	WEBP:   "webp",
	TIFF:   "tiff",
	GIF:    "gif",
	PDF:    "pdf",
	SVG:    "svg",
	MAGICK: "magick",
}

var imageInterpolatorToID = map[string]Interpolator {
	"bicubic": Bicubic,
	"bilinear": Bilinear,
	"nohalo": Nohalo,
	"nearest": Nearest,
}

var imageInterpolatorToCString = map[Interpolator]*C.char {
	Bicubic: C.CString("bicubic"),
	Bilinear: C.CString("bilinear"),
	Nohalo: C.CString("nohalo"),
	Nearest: C.CString("nearest"),
}

var imageInterpretationToID = map[string]Interpretation {
	"srgb":			 	InterpretationSRGB,
	"multiband":	InterpretationMultiband,
	"bw":					InterpretationBW,
	"cmyk":				InterpretationCMYK,
	"rgb":				InterpretationRGB,
	"rgb16":			InterpretationRGB16,
	"grey16":			InterpretationGREY16,
	"scrgb":			InterpretationScRGB,
	"lab":				InterpretationLAB,
	"xyz":				InterpretationXYZ,
}

var imageTypeToID = map[string]ImageType {
	"webp": WEBP,
	"jpeg": JPEG,
	"gif": GIF,
	"tiff": TIFF,
	"pdf": PDF,
	"png": PNG,
	"svg": SVG,
	"magick": MAGICK,
}

func (i *Interpolator) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*i = imageInterpolatorToID[s]
	return nil
}

func (i *Interpretation) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*i = imageInterpretationToID[s]
	return nil
}

func (t *ImageType) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*t = imageTypeToID[s]
	return nil
}

func (p *Position) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*p = positions[s]
	return nil
}

func (g *Gravity) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*g = gravityToID[s]
	return nil
}

// imageMutex is used to provide thread-safe synchronization
// for SupportedImageTypes map.
var imageMutex = &sync.RWMutex{}

// SupportedImageType represents whether a type can be loaded and/or saved by
// the current libvips compilation.
type SupportedImageType struct {
	Load bool
	Save bool
}

// SupportedImageTypes stores the optional image type supported
// by the current libvips compilation.
// Note: lazy evaluation as demand is required due
// to bootstrap runtime limitation with C/libvips world.
var SupportedImageTypes = map[ImageType]SupportedImageType{}

// discoverSupportedImageTypes is used to fill SupportedImageTypes map.
func discoverSupportedImageTypes() {
	imageMutex.Lock()
	for imageType := range ImageTypes {
		SupportedImageTypes[imageType] = SupportedImageType{
			Load: VipsIsTypeSupported(imageType),
			Save: VipsIsTypeSupportedSave(imageType),
		}
	}
	imageMutex.Unlock()
}

// isBinary checks if the given buffer is a binary file.
func isBinary(buf []byte) bool {
	if len(buf) < 24 {
		return false
	}
	for i := 0; i < 24; i++ {
		charCode, _ := utf8.DecodeRuneInString(string(buf[i]))
		if charCode == 65533 || charCode <= 8 {
			return true
		}
	}
	return false
}

// IsSVGImage returns true if the given buffer is a valid SVG image.
func IsSVGImage(buf []byte) bool {
	return !isBinary(buf) && svgRegex.Match(htmlCommentRegex.ReplaceAll(buf, []byte{}))
}

// DetermineImageType determines the image type format (jpeg, png, webp or tiff)
func DetermineImageType(buf []byte) ImageType {
	return vipsImageType(buf)
}

// DetermineImageTypeName determines the image type format by name (jpeg, png, webp or tiff)
func DetermineImageTypeName(buf []byte) string {
	return ImageTypeName(vipsImageType(buf))
}

// IsImageTypeSupportedByVips returns true if the given image type
// is supported by current libvips compilation.
func IsImageTypeSupportedByVips(t ImageType) SupportedImageType {
	imageMutex.RLock()

	// Discover supported image types and cache the result
	itShouldDiscover := len(SupportedImageTypes) == 0
	if itShouldDiscover {
		imageMutex.RUnlock()
		discoverSupportedImageTypes()
	}

	// Check if image type is actually supported
	supported, ok := SupportedImageTypes[t]
	if !itShouldDiscover {
		imageMutex.RUnlock()
	}

	if ok {
		return supported
	}
	return SupportedImageType{Load: false, Save: false}
}

// IsTypeSupported checks if a given image type is supported
func IsTypeSupported(t ImageType) bool {
	_, ok := ImageTypes[t]
	return ok && IsImageTypeSupportedByVips(t).Load
}

// IsTypeNameSupported checks if a given image type name is supported
func IsTypeNameSupported(t string) bool {
	for imageType, name := range ImageTypes {
		if name == t {
			return IsImageTypeSupportedByVips(imageType).Load
		}
	}
	return false
}

// IsTypeSupportedSave checks if a given image type is support for saving
func IsTypeSupportedSave(t ImageType) bool {
	_, ok := ImageTypes[t]
	return ok && IsImageTypeSupportedByVips(t).Save
}

// IsTypeNameSupportedSave checks if a given image type name is supported for
// saving
func IsTypeNameSupportedSave(t string) bool {
	for imageType, name := range ImageTypes {
		if name == t {
			return IsImageTypeSupportedByVips(imageType).Save
		}
	}
	return false
}

// ImageTypeName is used to get the human friendly name of an image format.
func ImageTypeName(t ImageType) string {
	imageType := ImageTypes[t]
	if imageType == "" {
		return "unknown"
	}
	return imageType
}
