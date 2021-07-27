package cloud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TLSBundleCert ai.crt
var TLSBundleCert = []byte(`-----BEGIN CERTIFICATE-----
MIICGjCCAYOgAwIBAgIJAIIu+NOoxxM0MA0GCSqGSIb3DQEBBQUAMDgxCzAJBgNV
BAYTAkdPMQ8wDQYDVQQIEwZHb3BoZXIxGDAWBgNVBAoTD1Rlc3RpbmcgUk9PVCBD
QTAeFw0xNzAzMDkwMDAzMTRaFw0yNzAzMDcwMDAzMTRaMFExCzAJBgNVBAYTAkdP
MQ8wDQYDVQQIDAZHb3BoZXIxHDAaBgNVBAoME1Rlc3RpbmcgQ2VydGlmaWNhdGUx
EzARBgNVBAsMClRlc3RpbmcgSVAwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGB
AN1hWHeioo/nASvbrjwCQzXCiWiEzGkw353NxsAB54/NqDL3LXNATtiSJu8kJBrm
Ah12IFLtWLGXjGjjYlHbQWnOR6awveeXnQZukJyRWh7m/Qlt9Ho0CgZE1U+832ac
5GWVldNxW1Lz4I+W9/ehzqe8I80RS6eLEKfUFXGiW+9RAgMBAAGjEzARMA8GA1Ud
EQQIMAaHBH8AAAEwDQYJKoZIhvcNAQEFBQADgYEAdF4WQHfVdPCbgv9sxgJjcR1H
Hgw9rZ47gO1IiIhzglnLXQ6QuemRiHeYFg4kjcYBk1DJguxzDTGnUwhUXOibAB+S
zssmrkdYYvn9aUhjc3XK3tjAoDpsPpeBeTBamuUKDHoH/dNRXxerZ8vu6uPR3Pgs
5v/KCV6IAEcvNyOXMPo=
-----END CERTIFICATE-----
`)

func TestNewSessionClientWithoutEnv(t *testing.T) {
	kmsObjet, err := New("us-west-2", "https://kms.us-west-2.amazonaws.com", 15, 5)
	assert.NoError(t, err, "Failed to create object with error (%v)", err)
	assert.NotNil(t, kmsObjet, "Failed to create object with error (%v)", err)
}

func TestNewSessionClientWithEnv(t *testing.T) {
	tempFile, err := createTmpFile(TLSBundleCert)
	assert.NoError(t, err, "Temporary file creation with CA bundle data is failing")
	defer os.Remove(tempFile)
	os.Setenv("AWS_CA_BUNDLE", tempFile)
	defer os.Unsetenv("AWS_CA_BUNDLE")
	kmsObjet, err := New("us-west-2", "https://kms.us-west-2.amazonaws.com", 15, 5)
	assert.NoError(t, err, "Failed to create object with error (%v)", err)
	assert.NotNil(t, kmsObjet, "Failed to create object with error (%v)", err)
}

func createTmpFile(b []byte) (string, error) {
	bundleFile, err := ioutil.TempFile(os.TempDir(), "aws-sdk-go-session-test")
	if err != nil {
		return "", err
	}

	_, err = bundleFile.Write(b)
	if err != nil {
		return "", err
	}

	defer bundleFile.Close()
	return bundleFile.Name(), nil
}
