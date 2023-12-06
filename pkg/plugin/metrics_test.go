package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	pb "k8s.io/kms/apis/v1beta1"
	"sigs.k8s.io/aws-encryption-provider/pkg/cloud"
	"sigs.k8s.io/aws-encryption-provider/pkg/server"
)

// TestMetrics tests /metrics handler.
func TestMetrics(t *testing.T) {
	zap.ReplaceGlobals(zap.NewExample())

	tt := []struct {
		key        string
		encryptErr error
		expects    string
	}{
		{
			key:        "test-key",
			encryptErr: errors.New("fail"),
			expects:    `aws_encryption_provider_kms_operations_total{key_arn="test-key",operation="encrypt",status="failure",version="v1"} 1`,
		},
		{
			key:        "test-key-throttle",
			encryptErr: awserr.New("RequestLimitExceeded", "test", errors.New("fail")),
			expects:    `aws_encryption_provider_kms_operations_total{key_arn="test-key-throttle",operation="encrypt",status="failure-throttle",version="v1"} 1`,
		},
	}
	for i, entry := range tt {
		t.Run(entry.key, func(t *testing.T) {
			addr := filepath.Join(os.TempDir(), fmt.Sprintf("metrics%x", rand.Int63()))
			defer os.RemoveAll(addr)

			c := &cloud.KMSMock{}
			c.SetEncryptResp("test", entry.encryptErr)
			sharedHealthCheck := NewSharedHealthCheck(DefaultHealthCheckPeriod, DefaultErrcBufSize)
			go sharedHealthCheck.Start()
			defer sharedHealthCheck.Stop()
			p := New(entry.key, c, nil, sharedHealthCheck)

			ready, errc := make(chan struct{}), make(chan error)
			s := server.New()
			p.Register(s.Server)
			defer func() {
				s.Server.Stop()
				if err := <-errc; err != nil {
					t.Fatalf("unexpected gRPC server stop error %v", err)
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

			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.Handler())

			ts := httptest.NewServer(mux)
			defer ts.Close()

			u := ts.URL + "/metrics"

			_, err := p.Encrypt(context.Background(), &pb.EncryptRequest{Plain: []byte("hello")})
			if err != nil {
				if entry.encryptErr == nil {
					t.Fatal(err)
				}
				if !strings.Contains(err.Error(), entry.encryptErr.Error()) {
					t.Fatalf("#%d: unexpected error %v", i, err)
				}
			}

			resp, err := http.Get(u)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			d, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(d), entry.expects) {
				t.Fatalf("#%d: expected %q, got\n\n%s\n\n", i, entry.expects, string(d))
			}
		})
	}
}
