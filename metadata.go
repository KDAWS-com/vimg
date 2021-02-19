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
	EXIF		EXIF
}

const (
	Make = "exif-ifd0-Make"
	Model = "exif-ifd0-Model"
	Orientation = "exif-ifd0-Orientation"
	XResolution = "exif-ifd0-XResolution"
	YResolution = "exif-ifd0-YResolution"
	ResolutionUnit = "exif-ifd0-ResolutionUnit"
	Software = "exif-ifd0-Software"
	Datetime = "exif-ifd0-DateTime"
	YCbCrPositioning = "exif-ifd0-YCbCrPositioning"
	Compression = "exif-ifd1-Compression"
	ExposureTime = "exif-ifd2-ExposureTime"
	FNumber = "exif-ifd2-FNumber"
	ExposureProgram = "exif-ifd2-ExposureProgram"
	ISOSpeedRatings = "exif-ifd2-ISOSpeedRatings"
	ExifVersion = "exif-ifd2-ExifVersion"
	DateTimeOriginal = "exif-ifd2-DateTimeOriginal"
	DateTimeDigitized = "exif-ifd2-DateTimeDigitized"
	ComponentsConfiguration = "exif-ifd2-ComponentsConfiguration"
	ShutterSpeedValue = "exif-ifd2-ShutterSpeedValue"
	ApertureValue = "exif-ifd2-ApertureValue"
	BrightnessValue = "exif-ifd2-BrightnessValue"
	ExposureBiasValue = "exif-ifd2-ExposureBiasValue"
	MeteringMode = "exif-ifd2-MeteringMode"
	Flash = "exif-ifd2-Flash"
	FocalLength = "exif-ifd2-FocalLength"
	SubjectArea = "exif-ifd2-SubjectArea"
	MakerNote = "exif-ifd2-MakerNote"
	SubSecTimeOriginal = "exif-ifd2-SubSecTimeOriginal"
	SubSecTimeDigitized = "exif-ifd2-SubSecTimeDigitized"
	ColorSpace = "exif-ifd2-ColorSpace"
	PixelXDimension = "exif-ifd2-PixelXDimension"
	PixelYDimension = "exif-ifd2-PixelYDimension"
	SensingMethod = "exif-ifd2-SensingMethod"
	SceneType = "exif-ifd2-SceneType"
	ExposureMode = "exif-ifd2-ExposureMode"
	WhiteBalance = "exif-ifd2-WhiteBalance"
	FocalLengthIn35mmFilm = "exif-ifd2-FocalLengthIn35mmFilm"
	SceneCaptureType = "exif-ifd2-SceneCaptureType"
	GPSLatitudeRef = "exif-ifd3-GPSLatitudeRef"
	GPSLatitude = "exif-ifd3-GPSLatitude"
	GPSLongitudeRef = "exif-ifd3-GPSLongitudeRef"
	GPSLongitude = "exif-ifd3-GPSLongitude"
	GPSAltitudeRef = "exif-ifd3-GPSAltitudeRef"
	GPSAltitude = "exif-ifd3-GPSAltitude"
	GPSSpeedRef = "exif-ifd3-GPSSpeedRef"
	GPSSpeed = "exif-ifd3-GPSSpeed"
	GPSImgDirectionRef = "exif-ifd3-GPSImgDirectionRef"
	GPSImgDirection = "exif-ifd3-GPSImgDirection"
	GPSDestBearingRef = "exif-ifd3-GPSDestBearingRef"
	GPSDestBearing = "exif-ifd3-GPSDestBearing"
	GPSDateStamp = "exif-ifd3-GPSDateStamp"
)

type EXIF struct {
	Make string
	Model string
	Orientation int
	XResolution string
	YResolution string
	ResolutionUnit int
	Software string
	Datetime string
	YCbCrPositioning int
	Compression int
	ExposureTime string
	FNumber string
	ExposureProgram int
	ISOSpeedRatings int
	ExifVersion string
	DateTimeOriginal string
	DateTimeDigitized string
	ComponentsConfiguration string
	ShutterSpeedValue string
	ApertureValue string
	BrightnessValue string
	ExposureBiasValue string
	MeteringMode int
	Flash int
	FocalLength string
	SubjectArea string
	MakerNote string
	SubSecTimeOriginal string
	SubSecTimeDigitized string
	ColorSpace int
	PixelXDimension int
	PixelYDimension int
	SensingMethod int
	SceneType string
	ExposureMode int
	WhiteBalance int
	FocalLengthIn35mmFilm int
	SceneCaptureType int
	GPSLatitudeRef string
	GPSLatitude string
	GPSLongitudeRef string
	GPSLongitude string
	GPSAltitudeRef string
	GPSAltitude string
	GPSSpeedRef string
	GPSSpeed string
	GPSImgDirectionRef string
	GPSImgDirection string
	GPSDestBearingRef string
	GPSDestBearing string
	GPSDateStamp string
}

