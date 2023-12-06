package livez

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

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms"
	"go.uber.org/zap"
	"sigs.k8s.io/aws-encryption-provider/pkg/cloud"
	"sigs.k8s.io/aws-encryption-provider/pkg/plugin"
	"sigs.k8s.io/aws-encryption-provider/pkg/server"
)

// TestLivez tests livez handlers.
func TestLivez(t *testing.T) {
	zap.ReplaceGlobals(zap.NewExample())

	tt := []struct {
		path          string
		kmsEncryptErr error
		shouldSucceed bool
	}{
		{
			path:          "/test-livez-default",
			kmsEncryptErr: nil,
			shouldSucceed: true,
		},
		{
			path:          "/test-livez-fail",
			kmsEncryptErr: errors.New("fail encrypt"),
			shouldSucceed: false,
		},
		{
			path:          "/test-livez-fail-with-internal-error",
			kmsEncryptErr: awserr.New(kms.ErrCodeInternalException, "test", errors.New("fail")),
			shouldSucceed: false,
		},

		// user-induced
		{
			path:          "/test-livez-fail-with-user-induced-invalid-key-state",
			kmsEncryptErr: awserr.New(kms.ErrCodeInvalidStateException, "test", errors.New("fail")),
			shouldSucceed: true,
		},
		{
			path:          "/test-livez-fail-with-user-induced-invalid-grant",
			kmsEncryptErr: awserr.New(kms.ErrCodeInvalidGrantTokenException, "test", errors.New("fail")),
			shouldSucceed: true,
		},
	}
	for i, entry := range tt {
		t.Run(entry.path, func(t *testing.T) {
			addr := filepath.Join(os.TempDir(), fmt.Sprintf("livez%x", rand.Int63()))
			defer os.RemoveAll(addr)

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
				s.Server.Stop()
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

			hd := NewHandler(p)

			mux := http.NewServeMux()
			mux.Handle(entry.path, hd)

			ts := httptest.NewServer(mux)
			defer ts.Close()

			u := ts.URL + entry.path

			resp, err := http.Get(u)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
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
