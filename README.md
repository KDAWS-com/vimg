# vimg ![License](https://img.shields.io/badge/license-MIT-blue.svg)

Small [Go](http://golang.org) package for fast high-level image processing using [libvips](https://github.com/jcupitt/libvips) via C bindings, providing a simple method based API.

vimg is a fork of [bimg](https://github.com/h2non/bimg) the fluent interface is partially done away with, as VipsImage.Process() doesn't return the buffer as it's an expensive operation.  It will be added back in via explicit calls to image.GetBuffer() at some point - but it does make it much more expensive to use the fluent interface once that is done.  

vimg is efficient.  In tests a fully instrumented (with Prometheus and pprof) HTTP server used just 170MB of RAM when processing a 1950x1300 pixel JPEG (409KB) to a 130x86 pixel WebP (3KB) at a concurrency level of 10, over 50,000 requests.
 
vimg supports the following [image operations](#supported-image-operations)

vimg is able to output images as JPEG, PNG, WEBP and TIFF formats, including transparent conversion across them.

vimg uses libvips, a powerful library written in C for image processing which requires a [low memory footprint](https://github.com/jcupitt/libvips/wiki/Speed_and_Memory_Use)
and it's typically 4x faster than using the quickest ImageMagick and GraphicsMagick settings or Go native `image` package, and in some cases it's even 8x faster processing JPEG images.

## Contents

- [Supported image operations](#supported-image-operations)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Performance](#performance)
- [Benchmark](#benchmark)
- [Examples](#examples)
- [Debugging](#debugging)
- [API](#api)
- [Authors](#authors)
- [Credits](#credits)

## Supported image operations

- Resize
- Enlarge
- Crop (including smart crop support, libvips 8.5+)
- Rotate (with auto-rotate based on EXIF orientation)
- Flip (with auto-flip based on EXIF metadata)
- Flop
- Zoom
- Thumbnail
- Extract area
- Watermark (using text or image)
- Gaussian blur effect
- Custom output color space (RGB, grayscale...)
- ICC Color Profile conversion
- ICC Color Profile extraction
- Format conversion (with additional quality/compression settings)
- EXIF metadata (size, alpha channel, profile, orientation...)
- Trim (libvips 8.6+)

## Prerequisites

- [libvips](https://github.com/jcupitt/libvips) 7.42+ or 8+ (8.4+ recommended)
- C compatible compiler such as gcc 4.6+ or clang 3.0+
- Go 1.3+

**Note**: `libvips` v8.3+ is required for GIF, PDF and SVG support.

## Installation

```bash
go get -u github.com/kdaws-com/vimg
```

### libvips

Run the following script as `sudo` (supports OSX, Debian/Ubuntu, Redhat, Fedora, Amazon Linux):
```bash
curl -s https://raw.githubusercontent.com/h2non/vimg/master/preinstall.sh | sudo bash -
```

If you wanna take the advantage of [OpenSlide](http://openslide.org/), simply add `--with-openslide` to enable it:
```bash
curl -s https://raw.githubusercontent.com/h2non/vimg/master/preinstall.sh | sudo bash -s --with-openslide
```

The [install script](https://github.com/kdaws-com/vimg/blob/master/preinstall.sh) requires `curl` and `pkg-config`.

## Performance

libvips is probably the fastest open source solution for image processing.
Here you can see some performance test comparisons for multiple scenarios:

- [libvips speed and memory usage](https://github.com/jcupitt/libvips/wiki/Speed-and-memory-use)

## Benchmark

## Examples

```go
import (
  "fmt"
  "os"
  "github.com/kdaws-com/vimg"
)
```

#### Resize

```go
buffer, err := vimg.Read("image.jpg")
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

newImage, err := vimg.NewImage(buffer, Options{})
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

err := newImage.Resize(800, 600)
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

size, err := newImage.Size()
if size.Width == 800 && size.Height == 600 {
  fmt.Println("The image size is valid")
}

imageBuffer = newImage.Save()
vimg.Write("new.jpg", imageBuffer)
```

#### Rotate

```go
buffer, err := vimg.Read("image.jpg")
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

newImage, err := vimg.NewImage(buffer, Options{})
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}
err := newImage.Rotate(90)
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

imageBuffer = newImage.Save()
vimg.Write("new.jpg", imageBuffer)
```

#### Convert

```go
buffer, err := vimg.Read("image.jpg")
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

newImage, err := vimg.NewImage(buffer, Options{})
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}
err := newImage.Convert(vimg.PNG)
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

if newImage.Type() == "png" {
  fmt.Fprintln(os.Stderr, "The image was converted into png")
}
```

#### Force resize

Force resize operation without perserving the aspect ratio:

```go
buffer, err := vimg.Read("image.jpg")
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

newImage, err := vimg.NewImage(buffer, Options{})
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}
err := newImage.ForceResize(1000, 500)
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

size, err := newImage.Size()
if size.Width != 1000 || size.Height != 500 {
  fmt.Println("Incorrect image size")
}

```

#### Custom options

See [Options](https://github.com/kdaws-com/vimg#Options) struct to discover all the available fields

```go
options := vimg.Options{
  Width:        800,
  Height:       600,
  Crop:         true,
  Quality:      95,
  Rotate:       180,
  Interlace:    true,
}

buffer, err := vimg.Read("image.jpg")
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

newImage, err := vimg.NewImage(buffer, options)
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

err := newImage.Process()
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

imageBuffer = newImage.Save()
vimg.Write("new.jpg", imageBuffer)
```

#### Watermark

```go
buffer, err := vimg.Read("image.jpg")
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

watermark := vimg.Watermark{
  Text:       "Chuck Norris (c) 2315",
  Opacity:    0.25,
  Width:      200,
  DPI:        100,
  Margin:     150,
  Font:       "sans bold 12",
  Background: vimg.Color{255, 255, 255},
}

newImage, err := vimg.NewImage(buffer, Options())
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

err := newImage.Watermark(watermark)
if err != nil {
  fmt.Fprintln(os.Stderr, err)
}

imageBuffer = newImage.Save()
vimg.Write("new.jpg", imageBuffer)
```

## Debugging

Run the process passing the `DEBUG` environment variable
```
DEBUG=vimg ./app
```

Enable libvips traces (note that a lot of data will be written in stdout):
```
VIPS_TRACE=1 ./app
```

You can also dump a core on failure, as [John Cuppit](https://github.com/jcupitt) said:
```c
g_log_set_always_fatal(
                G_LOG_FLAG_RECURSION |
                G_LOG_FLAG_FATAL |
                G_LOG_LEVEL_ERROR |
                G_LOG_LEVEL_CRITICAL |
                G_LOG_LEVEL_WARNING );
```

Or set the G_DEBUG environment variable:
```
export G_DEBUG=fatal-warnings,fatal-criticals
```

## Authors

- [Karl Austin](https://github.com/karlaustin) - Author of vimg and code changes since March 2019
- [Tomás Aparicio](https://github.com/h2non) - Original author and architect of [bimg](https://github.com/h2non/bimg).
- [Kirill Danshin](https://github.com/kirillDanshin) - Maintainer of bimg since April 2017.

## License

MIT - Karl Asutin, Tomas Aparicio