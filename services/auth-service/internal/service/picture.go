package service

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
)

const (
	MaxPictureBytes = 1 << 20
	PictureWidth    = 50
	PictureHeight   = 50
)

func ValidateJPEG(data []byte) error {
	if len(data) == 0 {
		return errors.New("picture is empty")
	}
	if len(data) > MaxPictureBytes {
		return errors.New("picture is too large")
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "jpeg" {
		return errors.New("picture must be a valid JPEG")
	}
	if cfg.Width != PictureWidth || cfg.Height != PictureHeight {
		return errors.New("picture must be exactly 50x50 pixels")
	}

	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return errors.New("picture must be a valid JPEG")
	}
	if img.Bounds().Dx() != PictureWidth || img.Bounds().Dy() != PictureHeight {
		return errors.New("picture must be exactly 50x50 pixels")
	}
	return nil
}

func ReadPicture(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxPictureBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxPictureBytes {
		return nil, errors.New("picture is too large")
	}
	if err := ValidateJPEG(data); err != nil {
		return nil, err
	}
	return data, nil
}
