package sign

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	gpgPlainKeyID     = "0279C27FC1602A0E"
	gpgEncryptedKeyID = "2AB4ABE1A4A47546"
	gpgPassword       = "secret"
	gpgHome           = "./testdata/gnupg"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}
