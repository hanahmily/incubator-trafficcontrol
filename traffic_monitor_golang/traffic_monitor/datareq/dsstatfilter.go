package datareq

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	dsdata "github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/deliveryservicedata"
	"github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/enum"
)

// DSStatFilter fulfills the cache.Filter interface, for filtering stats. See the `NewDSStatFilter` documentation for details on which query parameters are used to filter.
type DSStatFilter struct {
	historyCount     int
	statsToUse       map[string]struct{}
	wildcard         bool
	dsType           enum.DSType
	deliveryServices map[enum.DeliveryServiceName]struct{}
	dsTypes          map[enum.DeliveryServiceName]enum.DSType
}

// UseDeliveryService returns whether the given delivery service is in this filter.
func (f *DSStatFilter) UseDeliveryService(name enum.DeliveryServiceName) bool {
	if _, inDSes := f.deliveryServices[name]; len(f.deliveryServices) != 0 && !inDSes {
		return false
	}
	if f.dsType != enum.DSTypeInvalid && f.dsTypes[name] != f.dsType {
		return false
	}
	return true
}

// UseStat returns whether the given stat is in this filter.
func (f *DSStatFilter) UseStat(statName string) bool {
	if len(f.statsToUse) == 0 {
		return true
	}
	if !f.wildcard {
		_, ok := f.statsToUse[statName]
		return ok
	}
	for statToUse := range f.statsToUse {
		if strings.Contains(statName, statToUse) {
			return true
		}
	}
	return false
}

// WithinStatHistoryMax returns whether the given history index is less than the max history of this filter.
func (f *DSStatFilter) WithinStatHistoryMax(n int) bool {
	if f.historyCount == 0 {
		return true
	}
	if n <= f.historyCount {
		return true
	}
	return false
}

// NewDSStatFilter takes the HTTP query parameters and creates a cache.Filter, filtering according to the query parameters passed.
// Query parameters used are `hc`, `stats`, `wildcard`, `type`, and `deliveryservices`.
// If `hc` is 0, all history is returned. If `hc` is empty, 1 history is returned.
// If `stats` is empty, all stats are returned.
// If `wildcard` is empty, `stats` is considered exact.
// If `type` is empty, all types are returned.
func NewDSStatFilter(path string, params url.Values, dsTypes map[enum.DeliveryServiceName]enum.DSType) (dsdata.Filter, error) {
	validParams := map[string]struct{}{"hc": struct{}{}, "stats": struct{}{}, "wildcard": struct{}{}, "type": struct{}{}, "deliveryservices": struct{}{}}
	if len(params) > len(validParams) {
		return nil, fmt.Errorf("invalid query parameters")
	}
	for param := range params {
		if _, ok := validParams[param]; !ok {
			return nil, fmt.Errorf("invalid query parameter '%v'", param)
		}
	}

	historyCount := 1
	if paramHc, exists := params["hc"]; exists && len(paramHc) > 0 {
		v, err := strconv.Atoi(paramHc[0])
		if err == nil {
			historyCount = v
		}
	}

	statsToUse := map[string]struct{}{}
	if paramStats, exists := params["stats"]; exists && len(paramStats) > 0 {
		commaStats := strings.Split(paramStats[0], ",")
		for _, stat := range commaStats {
			statsToUse[stat] = struct{}{}
		}
	}

	wildcard := false
	if paramWildcard, exists := params["wildcard"]; exists && len(paramWildcard) > 0 {
		wildcard, _ = strconv.ParseBool(paramWildcard[0]) // ignore errors, error => false
	}

	dsType := enum.DSTypeInvalid
	if paramType, exists := params["type"]; exists && len(paramType) > 0 {
		dsType = enum.DSTypeFromString(paramType[0])
		if dsType == enum.DSTypeInvalid {
			return nil, fmt.Errorf("invalid query parameter type '%v' - valid types are: {http, dns}", paramType[0])
		}
	}

	deliveryServices := map[enum.DeliveryServiceName]struct{}{}
	// TODO rename 'hosts' to 'names' for consistency
	if paramNames, exists := params["deliveryservices"]; exists && len(paramNames) > 0 {
		commaNames := strings.Split(paramNames[0], ",")
		for _, name := range commaNames {
			deliveryServices[enum.DeliveryServiceName(name)] = struct{}{}
		}
	}

	pathArgument := getPathArgument(path)
	if pathArgument != "" {
		deliveryServices[enum.DeliveryServiceName(pathArgument)] = struct{}{}
	}

	// parameters without values are considered names, e.g. `?my-cache-0` or `?my-delivery-service`
	for maybeName, val := range params {
		if len(val) == 0 || (len(val) == 1 && val[0] == "") {
			deliveryServices[enum.DeliveryServiceName(maybeName)] = struct{}{}
		}
	}

	return &DSStatFilter{
		historyCount:     historyCount,
		statsToUse:       statsToUse,
		wildcard:         wildcard,
		dsType:           dsType,
		deliveryServices: deliveryServices,
		dsTypes:          dsTypes,
	}, nil
}