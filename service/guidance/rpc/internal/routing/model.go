package routing

// LocationPoint 是所有路线调用统一使用的地点契约。地图选点和医院楼栋坐标都转换成这个结构。
type LocationPoint struct {
	Name            string
	Address         string
	Latitude        float64
	Longitude       float64
	ProviderPlaceID string
}

type RoutePoint struct {
	Latitude  float64
	Longitude float64
}

type Step struct {
	Instruction     string
	RoadName        string
	DistanceMeters  int32
	DurationSeconds int32
}

type WalkingRoute struct {
	Origin          LocationPoint
	Destination     LocationPoint
	DistanceMeters  int32
	DurationSeconds int32
	Polyline        []RoutePoint
	Steps           []Step
	Provider        string
}
