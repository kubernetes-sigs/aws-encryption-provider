package server

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestListenAndServe(t *testing.T) {
	zap.ReplaceGlobals(zap.NewExample())
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir) //nolint:errcheck
	tt := []struct {
		addr   string
		retErr error
	}{
		{
			addr:   dir + "/fileNonExists.sock",
			retErr: nil,
		},
		{
			addr:   dir + "/fileExists.sock",
			retErr: nil,
		},
		{
			addr:   ":8080",
			retErr: nil,
		},
	}
	for _, entry := range tt {
		t.Run(entry.addr, func(t *testing.T) {
			ch := make(chan error, 1)
			s := New()
			err := os.Remove(entry.addr) //nolint:errcheck
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Errorf("error removing the file %v", err)
			}
			if entry.addr == (dir + "/fileExists.sock") {
				_, err := os.Create(entry.addr)
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(3 * time.Second)
			}
			go func(addr string) {
				ch <- s.ListenAndServe(addr)
			}(entry.addr)
			select {
			case err := <-ch:
				if err != entry.retErr {
					t.Errorf("ListenAndServe() error = %v, wantErr %v", err, entry.retErr)
				}
				return
			case <-time.After(time.Second * 3):
				break
			}
		})
	}
}
