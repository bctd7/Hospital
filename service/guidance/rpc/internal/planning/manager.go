package planning

import (
	"errors"
	"sync"
	"time"
)

// Manager 是患者方案编排入口。生成、确认和当日推荐分别位于
// generate.go、confirm.go 和 today.go，避免把三条流程塞在同一文件。
type Manager struct {
	store       Store
	appointment Appointment
	travel      TravelEstimator
	travelMu    sync.RWMutex
	travelCache map[string]cachedTravelEstimate
}

type cachedTravelEstimate struct {
	value     travelEstimate
	expiresAt time.Time
}

func NewManager(store Store, appointment Appointment, travel TravelEstimator) (*Manager, error) {
	if store == nil || appointment == nil {
		return nil, errors.New("planning dependencies are required")
	}
	return &Manager{store: store, appointment: appointment, travel: travel, travelCache: map[string]cachedTravelEstimate{}}, nil
}
