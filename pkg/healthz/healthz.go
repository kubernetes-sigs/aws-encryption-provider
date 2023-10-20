// Package healthz implements healthz handlers.
package healthz

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"sigs.k8s.io/aws-encryption-provider/pkg/plugin"
)

// NewHandler returns a new healthz handler.
func NewHandler(p *plugin.V1Plugin) http.Handler {
	return &handler{p: p}
}

type handler struct {
	p *plugin.V1Plugin
}

func (hd *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err := hd.p.Health()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, err)
		zap.L().Error("health check failed", zap.Error(err))
		return
	}
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, http.StatusText(http.StatusOK))
	zap.L().Debug("health check success")
}
