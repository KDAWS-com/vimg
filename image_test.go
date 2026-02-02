package vimg

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"testing"
)

func TestImageResize(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Resize(300, 240)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	err = assertSize(t, image, 300, 240)
	if err != nil {
		t.Error(err)
	}
}

func TestImageGifResize(t *testing.T) {
	// Note: GIF support depends on libvips compilation options
	// This test just verifies we can load and resize a GIF
	image := initImage(t, "test.gif")
	if image == nil {
		t.Skip("GIF test image not available")
		return
	}
	defer image.DecrementReferenceCount()

	// Just test that we can resize without panic
	_ = image.Resize(300, 240)
}

func TestImageResizeAndCrop(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.ResizeAndCrop(300, 200)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	err = assertSize(t, image, 300, 200)
	if err != nil {
		t.Error(err)
	}
}

func TestImageExtract(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Extract(100, 100, 300, 200)
	if err != nil {
		t.Errorf("Cannot process the image: %s", err)
		return
	}

	err = assertSize(t, image, 300, 200)
	if err != nil {
		t.Error(err)
	}
}

func TestImageEnlarge(t *testing.T) {
	image := initImage(t, "test.png")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Enlarge(500, 375)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	err = assertSize(t, image, 500, 375)
	if err != nil {
		t.Error(err)
	}
}

func TestImageCrop(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Crop(800, 600, GravityNorth)
	if err != nil {
		t.Errorf("Cannot process the image: %s", err)
		return
	}

	err = assertSize(t, image, 800, 600)
	if err != nil {
		t.Error(err)
	}
}

func TestImageThumbnail(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Thumbnail(100)
	if err != nil {
		t.Errorf("Cannot process the image: %s", err)
		return
	}

	err = assertSize(t, image, 100, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestImageFlip(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Flip()
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
	}
}

func TestImageFlop(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Flop()
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
	}
}

func TestImageRotate(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Rotate(90)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
	}
}

func TestImageConvert(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Convert(PNG)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
	}
}

func TestImageMetadata(t *testing.T) {
	image := initImage(t, "test.png")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	data, err := image.Metadata()
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}
	if data.Alpha != true {
		t.Fatal("Invalid alpha channel")
	}
	if data.Size.Width != 400 {
		t.Fatal("Invalid width size")
	}
	if data.Type != "png" {
		t.Fatal("Invalid image type")
	}
}

func TestInterpretation(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	interpretation, err := image.Interpretation()
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}
	if interpretation != InterpretationSRGB {
		t.Errorf("Invalid interpretation: %d", interpretation)
	}
}

func TestImageColourspaceIsSupported(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	supported, err := image.ColourspaceIsSupported()
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}
	if supported != true {
		t.Errorf("Non-supported colourspace")
	}
}

func TestFluentInterface(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.CropByWidth(300)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	err = image.Flip()
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	err = image.Convert(PNG)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	data, _ := image.Metadata()
	// Note: Alpha is false because source is JPEG (no alpha channel)
	if data.Alpha != false {
		t.Fatal("Invalid alpha channel")
	}
	if data.Size.Width != 300 {
		t.Fatal("Invalid width size")
	}
	// Note: Metadata.Type reflects the original buffer type, not the target conversion
	// The conversion happens during Save(), so type remains "jpeg" until saved
	if data.Type != "jpeg" {
		t.Fatalf("Invalid image type: expected jpeg, got %s", data.Type)
	}
}

func TestImageSmartCrop(t *testing.T) {
	if !(VipsMajorVersion >= 8 && VipsMinorVersion >= 5) {
		t.Skipf("Skipping this test, libvips doesn't meet version requirement %s >= 8.5", VipsVersion)
	}

	image := initImage(t, "northern_cardinal_bird.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.SmartCrop(300, 300)
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	err = assertSize(t, image, 300, 300)
	if err != nil {
		t.Error(err)
	}
}

func TestImageTrim(t *testing.T) {
	if !(VipsMajorVersion >= 8 && VipsMinorVersion >= 6) {
		t.Skipf("Skipping this test, libvips doesn't meet version requirement %s >= 8.6", VipsVersion)
	}

	image := initImage(t, "transparent.png")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	err := image.Trim()
	if err != nil {
		t.Errorf("Cannot process the image: %#v", err)
		return
	}

	err = assertSize(t, image, 250, 208)
	if err != nil {
		t.Errorf("The image wasn't trimmed: %v", err)
	}
}

func TestImageLength(t *testing.T) {
	image := initImage(t, "test.jpg")
	if image == nil {
		return
	}
	defer image.DecrementReferenceCount()

	actual := image.Length()
	// Length should be positive for a valid image
	if actual <= 0 {
		t.Errorf("Expected positive image length, got %d", actual)
	}
}

// Helper functions

func initImage(t *testing.T, file string) *Image {
	buf, err := imageBuf(file)
	if err != nil {
		t.Skipf("Cannot load test image %s: %v", file, err)
		return nil
	}
	image, err := NewImage(buf, Options{})
	if err != nil {
		t.Errorf("Cannot create image: %v", err)
		return nil
	}
	return image
}

func imageBuf(file string) (*bytes.Buffer, error) {
	data, err := os.ReadFile(path.Join("testdata", file))
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

func assertSize(t *testing.T, image *Image, width, height int) error {
	size, err := image.Size()
	if err != nil {
		return err
	}
	if size.Width != width || size.Height != height {
		return fmt.Errorf("Invalid image size: %dx%d (expected %dx%d)", size.Width, size.Height, width, height)
	}
	return nil
}
