package etherscan

import (
	"testing"
)

func TestGetABI(t *testing.T) {
	if _, err := getABI("0xfb6916095ca1df60bb79ce92ce3ea74c37c5d359"); err != nil {
		t.Fatal(err)
	}
}
