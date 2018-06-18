package httpServer

import (
	"error"
	"github.com/Sirupsen/logrus"
	"github.com/parnurzeal/gorequest"
	"sync"
	"time"
)

type HealthProvider struct {
	name   string
	appUrl string
	health error
	done   chan struct{}
	lock   sync.RWMutex
}

func NewHealthProvider(name string, appUrl string) *HealthProvider {
	if name == nil || name == "" {
		name = "httpServer"
	}
	return &HealthProvider{name: name, appUrl: appUrl, health: "health status unchecked"}
}

func (p *HealthProvider) Name() string {
	return p.name
}

func (p *HealthProvider) Start() error {
	logrus.Debug("Starting httpServer health provider")
	p.done = make(chan struct{}, 1)

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		p.performCheck()
		for {
			select {
			case <-ticker.C:
				p.performCheck()
			case <-p.done:
				ticker.Stop()
				logrus.WithField("healthProvider", p.Name()).Warn("Received close signal, shutting down")
				return
			}
		}
	}()

	return nil
}

func (p *HealthProvider) performCheck() {
	logrus.Debug("Checking httpServer health")
	p.lock.Lock()
	defer p.lock.Unlock()
	resp, _, errs := gorequest.New().Get(p.appUrl).End()

	if errs != nil {
		// default return first error
		p.healthy = errs[0]
		return
	}

	if resp.StatusCode != 200 {
		p.health = error.Error("Checked httpServer with code=" + resp.StatusCode)
		return
	}
	p.healthy = nil
}

func (p *HealthProvider) IsHealthy() error {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.healthy
}

func (p *HealthProvider) Close() error {
	p.done <- struct{}{}
	return nil
}
