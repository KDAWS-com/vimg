package vimg

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
)

// vipsVersionAtLeast compares semantic versions correctly.
// String comparison fails: "8.15.1" < "8.5.0" lexicographically because '1' < '5'
func vipsVersionAtLeast(minVersion string) bool {
	parseParts := func(v string) (int, int, int) {
		parts := strings.Split(v, ".")
		major, minor, patch := 0, 0, 0
		if len(parts) >= 1 {
			major, _ = strconv.Atoi(parts[0])
		}
		if len(parts) >= 2 {
			minor, _ = strconv.Atoi(parts[1])
		}
		if len(parts) >= 3 {
			patch, _ = strconv.Atoi(parts[2])
		}
		return major, minor, patch
	}

	curMaj, curMin, curPatch := parseParts(VipsVersion)
	minMaj, minMin, minPatch := parseParts(minVersion)

	if curMaj != minMaj {
		return curMaj > minMaj
	}
	if curMin != minMin {
		return curMin > minMin
	}
	return curPatch >= minPatch
}

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
	if vipsVersionAtLeast("8.5.0") {
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
		{"tiff", vipsVersionAtLeast("8.5.0")},
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

func TestHEIFAVIFBrandDetection(t *testing.T) {
	// Build minimal ftyp box: [size:4][ftyp:4][brand:4]
	makeFtypBuf := func(brand string) []byte {
		buf := make([]byte, 12)
		copy(buf[4:8], "ftyp")
		copy(buf[8:12], brand)
		return buf
	}

	tests := []struct {
		name     string
		brand    string
		expected ImageType
	}{
		// HEIF brands
		{"heic brand", "heic", HEIF},
		{"heix brand", "heix", HEIF},
		{"hevc brand", "hevc", HEIF},
		{"hevx brand", "hevx", HEIF},
		{"mif1 brand", "mif1", HEIF},
		{"msf1 brand", "msf1", HEIF},
		{"MiHE brand", "MiHE", HEIF},
		{"MiHB brand", "MiHB", HEIF},
		// AVIF brands
		{"avif brand", "avif", AVIF},
		{"avis brand", "avis", AVIF},
		{"MA1B brand", "MA1B", AVIF},
		{"MA1A brand", "MA1A", AVIF},
		// Unknown brands should return UNKNOWN
		{"mp41 brand", "mp41", UNKNOWN},
		{"isom brand", "isom", UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := makeFtypBuf(tt.brand)
			result := vipsImageType(buf)

			// If format is supported, verify correct detection
			if tt.expected != UNKNOWN && VipsIsTypeSupported(tt.expected) {
				if result != tt.expected {
					t.Errorf("Expected %v for brand %s, got %v", tt.expected, tt.brand, result)
				}
			} else if tt.expected == UNKNOWN {
				// Unknown brands should return UNKNOWN regardless of support
				if result != UNKNOWN {
					t.Errorf("Expected UNKNOWN for brand %s, got %v", tt.brand, result)
				}
			}
		})
	}
}
