package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPurchaseLinksValidation(t *testing.T) {
	links, err := ParsePurchaseLinks(`{"10":" https://shop.example/ten ","20":"http://shop.example/twenty","other":""}`)
	require.NoError(t, err)
	require.Equal(t, "https://shop.example/ten", links["10"])
	require.Contains(t, links, "other")
	for _, invalid := range []string{
		`null`, `[]`, `{"500":"https://shop.example"}`, `{"10":10}`,
		`{"10":"javascript:alert(1)"}`, `{"10":"//shop.example"}`,
		`{"10":"https://user:password@shop.example"}`, `{"10":"https://shop.example/has space"}`,
	} {
		_, err := ParsePurchaseLinks(invalid)
		require.Error(t, err, invalid)
	}
}
