package staff

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

func TestContainsWindowRequiresFullContainmentAndSameSlot(t *testing.T) {
	room := RoomWeeklyWindow{Weekday: 1, Session: SessionMorning, OpenTime: "09:00", CloseTime: "12:00", Status: StatusActive}
	tests := []struct {
		name string
		item ItemWeeklyWindow
		want bool
	}{
		{"contained", ItemWeeklyWindow{Weekday: 1, Session: SessionMorning, StartTime: "09:00", EndTime: "12:00", Status: StatusActive}, true},
		{"starts early", ItemWeeklyWindow{Weekday: 1, Session: SessionMorning, StartTime: "08:59", EndTime: "11:00", Status: StatusActive}, false},
		{"ends late", ItemWeeklyWindow{Weekday: 1, Session: SessionMorning, StartTime: "10:00", EndTime: "12:01", Status: StatusActive}, false},
		{"different session", ItemWeeklyWindow{Weekday: 1, Session: SessionAfternoon, StartTime: "09:00", EndTime: "12:00", Status: StatusActive}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := staffsupport.ContainsWindow(room, tt.item); got != tt.want {
				t.Fatalf("containsWindow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeClockRejectsSecondsAndNormalizes(t *testing.T) {
	value, minutes, err := staffsupport.NormalizeClock(" 09:05 ", "open_time")
	if err != nil || value != "09:05" || minutes != 545 {
		t.Fatalf("staffsupport.NormalizeClock() = %q, %d, %v", value, minutes, err)
	}
	if _, _, err := staffsupport.NormalizeClock("09:05:30", "open_time"); err == nil {
		t.Fatal("staffsupport.NormalizeClock accepted seconds")
	}
}

func TestFitsSessionBoundaryUsesNoonAsTheOnlySplit(t *testing.T) {
	tests := []struct {
		name    string
		session Session
		start   int
		end     int
		want    bool
	}{
		{name: "morning may end at noon", session: SessionMorning, start: 9 * 60, end: 12 * 60, want: true},
		{name: "morning may not cross noon", session: SessionMorning, start: 11 * 60, end: 13 * 60, want: false},
		{name: "afternoon may start at noon", session: SessionAfternoon, start: 12 * 60, end: 13 * 60, want: true},
		{name: "afternoon may not start before noon", session: SessionAfternoon, start: 11 * 60, end: 12 * 60, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := staffsupport.FitsSessionBoundary(tt.session, tt.start, tt.end); got != tt.want {
				t.Fatalf("staffsupport.FitsSessionBoundary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeRoomLocationProducesCanonicalDisplayName(t *testing.T) {
	campusID := uuid.NewString()
	gotCampus, building, floor, roomNumber, displayName, err := staffsupport.NormalizeRoomLocation(
		" "+campusID+" ", " 门诊楼 ", -1, " A_01 ",
	)
	if err != nil {
		t.Fatalf("staffsupport.NormalizeRoomLocation() error = %v", err)
	}
	if gotCampus != campusID || building != "门诊楼" || floor != -1 || roomNumber != "A_01" {
		t.Fatalf("unexpected normalized location: %q, %q, %d, %q", gotCampus, building, floor, roomNumber)
	}
	if displayName != "门诊楼 · B1层 · A_01室" {
		t.Fatalf("display name = %q", displayName)
	}
}

func TestNormalizeRoomLocationRejectsInvalidStructure(t *testing.T) {
	campusID := uuid.NewString()
	tests := []struct {
		name       string
		campusID   string
		building   string
		floor      int32
		roomNumber string
	}{
		{name: "campus must be uuid", campusID: "north-campus", building: "门诊楼", floor: 1, roomNumber: "101"},
		{name: "floor cannot be zero", campusID: campusID, building: "门诊楼", floor: 0, roomNumber: "101"},
		{name: "building required", campusID: campusID, building: " ", floor: 1, roomNumber: "101"},
		{name: "room number has strict characters", campusID: campusID, building: "门诊楼", floor: 1, roomNumber: "10 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, _, _, _, err := staffsupport.NormalizeRoomLocation(tt.campusID, tt.building, tt.floor, tt.roomNumber); err == nil {
				t.Fatal("staffsupport.NormalizeRoomLocation accepted invalid structure")
			}
		})
	}
}

type memoryCache struct {
	mu     sync.Mutex
	values map[string][]byte
}

func (c *memoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.values[key]
	return append([]byte(nil), v...), ok, nil
}
func (c *memoryCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.values == nil {
		c.values = map[string][]byte{}
	}
	c.values[key] = append([]byte(nil), value...)
	return nil
}
func (c *memoryCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		delete(c.values, key)
	}
	return nil
}
func (c *memoryCache) DepartmentGeneration(context.Context, string) (string, error) { return "1", nil }
func (c *memoryCache) BumpDepartment(context.Context, string) error                 { return nil }

func TestLoadCachedCollapsesConcurrentMisses(t *testing.T) {
	cache := &memoryCache{values: map[string][]byte{}}
	var flights staffsupport.FlightGroup
	var loads atomic.Int32
	start := make(chan struct{})
	loader := func() (string, bool, error) { loads.Add(1); <-start; return "hot", true, nil }
	const callers = 12
	var wg sync.WaitGroup
	wg.Add(callers)
	results := make(chan string, callers)
	for range callers {
		go func() {
			defer wg.Done()
			value, found, err := staffsupport.LoadCached(context.Background(), cache, &flights, "same-key", time.Minute, loader)
			if err != nil || !found {
				results <- "error"
				return
			}
			results <- value
		}()
	}
	for loads.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(start)
	wg.Wait()
	close(results)
	for value := range results {
		if value != "hot" {
			t.Fatalf("unexpected cached value %q", value)
		}
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("loader called %d times, want 1", got)
	}
	data, found, _ := cache.Get(context.Background(), "same-key")
	if !found {
		t.Fatal("result was not cached")
	}
	var entry struct {
		Found bool   `json:"found"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &entry); err != nil || entry.Value != "hot" {
		t.Fatalf("invalid cache entry: %s (%v)", data, err)
	}
}

func TestJitteredTTLStaysWithinTwentyPercent(t *testing.T) {
	base := 5 * time.Minute
	for range 100 {
		got := staffsupport.JitteredTTL(base)
		if got < base || got > base+base/5 {
			t.Fatalf("jitteredTTL() = %s", got)
		}
	}
}
