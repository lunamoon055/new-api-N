package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadResponseBodyWithLimit(t *testing.T) {
	body, err := ReadResponseBodyWithLimit(strings.NewReader("hello"), 5)

	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), body)
}

func TestReadResponseBodyWithLimitRejectsOversizedBody(t *testing.T) {
	body, err := ReadResponseBodyWithLimit(strings.NewReader("hello"), 4)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrResponseBodyTooLarge)
	assert.Nil(t, body)
}
