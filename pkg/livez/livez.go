// Package livez implements livez handlers.
package livez

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"sigs.k8s.io/aws-encryption-provider/pkg/plugin"
)

// NewHandler returns a new livez handler.
func NewHandler(p *plugin.V1Plugin) http.Handler {
	return &handler{p: p}
}

type handler struct {
	p *plugin.V1Plugin
}

func (hd *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err := hd.p.Live()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rw, err)
		zap.L().Error("live check failed", zap.Error(err))
		return
	}
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, http.StatusText(http.StatusOK))
	zap.L().Debug("live check success")
}
