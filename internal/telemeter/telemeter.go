package telemeter

import (
	"errors"
	"gateway/internal/config"
	"sync/atomic"
)

type Recorder interface {
	Record(serviceName, path string)
	GetTelemetery(serviceName, path string) (uint64, uint64, error)
}

type requestTelemeter struct {
	serviceMap map[string]struct {
		traffic   *atomic.Uint64
		routesMap map[string]*atomic.Uint64
	}
}

func NewReqTelemeter(scm config.ServiceConfigMap) *requestTelemeter {
	rt := requestTelemeter{
		serviceMap: make(map[string]struct {
			traffic   *atomic.Uint64
			routesMap map[string]*atomic.Uint64
		}),
	}

	for _, s := range scm {
		rt.serviceMap[s.ServiceName] = struct {
			traffic   *atomic.Uint64
			routesMap map[string]*atomic.Uint64
		}{
			traffic:   &atomic.Uint64{},
			routesMap: make(map[string]*atomic.Uint64),
		}

	}

	return &rt
}

func (rt *requestTelemeter) Record(serviceName, path string) {
	st, ok := rt.serviceMap[serviceName]
	if !ok {
		return
	}

	routeTraffic, ok := st.routesMap[path]
	if !ok {
		routeTraffic = &atomic.Uint64{}
		st.routesMap[path] = routeTraffic
	}

	st.traffic.Add(1)
	routeTraffic.Add(1)

}

func (rt *requestTelemeter) GetTelemetery(serviceName, path string) (uint64, uint64, error) {
	st, ok := rt.serviceMap[serviceName]
	if !ok {
		return 0, 0, errors.New("service not found for telemetric data")
	}

	routeTraffic, ok := st.routesMap[path]
	if !ok {
		return 0, 0, errors.New("route not found for telemetric data")
	}

	return st.traffic.Load(), routeTraffic.Load(), nil

}