// Metadata returns the image metadata (size, type, alpha channel, profile, EXIF orientation...).
func (img *VipsImage) Metadata() (ImageMetadata, error) {

	size := ImageSize{
		Width:  int(img.Image.Xsize),
		Height: int(img.Image.Ysize),
	}

	o, err := img.vipsExifIntTag(Orientation)
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
		EXIF: EXIF{
			Make: img.vipsExifStringTag(Make),
			Model: img.vipsExifStringTag(Model),
			Orientation: o,
			XResolution: img.vipsExifStringTag(XResolution),
			YResolution: img.vipsExifStringTag(YResolution),
			ResolutionUnit: img.vipsExifIntTag(ResolutionUnit),
			Software: img.vipsExifStringTag(Software),
			Datetime: img.vipsExifStringTag(Datetime),
			YCbCrPositioning: img.vipsExifIntTag(YCbCrPositioning),
			Compression: img.vipsExifIntTag(Compression),
			ExposureTime: img.vipsExifStringTag(ExposureTime),
			FNumber: img.vipsExifStringTag(FNumber),
			ExposureProgram: img.vipsExifIntTag(ExposureProgram),
			ISOSpeedRatings: img.vipsExifIntTag(ISOSpeedRatings),
			ExifVersion: img.vipsExifStringTag(ExifVersion),
			DateTimeOriginal: img.vipsExifStringTag(DateTimeOriginal),
			DateTimeDigitized: img.vipsExifStringTag(DateTimeDigitized),
			ComponentsConfiguration: img.vipsExifStringTag(ComponentsConfiguration),
			ShutterSpeedValue: img.vipsExifStringTag(ShutterSpeedValue),
			ApertureValue: img.vipsExifStringTag(ApertureValue),
			BrightnessValue: img.vipsExifStringTag(BrightnessValue),
			ExposureBiasValue: img.vipsExifStringTag(ExposureBiasValue),
			MeteringMode: img.vipsExifIntTag(MeteringMode),
			Flash: img.vipsExifIntTag(Flash),
			FocalLength: img.vipsExifStringTag(FocalLength),
			SubjectArea: img.vipsExifStringTag(SubjectArea),
			MakerNote: img.vipsExifStringTag(MakerNote),
			SubSecTimeOriginal: img.vipsExifStringTag(SubSecTimeOriginal),
			SubSecTimeDigitized: img.vipsExifStringTag(SubSecTimeDigitized),
			ColorSpace: img.vipsExifIntTag(ColorSpace),
			PixelXDimension: img.vipsExifIntTag(PixelXDimension),
			PixelYDimension: img.vipsExifIntTag(PixelYDimension),
			SensingMethod: img.vipsExifIntTag(SensingMethod),
			SceneType: img.vipsExifStringTag(SceneType),
			ExposureMode: img.vipsExifIntTag(ExposureMode),
			WhiteBalance: img.vipsExifIntTag(WhiteBalance),
			FocalLengthIn35mmFilm: img.vipsExifIntTag(FocalLengthIn35mmFilm),
			SceneCaptureType: img.vipsExifIntTag(SceneCaptureType),
			GPSLatitudeRef: img.vipsExifStringTag(GPSLatitudeRef),
			GPSLatitude: img.vipsExifStringTag(GPSLatitude),
			GPSLongitudeRef: img.vipsExifStringTag(GPSLongitudeRef),
			GPSLongitude: img.vipsExifStringTag(GPSLongitude),
			GPSAltitudeRef: img.vipsExifStringTag(GPSAltitudeRef),
			GPSAltitude: img.vipsExifStringTag(GPSAltitude),
			GPSSpeedRef: img.vipsExifStringTag(GPSSpeedRef),
			GPSSpeed: img.vipsExifStringTag(GPSSpeed),
			GPSImgDirectionRef: img.vipsExifStringTag(GPSImgDirectionRef),
			GPSImgDirection: img.vipsExifStringTag(GPSImgDirection),
			GPSDestBearingRef: img.vipsExifStringTag(GPSDestBearingRef),
			GPSDestBearing: img.vipsExifStringTag(GPSDestBearing),
			GPSDateStamp: img.vipsExifStringTag(GPSDateStamp),
		},
	}

	return metadata, nil
}
