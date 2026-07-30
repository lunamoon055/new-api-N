package common

import (
	"errors"
	"fmt"
	"io"
	"math"
)

var ErrResponseBodyTooLarge = errors.New("response body too large")

const MaxUpstreamResponseBodyBytes int64 = 16 << 20

func ReadResponseBodyWithLimit(reader io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes < 0 || maxBytes == math.MaxInt64 {
		return nil, fmt.Errorf("invalid response body limit: %d", maxBytes)
	}

	body, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%w: maximum is %d bytes", ErrResponseBodyTooLarge, maxBytes)
	}
	return body, nil
}
