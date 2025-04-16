// Package livez implements livez handlers.
package livez

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"sigs.k8s.io/aws-encryption-provider/pkg/plugin"
)

// NewHandler returns a new livez handler.
func NewHandler(p1s []*plugin.V1Plugin, p2s []*plugin.V2Plugin) http.Handler {
	return &handler{p1s: p1s, p2s: p2s}
}

type handler struct {
	p1s []*plugin.V1Plugin
	p2s []*plugin.V2Plugin
}

func (hd *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, p := range hd.p1s {
		err := p.Live()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, e := fmt.Fprint(rw, err)
			if e != nil {
				zap.L().Error("error writing response", zap.Error(e))
			}
			zap.L().Error("live check failed", zap.Error(err))
			return
		}
	}
	for _, p := range hd.p2s {
		err := p.Live()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, e := fmt.Fprint(rw, err)
			if e != nil {
				zap.L().Error("error writing response", zap.Error(e))
			}
			zap.L().Error("live check failed", zap.Error(err))
			return
		}
	}

	rw.WriteHeader(http.StatusOK)
	_, e := fmt.Fprint(rw, http.StatusText(http.StatusOK))
	if e != nil {
		zap.L().Error("error writing response", zap.Error(e))
	}
	zap.L().Debug("live check success")
}
