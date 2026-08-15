package svc

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	bookingCleanupInterval = 5 * time.Second
	bookingCleanupBatch    = 100
)

func (s *ServiceContext) startBookingCleanup() {
	ctx, cancel := context.WithCancel(context.Background())
	s.bookingCleanupCancel = cancel
	s.bookingCleanupDone = make(chan struct{})
	go func() {
		defer close(s.bookingCleanupDone)
		s.cleanupExpiredBookings(ctx)
		ticker := time.NewTicker(bookingCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupExpiredBookings(ctx)
			}
		}
	}()
}

func (s *ServiceContext) cleanupExpiredBookings(ctx context.Context) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	now := time.Now().In(location)
	for {
		result, err := s.appointmentStore.CleanupExpiredBookings(ctx, now, bookingCleanupBatch)
		if err != nil {
			if ctx.Err() == nil {
				logx.Errorf("cleanup expired bookings: %v", err)
			}
			return
		}
		for _, departmentID := range result.DepartmentIDs {
			if s.bookingCache != nil {
				_ = s.bookingCache.BumpDepartment(ctx, departmentID)
			}
		}
		if result.Processed < bookingCleanupBatch {
			return
		}
	}
}
