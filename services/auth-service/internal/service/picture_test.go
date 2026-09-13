package service

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

func TestValidateJPEG50x50(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateJPEG(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateJPEGRejectsWrongSize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 49, 50))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateJPEG(buf.Bytes()); err == nil {
		t.Fatal("expected wrong dimensions to fail")
	}
}
