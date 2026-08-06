package service

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
)

// decodeImage decodes an image from a reader.
func decodeImage(r io.Reader) (image.Image, string, error) {
	return image.Decode(r)
}

// encodeJPEG encodes an image as JPEG with the given quality (1-100).
func encodeJPEG(w io.Writer, img image.Image, quality int) error {
	return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
}

// reencodeAsJPEG decodes image data and re-encodes it as JPEG.
// This ensures the file content matches the .jpg extension.
func reencodeAsJPEG(data []byte) ([]byte, error) {
	img, _, err := decodeImage(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := encodeJPEG(&buf, img, 85); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// resizeImage scales an image to the specified dimensions using nearest-neighbor.
func resizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := (x * srcW) / width
			sy := (y * srcH) / height
			dst.Set(x, y, img.At(srcBounds.Min.X+sx, srcBounds.Min.Y+sy))
		}
	}
	return dst
}
