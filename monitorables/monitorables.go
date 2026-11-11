package monitorables

import (
	"github.com/monitoror/monitoror/monitorables/http"
	"github.com/monitoror/monitoror/store"
)

func RegisterMonitorables(s *store.Store) {
	// ------------ HTTP ------------
	s.Registry.RegisterMonitorable(http.NewMonitorable(s))
}
