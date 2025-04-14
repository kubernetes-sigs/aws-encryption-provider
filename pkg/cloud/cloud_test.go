package cloud

import (
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
	kmsObjet, err := New("us-west-2", "https://kms.us-west-2.amazonaws.com", 0, 0, 500)
	assert.NoError(t, err, "Failed to create object with error (%v)", err)
	assert.NotNil(t, kmsObjet, "Failed to create object with error (%v)", err)
}

func TestNewSessionClientWithEnv(t *testing.T) {
	tempFile, err := createTmpFile(TLSBundleCert)
	assert.NoError(t, err, "Temporary file creation with CA bundle data is failing")
	defer os.Remove(tempFile)
	os.Setenv("AWS_CA_BUNDLE", tempFile)
	defer os.Unsetenv("AWS_CA_BUNDLE")
	kmsObjet, err := New("us-west-2", "https://kms.us-west-2.amazonaws.com", 0, 0, 500)
	assert.NoError(t, err, "Failed to create object with error (%v)", err)
	assert.NotNil(t, kmsObjet, "Failed to create object with error (%v)", err)
}

func createTmpFile(b []byte) (string, error) {
	bundleFile, err := os.CreateTemp(os.TempDir(), "aws-sdk-go-session-test")
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

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name               string
		region             string
		endpoint           string
		qps                int
		burst              int
		retryTokenCapacity int
		expectErr          bool
	}{
		{
			name:      "region specified",
			region:    "us-west-2",
			expectErr: false,
		},
		{
			name:      "valid qps+burst override",
			region:    "us-east-1",
			qps:       1,
			burst:     5000,
			expectErr: false,
		},
		{
			name:      "invalid qps+burst override",
			region:    "us-east-1",
			qps:       1,
			burst:     -10,
			expectErr: true,
		},
		{
			name:               "specify retry token capacity",
			region:             "us-west-2",
			retryTokenCapacity: 5000,
			expectErr:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(test.region, test.endpoint, test.qps, test.burst, test.retryTokenCapacity)
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
