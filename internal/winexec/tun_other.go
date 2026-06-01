//go:build !windows

package winexec

import "sync"

type TunWatcher struct {
	once sync.Once
}

func NewTunWatcher(_, _ func()) *TunWatcher { return &TunWatcher{} }
func (w *TunWatcher) Start()                {}
func (w *TunWatcher) Stop()                 { w.once.Do(func() {}) }
