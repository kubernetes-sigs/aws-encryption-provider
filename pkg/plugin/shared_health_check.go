package plugin

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// TODO: make configurable
const (
	DefaultHealthCheckPeriod = 30 * time.Second
	DefaultErrcBufSize       = 100
)

type SharedHealthCheck struct {
	lastMu  sync.RWMutex
	lastErr error
	lastTs  time.Time

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
			p.recordErr(err)
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

func (p *SharedHealthCheck) recordErr(err error) {
	p.lastMu.Lock()
	p.lastErr, p.lastTs = err, time.Now()
	p.lastMu.Unlock()
}
