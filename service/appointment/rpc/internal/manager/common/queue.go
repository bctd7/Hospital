package common

import "time"

const CallTimeout = time.Minute

type CheckQueue struct {
	QueueID                string
	RoomID                 string
	ServiceDate            time.Time
	Session                Session
	NextTicketNumber       int64
	CallSequence           int64
	CurrentCalledBookingID string
	Version                int64
}

type QueueEntry struct {
	BookingID                 string
	QueueID                   string
	TicketNumber              int64
	EligibleAfterCallSequence int64
	CallAttempts              int64
	CheckedInAt               time.Time
	CalledAt                  *time.Time
	CallDeadline              *time.Time
}

type QueueEventType string

const (
	QueueEventCheckedIn QueueEventType = "checked_in"
	QueueEventCalled    QueueEventType = "called"
	QueueEventDeferred  QueueEventType = "deferred"
	QueueEventStarted   QueueEventType = "started"
	QueueEventEnded     QueueEventType = "ended"
)

type QueueEvent struct {
	EventID      string
	QueueID      string
	BookingID    string
	EventType    QueueEventType
	CallSequence int64
	OccurredAt   time.Time
}
