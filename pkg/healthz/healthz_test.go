package healthz

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

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
	}{
		{
			path:          "/test-healthz-default",
			kmsEncryptErr: nil,
		},
		{
			path:          "/test-healthz-fail",
			kmsEncryptErr: errors.New("fail encrypt"),
		},
	}
	for i, entry := range tt {
		func() {
			// create temporary unix socket file
			f, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("%x", rand.Int63()))
			if err != nil {
				t.Fatal(err)
			}
			addr := f.Name()
			f.Close()
			os.RemoveAll(addr)
			defer os.RemoveAll(addr)
			c := &cloud.KMSMock{}
			c.SetEncryptResp("test", entry.kmsEncryptErr)

			p := plugin.New("test-key", c, nil)

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
			d, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if entry.kmsEncryptErr == nil && string(d) != "OK" {
				t.Fatalf("#%d: unexpected response %q, expected 'OK'", i, string(d))
			}
		}()
	}
}
