package vimg

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestDeterminateImageType(t *testing.T) {
	files := []struct {
		name     string
		expected ImageType
	}{
		{"test.jpg", JPEG},
		{"test.png", PNG},
		{"test.webp", WEBP},
		{"test.gif", GIF},
		{"test.pdf", PDF},
		{"test.svg", SVG},
		{"test.jp2", MAGICK},
	}

	for _, file := range files {
		img, _ := os.Open(path.Join("testdata", file.name))
		buf, _ := ioutil.ReadAll(img)
		defer img.Close()

		if VipsIsTypeSupported(file.expected) {
			if DetermineImageType(buf) != file.expected {
				t.Fatalf("Image type is not valid: %s != %s", file.name, ImageTypes[file.expected])
			}
		}
	}
}

func TestDeterminateImageTypeName(t *testing.T) {
	files := []struct {
		name     string
		expected string
	}{
		{"test.jpg", "jpeg"},
		{"test.png", "png"},
		{"test.webp", "webp"},
		{"test.gif", "gif"},
		{"test.pdf", "pdf"},
		{"test.svg", "svg"},
		{"test.jp2", "magick"},
	}

	for _, file := range files {
		img, _ := os.Open(path.Join("testdata", file.name))
		buf, _ := ioutil.ReadAll(img)
		defer img.Close()

		if DetermineImageTypeName(buf) != file.expected {
			t.Fatalf("Image type is not valid: %s != %s", file.name, file.expected)
		}
	}
}

func TestIsTypeSupported(t *testing.T) {
	types := []struct {
		name ImageType
	}{
		{JPEG}, {PNG}, {WEBP}, {GIF}, {PDF},
	}

	for _, n := range types {
		if IsTypeSupported(n.name) == false {
			t.Fatalf("Image type %s is not valid", ImageTypes[n.name])
		}
	}
}

func TestIsTypeNameSupported(t *testing.T) {
	types := []struct {
		name     string
		expected bool
	}{
		{"jpeg", true},
		{"png", true},
		{"webp", true},
		{"gif", true},
		{"pdf", true},
	}

	for _, n := range types {
		if IsTypeNameSupported(n.name) != n.expected {
			t.Fatalf("Image type %s is not valid", n.name)
		}
	}
}

func TestIsTypeSupportedSave(t *testing.T) {
	types := []struct {
		name ImageType
	}{
		{JPEG}, {PNG}, {WEBP},
	}
	if VipsVersion >= "8.5.0" {
		types = append(types, struct{ name ImageType }{TIFF})
	}

	for _, n := range types {
		if IsTypeSupportedSave(n.name) == false {
			t.Fatalf("Image type %s is not valid", ImageTypes[n.name])
		}
	}
}

func TestIsTypeNameSupportedSave(t *testing.T) {
	types := []struct {
		name     string
		expected bool
	}{
		{"jpeg", true},
		{"png", true},
		{"webp", true},
		{"gif", false},
		{"pdf", false},
		{"tiff", VipsVersion >= "8.5.0"},
	}

	for _, n := range types {
		if IsTypeNameSupportedSave(n.name) != n.expected {
			t.Fatalf("Image type %s is not valid", n.name)
		}
	}
}

func TestHEIFSupport(t *testing.T) {
	if !VipsIsTypeSupported(HEIF) {
		t.Skipf("Skipping, libvips %s does not support HEIF", VipsVersion)
	}

	if !IsTypeSupported(HEIF) {
		t.Error("HEIF should be supported when VipsIsTypeSupported returns true")
	}

	if ImageTypeName(HEIF) != "heif" {
		t.Error("HEIF type name should be 'heif'")
	}

	// Test heic alias
	if imageTypeToID["heic"] != HEIF {
		t.Error("'heic' should be an alias for HEIF")
	}
}

func TestAVIFSupport(t *testing.T) {
	if !VipsIsTypeSupported(AVIF) {
		t.Skipf("Skipping, libvips %s does not support AVIF", VipsVersion)
	}

	if !IsTypeSupported(AVIF) {
		t.Error("AVIF should be supported when VipsIsTypeSupported returns true")
	}

	if ImageTypeName(AVIF) != "avif" {
		t.Error("AVIF type name should be 'avif'")
	}
}

func TestHEIFSaveSupport(t *testing.T) {
	if !VipsIsTypeSupportedSave(HEIF) {
		t.Skipf("Skipping, libvips %s does not support HEIF save", VipsVersion)
	}

	if !IsTypeSupportedSave(HEIF) {
		t.Error("HEIF save should be supported when VipsIsTypeSupportedSave returns true")
	}
}

func TestAVIFSaveSupport(t *testing.T) {
	if !VipsIsTypeSupportedSave(AVIF) {
		t.Skipf("Skipping, libvips %s does not support AVIF save", VipsVersion)
	}

	if !IsTypeSupportedSave(AVIF) {
		t.Error("AVIF save should be supported when VipsIsTypeSupportedSave returns true")
	}
}
