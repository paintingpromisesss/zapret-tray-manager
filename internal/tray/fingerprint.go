package tray

import (
	"hash/fnv"
	"strconv"

	"zapret-tray-manager/internal/app"
	"zapret-tray-manager/internal/config"
	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/zapretver"
)

// stateInputs is the complete set of state that applyStatus renders into the
// tray menu. If two refreshes produce an equal fingerprint, the rendered menu
// would be identical, so applyStatus can be skipped.
type stateInputs struct {
	status           manager.Status
	cfg              config.Config
	autostartEnabled bool
	autostartErr     bool
	releases         []zapretver.Release
	localRoots       []app.LocalZapretRoot
	userRoots        []app.UserLocalRoot
	downloaded       []bool // parallel to releases; release i is downloaded
	releaseRoots     []string
	busy             bool
	pendingUpdateVer string
}

// fingerprint hashes every input that influences the menu into a single
// uint64. FNV-64a over a flat textual encoding: cheap, allocation-light, and
// avoids reflection or deep struct comparison.
func (in stateInputs) fingerprint() uint64 {
	h := fnv.New64a()
	w := func(s string) {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0}) // field separator so "a"+"b" != "ab"
	}
	wb := func(b bool) {
		if b {
			w("1")
		} else {
			w("0")
		}
	}

	st := in.status
	wb(st.Valid)
	if st.ValidationError != nil {
		w(st.ValidationError.Error())
	} else {
		w("")
	}
	w(string(st.ServiceStatus.Zapret))
	w(st.ServiceStatus.InstalledStrategy)
	w(string(st.GameFilterMode))
	w(string(st.IPSetMode))
	wb(st.IPSetError != nil)
	for _, s := range st.Strategies {
		w(s.Name)
		wb(s.Custom)
	}
	w(strconv.Itoa(len(st.Strategies)))

	c := in.cfg
	wb(c.ZapretAutoRunEnabled)
	wb(c.GlobalSettingsEnabled)
	wb(c.VPNManageEnabled)
	w(c.CurrentRoot)

	wb(in.autostartEnabled)
	wb(in.autostartErr)
	wb(in.busy)
	w(in.pendingUpdateVer)

	for i, r := range in.releases {
		w(r.Version)
		w(r.AssetURL)
		w(in.releaseRoots[i])
		wb(in.downloaded[i])
	}
	w(strconv.Itoa(len(in.releases)))

	for _, r := range in.localRoots {
		w(r.Version)
		w(r.Path)
	}
	w(strconv.Itoa(len(in.localRoots)))

	for _, r := range in.userRoots {
		w(r.Path)
		wb(r.Valid)
	}
	w(strconv.Itoa(len(in.userRoots)))

	return h.Sum64()
}
