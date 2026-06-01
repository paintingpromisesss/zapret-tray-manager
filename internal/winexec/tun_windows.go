//go:build windows

package winexec

import (
	"net"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	iphlpapi                    = syscall.NewLazyDLL("iphlpapi.dll")
	procNotifyIPInterfaceChange = iphlpapi.NewProc("NotifyIpInterfaceChange")
	procCancelMibChangeNotify2  = iphlpapi.NewProc("CancelMibChangeNotify2")
)

const afUnspec = 0 // AF_UNSPEC — notify for both IPv4 and IPv6

type TunWatcher struct {
	onConnect    func()
	onDisconnect func()
	callback     uintptr

	mu      sync.Mutex
	handle  uintptr
	running bool
	prev    bool
}

func NewTunWatcher(onConnect, onDisconnect func()) *TunWatcher {
	w := &TunWatcher{
		onConnect:    onConnect,
		onDisconnect: onDisconnect,
	}
	w.callback = syscall.NewCallback(func(_ uintptr, _ uintptr, _ uint32) uintptr {
		w.mu.Lock()
		if !w.running {
			w.mu.Unlock()
			return 0
		}
		cur := hasTunAdapter()
		prev := w.prev
		w.prev = cur
		onConnect := w.onConnect
		onDisconnect := w.onDisconnect
		w.mu.Unlock()

		if cur && !prev && onConnect != nil {
			go onConnect()
		} else if !cur && prev && onDisconnect != nil {
			go onDisconnect()
		}
		return 0
	})
	return w
}

func (w *TunWatcher) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return
	}
	w.prev = hasTunAdapter()

	var handle uintptr
	procNotifyIPInterfaceChange.Call( //nolint:errcheck,gosec // see comment; G104 error is GetLastError
		afUnspec,
		w.callback,
		0,
		0,                                // no initial notification
		uintptr(unsafe.Pointer(&handle)), //nolint:gosec // required to pass &handle to the WinAPI callback registration
	)
	w.handle = handle
	w.running = true
}

func (w *TunWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	w.running = false
	if w.handle != 0 {
		procCancelMibChangeNotify2.Call(w.handle) //nolint:errcheck,gosec // LazyProc.Call error is GetLastError; nothing to recover on cancel.
		w.handle = 0
	}
}

func hasTunAdapter() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		if strings.Contains(strings.ToLower(iface.Name), "tun") {
			return true
		}
	}
	return false
}
