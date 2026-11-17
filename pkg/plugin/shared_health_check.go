package plugin

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"sigs.k8s.io/aws-encryption-provider/pkg/kmsplugin"
)

// TODO: make configurable
const (
	DefaultHealthCheckPeriod = 30 * time.Second
	DefaultErrcBufSize       = 100

	// DefaultKeyRevokedAfter is how long KMS must keep reporting the key as revoked,
	// with no successful call in between, before the key is treated as revoked.
	DefaultKeyRevokedAfter = 60 * time.Second
)

type SharedHealthCheck struct {
	lastMu  sync.RWMutex
	lastErr error
	lastTs  time.Time

	// revokedSince is when KMS first reported the key as revoked since the last successful call
	revokedSince    time.Time
	keyRevokedAfter time.Duration

	healthCheckPeriod         time.Duration
	healthCheckErrc           chan error
	healthCheckStopcCloseOnce *sync.Once
	healthCheckStopc          chan struct{}
	healthCheckClosed         chan struct{}
}

func NewSharedHealthCheck(
	checkPeriod time.Duration,
	errcBuf int,
) *SharedHealthCheck {
	p := &SharedHealthCheck{
		keyRevokedAfter:           DefaultKeyRevokedAfter,
		healthCheckPeriod:         checkPeriod,
		healthCheckErrc:           make(chan error, errcBuf),
		healthCheckStopcCloseOnce: new(sync.Once),
		healthCheckStopc:          make(chan struct{}),
		healthCheckClosed:         make(chan struct{}),
	}
	return p
}

func (p *SharedHealthCheck) Start() {
	zap.L().Info("starting health check routine", zap.String("period", p.healthCheckPeriod.String()))
	for {
		select {
		case <-p.healthCheckStopc:
			zap.L().Warn("exiting health check routine")
			p.healthCheckClosed <- struct{}{}
			return
		case err := <-p.healthCheckErrc:
			p.RecordErr(err)
		}
	}
}

func (p *SharedHealthCheck) Stop() {
	p.healthCheckStopcCloseOnce.Do(func() {
		close(p.healthCheckStopc)
		<-p.healthCheckClosed
	})
}

func (p *SharedHealthCheck) isRecentlyChecked() (bool, error) {
	p.lastMu.RLock()
	err, ts := p.lastErr, p.lastTs
	never, latest := err == nil && ts.IsZero(), time.Since(ts) < p.healthCheckPeriod
	p.lastMu.RUnlock()
	return !never && latest, err
}

func (p *SharedHealthCheck) RecordErr(err error) {
	p.lastMu.Lock()
	p.lastErr, p.lastTs = err, time.Now()
	switch {
	case err == nil:
		p.revokedSince = time.Time{}
	case kmsplugin.IsKeyRevoked(err) && p.revokedSince.IsZero():
		p.revokedSince = p.lastTs
	}
	p.lastMu.Unlock()
}

// isKeyRevoked returns true if the latest recorded error is a key revocation and
// KMS has kept reporting the key as revoked for at least keyRevokedAfter.
func (p *SharedHealthCheck) isKeyRevoked() bool {
	p.lastMu.RLock()
	defer p.lastMu.RUnlock()
	return !p.revokedSince.IsZero() &&
		kmsplugin.IsKeyRevoked(p.lastErr) &&
		time.Since(p.revokedSince) >= p.keyRevokedAfter
}
