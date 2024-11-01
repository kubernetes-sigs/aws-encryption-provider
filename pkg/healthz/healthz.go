// Package healthz implements healthz handlers.
package healthz

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"sigs.k8s.io/aws-encryption-provider/pkg/plugin"
)

// NewHandler returns a new healthz handler.
func NewHandler(p1s []*plugin.V1Plugin, p2s []*plugin.V2Plugin) http.Handler {
	return &handler{p1s: p1s, p2s: p2s}
}

type handler struct {
	p1s []*plugin.V1Plugin
	p2s []*plugin.V2Plugin
}

func (hd *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, p := range hd.p1s {
		err := p.Health()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			zap.L().Error("health check failed", zap.Error(err))
			return
		}
	}
	for _, p := range hd.p2s {
		err := p.Health()
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err)
			zap.L().Error("health check failed", zap.Error(err))
			return
		}
	}
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, http.StatusText(http.StatusOK))
	zap.L().Debug("health check success")
}
