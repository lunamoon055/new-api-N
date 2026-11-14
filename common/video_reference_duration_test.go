package common

import (
	"context"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoReferenceDurationUsesContainerMetadata(t *testing.T) {
	element := func(id, payload []byte) []byte {
		result := append([]byte{}, id...)
		result = append(result, byte(len(payload))|0x80)
		return append(result, payload...)
	}
	for _, value := range []float64{0.25, 7.5, 10} {
		bits := make([]byte, 8)
		binary.BigEndian.PutUint64(bits, math.Float64bits(value*1000))
		info := element([]byte{0x44, 0x89}, bits)
		data := append(element([]byte{0x1a, 0x45, 0xdf, 0xa3}, nil), element([]byte{0x18, 0x53, 0x80, 0x67}, element([]byte{0x15, 0x49, 0xa9, 0x66}, info))...)
		duration, format, err := VideoReferenceDuration(context.Background(), data)
		require.NoError(t, err)
		require.Equal(t, "video/webm", format)
		require.InDelta(t, value, duration, 1e-9)
		_, _, err = VideoReferenceDuration(context.Background(), data[:len(data)-1])
		require.Error(t, err)
	}
	for _, data := range [][]byte{nil, []byte("<html>not media</html>"), {0x1a, 0x45, 0xdf, 0xa3, 0xff}, {0x1a, 0x45, 0xdf, 0xa3, 0x80}} {
		_, _, err := VideoReferenceDuration(context.Background(), data)
		require.Error(t, err)
	}
	// mvhd metadata is enough for the parser's deterministic fixture; no video
	// generation or outbound media upload is needed to validate duration reading.
	box := func(kind string, payload []byte) []byte {
		b := make([]byte, 8)
		binary.BigEndian.PutUint32(b, uint32(len(payload)+8))
		copy(b[4:], kind)
		return append(b, payload...)
	}
	mvhd := make([]byte, 100)
	binary.BigEndian.PutUint32(mvhd[12:], 1000)
	binary.BigEndian.PutUint32(mvhd[16:], 7500)
	data := append(box("ftyp", []byte("isom\x00\x00\x02\x00isomiso2")), box("moov", box("mvhd", mvhd))...)
	duration, format, err := VideoReferenceDuration(context.Background(), data)
	require.NoError(t, err)
	require.Equal(t, "video/mp4", format)
	require.InDelta(t, 7.5, duration, 1e-9)
}
