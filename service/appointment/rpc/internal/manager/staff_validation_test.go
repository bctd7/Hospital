package manager

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
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
			if got := containsWindow(room, tt.item); got != tt.want {
				t.Fatalf("containsWindow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeClockRejectsSecondsAndNormalizes(t *testing.T) {
	value, minutes, err := normalizeClock(" 09:05 ", "open_time")
	if err != nil || value != "09:05" || minutes != 545 {
		t.Fatalf("normalizeClock() = %q, %d, %v", value, minutes, err)
	}
	if _, _, err := normalizeClock("09:05:30", "open_time"); err == nil {
		t.Fatal("normalizeClock accepted seconds")
	}
}

func TestNormalizeRoomLocationProducesCanonicalDisplayName(t *testing.T) {
	campusID := uuid.NewString()
	gotCampus, building, floor, roomNumber, displayName, err := normalizeRoomLocation(
		" "+campusID+" ", " 门诊楼 ", -1, " A_01 ",
	)
	if err != nil {
		t.Fatalf("normalizeRoomLocation() error = %v", err)
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
			if _, _, _, _, _, err := normalizeRoomLocation(tt.campusID, tt.building, tt.floor, tt.roomNumber); err == nil {
				t.Fatal("normalizeRoomLocation accepted invalid structure")
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
	var flights flightGroup
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
			value, found, err := loadCached(context.Background(), cache, &flights, "same-key", time.Minute, loader)
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
	var entry cachedValue[string]
	if err := json.Unmarshal(data, &entry); err != nil || entry.Value != "hot" {
		t.Fatalf("invalid cache entry: %s (%v)", data, err)
	}
}

func TestJitteredTTLStaysWithinTwentyPercent(t *testing.T) {
	base := 5 * time.Minute
	for range 100 {
		got := jitteredTTL(base)
		if got < base || got > base+base/5 {
			t.Fatalf("jitteredTTL() = %s", got)
		}
	}
}

func TestPatientManagerAcceptsPatientIdentityWithoutStaffRole(t *testing.T) {
	err := requirePatient(authn.Principal{AccountID: uuid.NewString(), AccountType: authn.AccountTypePatient})
	if err != nil {
		t.Fatalf("patient read rejected: %v", err)
	}
}
