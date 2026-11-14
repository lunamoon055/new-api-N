package common

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
)

// Read container metadata from the exact bounded bytes forwarded upstream.
// Client duration_seconds hints never determine the charged input duration.
func VideoReferenceDuration(ctx context.Context, data []byte) (result float64, contentType string, resultErr error) {
	defer func() {
		if recover() != nil {
			result, contentType, resultErr = 0, "", fmt.Errorf("invalid reference-video container")
		}
	}()
	var seconds float64
	var err error
	format := ""
	switch {
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		format = "video/mp4"
		seconds, err = GetAudioDuration(ctx, bytes.NewReader(data), ".mp4")
	case len(data) >= 4 && binary.BigEndian.Uint32(data[:4]) == 0x1a45dfa3:
		format = "video/webm"
		seconds, err = webMReferenceDuration(data)
	default:
		err = fmt.Errorf("reference videos must be MP4 or WebM files")
	}
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > 3600 {
		return 0, "", fmt.Errorf("cannot verify video duration; use a complete MP4/WebM file with valid duration metadata")
	}
	return seconds, format, nil
}

func webMReferenceDuration(data []byte) (float64, error) {
	scale := uint64(1000000)
	duration := float64(0)
	foundSegment, foundInfo := false, false
	var walk func([]byte, int) error
	walk = func(data []byte, depth int) error {
		if depth > 3 {
			return fmt.Errorf("invalid WebM nesting")
		}
		for count := 0; len(data) > 0; count++ {
			if count > 10000 {
				return fmt.Errorf("too many WebM elements")
			}
			id, n, err := readWebMVInt(data, true)
			if err != nil {
				return err
			}
			data = data[n:]
			size, n, err := readWebMVInt(data, false)
			if err != nil {
				return err
			}
			data = data[n:]
			unknown := size == (uint64(1)<<(uint(n)*7))-1
			if unknown && id != 0x18538067 {
				return fmt.Errorf("unknown WebM element size")
			}
			if unknown {
				size = uint64(len(data))
			}
			if size > uint64(len(data)) {
				return fmt.Errorf("truncated WebM element")
			}
			value := data[:int(size)]
			data = data[int(size):]
			switch {
			case depth == 0 && id == 0x18538067:
				foundSegment = true
				if err := walk(value, 1); err != nil {
					return err
				}
			case depth == 1 && id == 0x1549a966:
				if foundInfo {
					return fmt.Errorf("duplicate WebM info")
				}
				foundInfo = true
				if err := walk(value, 2); err != nil {
					return err
				}
			case depth == 2 && id == 0x2ad7b1:
				if len(value) < 1 || len(value) > 8 {
					return fmt.Errorf("invalid WebM time scale")
				}
				scale = 0
				for _, b := range value {
					scale = scale<<8 | uint64(b)
				}
			case depth == 2 && id == 0x4489:
				switch len(value) {
				case 4:
					duration = float64(math.Float32frombits(binary.BigEndian.Uint32(value)))
				case 8:
					duration = math.Float64frombits(binary.BigEndian.Uint64(value))
				default:
					return fmt.Errorf("invalid WebM duration")
				}
			}
		}
		return nil
	}
	if err := walk(data, 0); err != nil {
		return 0, err
	}
	if !foundSegment || !foundInfo || scale == 0 || scale > 1000000000 {
		return 0, fmt.Errorf("WebM duration metadata missing")
	}
	return duration * float64(scale) / 1e9, nil
}

func readWebMVInt(data []byte, id bool) (uint64, int, error) {
	if len(data) == 0 || data[0] == 0 {
		return 0, 0, fmt.Errorf("invalid WebM element")
	}
	n, mask := 1, byte(0x80)
	for data[0]&mask == 0 {
		n++
		mask >>= 1
	}
	if n > len(data) || (id && n > 4) {
		return 0, 0, fmt.Errorf("invalid WebM element length")
	}
	v := uint64(data[0])
	if !id {
		v &= uint64(mask - 1)
	}
	for _, b := range data[1:n] {
		v = v<<8 | uint64(b)
	}
	return v, n, nil
}
