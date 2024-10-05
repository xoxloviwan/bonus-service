package helpers

import (
	"errors"
	"fmt"
)

// borrowed from benchfmt/internal/bytesconv/atoi.go
// atoi is equivalent to ParseInt(s, 10, 0), converted to type int.
func Atoi(s []byte) (int, error) {
	const intSize = 32 << (^uint(0) >> 63)

	sLen := len(s)
	if intSize == 32 && (0 < sLen && sLen < 10) ||
		intSize == 64 && (0 < sLen && sLen < 19) {
		// Fast path for small integers that fit int type.
		s0 := s

		n := 0
		for _, ch := range s {
			ch -= '0'
			if ch > 9 {
				return 0, fmt.Errorf("atoi: invalid bytes: %q", string(s0))
			}
			n = n*10 + int(ch)
		}

		return n, nil
	}
	return 0, errors.New("atoi: not realized")
}
