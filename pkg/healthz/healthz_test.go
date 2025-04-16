package healthz

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"go.uber.org/zap"
	"sigs.k8s.io/aws-encryption-provider/pkg/cloud"
	"sigs.k8s.io/aws-encryption-provider/pkg/plugin"
	"sigs.k8s.io/aws-encryption-provider/pkg/server"
)

// TestHealthz tests healthz handlers.
func TestHealthz(t *testing.T) {
	zap.ReplaceGlobals(zap.NewExample())

	tt := []struct {
		path          string
		kmsEncryptErr error
		shouldSucceed bool
	}{
		{
			path:          "/test-healthz-default",
			kmsEncryptErr: nil,
			shouldSucceed: true,
		},
		{
			path:          "/test-healthz-fail",
			kmsEncryptErr: errors.New("fail encrypt"),
			shouldSucceed: false,
		},

		{
			path:          "/test-healthz-fail-with-internal-error",
			kmsEncryptErr: &kmstypes.KMSInternalException{Message: aws.String("test")},
			shouldSucceed: false,
		},
		// user-induced errors should still fail "/healthz"
		{
			path:          "/test-healthz-fail-with-user-induced-invalid-key-state",
			kmsEncryptErr: &kmstypes.KMSInvalidStateException{Message: aws.String("test")},
			shouldSucceed: false,
		},
		{
			path:          "/test-healthz-fail-with-user-induced-invalid-grant",
			kmsEncryptErr: &kmstypes.InvalidGrantTokenException{Message: aws.String("test")},
			shouldSucceed: false,
		},
	}
	for i, entry := range tt {
		t.Run(entry.path, func(t *testing.T) {
			addr := filepath.Join(os.TempDir(), fmt.Sprintf("healthz%x", rand.Int63()))
			defer os.RemoveAll(addr) //nolint:errcheck

			c := &cloud.KMSMock{}
			c.SetEncryptResp("test", entry.kmsEncryptErr)
			sharedHealthCheck := plugin.NewSharedHealthCheck(plugin.DefaultHealthCheckPeriod, plugin.DefaultErrcBufSize)
			go sharedHealthCheck.Start()
			defer sharedHealthCheck.Stop()
			p := plugin.New("test-key", c, nil, sharedHealthCheck)

			ready, errc := make(chan struct{}), make(chan error)
			s := server.New()
			p.Register(s.Server)
			defer func() {
				s.Stop()
				if err := <-errc; err != nil {
					t.Fatalf("#%d: unexpected gRPC server stop error %v", i, err)
				}
			}()
			go func() {
				close(ready)
				errc <- s.ListenAndServe(addr)
			}()

			// wait enough for unix socket to be open
			time.Sleep(time.Second)
			select {
			case <-ready:
			case <-time.After(2 * time.Second):
				t.Fatal("took too long to start gRPC server")
			}

			hd := NewHandler([]*plugin.V1Plugin{p}, []*plugin.V2Plugin{})

			mux := http.NewServeMux()
			mux.Handle(entry.path, hd)

			ts := httptest.NewServer(mux)
			defer ts.Close()

			u := ts.URL + entry.path

			resp, err := http.Get(u)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close() //nolint:errcheck
			d, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if entry.shouldSucceed && string(d) != "OK" {
				t.Fatalf("#%d: expected 200 OK, got %q", i, string(d))
			}
			if !entry.shouldSucceed && string(d) == "Internal Server Error" {
				t.Fatalf("#%d: expected 500 Internal Server Error, got %q", i, string(d))
			}
		})
	}
}
