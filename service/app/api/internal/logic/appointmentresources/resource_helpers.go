package appointmentresources

import (
	"context"

	"hospital/common/observability/logging"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func rpcContext(ctx context.Context, svcCtx *svc.ServiceContext) (context.Context, string, error) {
	rpcCtx, err := svcCtx.AuthenticatedRPCContext(ctx)
	return rpcCtx, logging.RequestIDFromContext(ctx), err
}
func room(v *appointmentv1.Room) *types.AppointmentRoomResponse {
	return &types.AppointmentRoomResponse{RoomID: v.RoomId, DepartmentID: v.DepartmentId, CampusID: v.CampusId, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, DisplayName: v.DisplayName, Version: v.Version, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}
func relation(v *appointmentv1.RoomExaminationItem) types.RoomExaminationItemResponse {
	return types.RoomExaminationItemResponse{RelationID: v.RelationId, RoomID: v.RoomId, ItemID: v.ItemId, RoomDisplayName: v.RoomDisplayName, CampusID: v.CampusId, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, ItemName: v.ItemName, Status: v.Status, Version: v.Version, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}
func roomWindow(v *appointmentv1.RoomWeeklyWindow) *types.RoomWeeklyWindowResponse {
	return &types.RoomWeeklyWindowResponse{WindowID: v.WindowId, RoomID: v.RoomId, Weekday: v.Weekday, Session: v.Session, OpenTime: v.OpenTime, CloseTime: v.CloseTime, ActiveCapacity: v.ActiveCapacity, Status: v.Status, Version: v.Version, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}
func itemWindow(v *appointmentv1.ItemWeeklyWindow) *types.ItemWeeklyWindowResponse {
	return &types.ItemWeeklyWindowResponse{WindowID: v.WindowId, ItemID: v.ItemId, Weekday: v.Weekday, Session: v.Session, StartTime: v.StartTime, BookingCutoffTime: v.BookingCutoffTime, EndTime: v.EndTime, Status: v.Status, Version: v.Version, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

func createRoom(ctx context.Context, s *svc.ServiceContext, req *types.CreateAppointmentRoomRequest) (*types.AppointmentRoomResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.CreateRoom(rpcCtx, &appointmentv1.CreateRoomRequest{DepartmentId: req.DepartmentID, CampusId: req.CampusID, Building: req.Building, FloorNumber: req.FloorNumber, RoomNumber: req.RoomNumber, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return room(v), nil
}
func getRoom(ctx context.Context, s *svc.ServiceContext, req *types.AppointmentRoomPathRequest) (*types.AppointmentRoomResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.GetRoom(rpcCtx, &appointmentv1.GetRoomRequest{RoomId: req.RoomID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return room(v), nil
}
func listRooms(ctx context.Context, s *svc.ServiceContext, req *types.ListAppointmentRoomsRequest) (*types.ListAppointmentRoomsResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.ListRooms(rpcCtx, &appointmentv1.ListRoomsRequest{DepartmentId: req.DepartmentID, Page: req.Page, PageSize: req.PageSize, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	out := &types.ListAppointmentRoomsResponse{Page: v.Page, PageSize: v.PageSize, Total: v.Total}
	for _, x := range v.Rooms {
		out.Rooms = append(out.Rooms, *room(x))
	}
	return out, nil
}
func updateRoom(ctx context.Context, s *svc.ServiceContext, req *types.UpdateAppointmentRoomRequest) (*types.AppointmentRoomResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.UpdateRoom(rpcCtx, &appointmentv1.UpdateRoomRequest{RoomId: req.RoomID, CampusId: req.CampusID, Building: req.Building, FloorNumber: req.FloorNumber, RoomNumber: req.RoomNumber, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return room(v), nil
}
func retireRoom(ctx context.Context, s *svc.ServiceContext, req *types.ChangeAppointmentResourceStatusRequest) (*types.AppointmentRoomResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.RetireRoom(rpcCtx, &appointmentv1.RetireRoomRequest{RoomId: req.ResourceID, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return room(v), nil
}
func addRoomItem(ctx context.Context, s *svc.ServiceContext, req *types.AddRoomExaminationItemAPIRequest) (*types.RoomExaminationItemResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.AddRoomExaminationItem(rpcCtx, &appointmentv1.AddRoomExaminationItemRequest{RoomId: req.RoomID, ItemId: req.ItemID, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	out := relation(v)
	return &out, nil
}
func changeRoomItem(ctx context.Context, s *svc.ServiceContext, req *types.ChangeAppointmentResourceStatusRequest, enable bool) (*types.RoomExaminationItemResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	in := &appointmentv1.ChangeResourceStatusRequest{ResourceId: req.ResourceID, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID}
	var v *appointmentv1.RoomExaminationItem
	if enable {
		v, err = s.Appointment.EnableRoomExaminationItem(rpcCtx, in)
	} else {
		v, err = s.Appointment.DisableRoomExaminationItem(rpcCtx, in)
	}
	if err != nil {
		return nil, err
	}
	out := relation(v)
	return &out, nil
}
func listRoomItems(ctx context.Context, s *svc.ServiceContext, req *types.ListRoomExaminationItemsAPIRequest) (*types.ListRoomExaminationItemsAPIResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.ListRoomExaminationItems(rpcCtx, &appointmentv1.ListRoomExaminationItemsRequest{RoomId: req.RoomID, Status: req.Status, Page: req.Page, PageSize: req.PageSize, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	out := &types.ListRoomExaminationItemsAPIResponse{Page: v.Page, PageSize: v.PageSize, Total: v.Total}
	for _, x := range v.Relations {
		out.Relations = append(out.Relations, relation(x))
	}
	return out, nil
}
func listItemRooms(ctx context.Context, s *svc.ServiceContext, req *types.ItemRoomsPathRequest) (*types.ListRoomExaminationItemsAPIResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.ListAvailableRoomsByExaminationItem(rpcCtx, &appointmentv1.ListAvailableRoomsByExaminationItemRequest{ItemId: req.ItemID, ActiveOnly: true, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	out := &types.ListRoomExaminationItemsAPIResponse{Page: v.Page, PageSize: v.PageSize, Total: v.Total}
	for _, x := range v.Relations {
		out.Relations = append(out.Relations, relation(x))
	}
	return out, nil
}
func setRoomWindow(ctx context.Context, s *svc.ServiceContext, req *types.SetRoomWeeklyWindowAPIRequest) (*types.RoomWeeklyWindowResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.SetRoomWeeklyWindow(rpcCtx, &appointmentv1.SetRoomWeeklyWindowRequest{WindowId: req.WindowID, RoomId: req.RoomID, Weekday: req.Weekday, Session: req.Session, OpenTime: req.OpenTime, CloseTime: req.CloseTime, ActiveCapacity: req.ActiveCapacity, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return roomWindow(v), nil
}
func disableRoomWindow(ctx context.Context, s *svc.ServiceContext, req *types.ChangeAppointmentResourceStatusRequest) (*types.RoomWeeklyWindowResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.DisableRoomWeeklyWindow(rpcCtx, &appointmentv1.ChangeResourceStatusRequest{ResourceId: req.ResourceID, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return roomWindow(v), nil
}
func listRoomWindows(ctx context.Context, s *svc.ServiceContext, req *types.WeeklyWindowsPathRequest) (*types.ListRoomWeeklyWindowsAPIResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.ListRoomWeeklyWindows(rpcCtx, &appointmentv1.ListWeeklyWindowsRequest{ResourceId: req.ResourceID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	out := &types.ListRoomWeeklyWindowsAPIResponse{}
	for _, x := range v.Windows {
		out.Windows = append(out.Windows, *roomWindow(x))
	}
	return out, nil
}
func setItemWindow(ctx context.Context, s *svc.ServiceContext, req *types.SetItemWeeklyWindowAPIRequest) (*types.ItemWeeklyWindowResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.SetItemWeeklyWindow(rpcCtx, &appointmentv1.SetItemWeeklyWindowRequest{WindowId: req.WindowID, ItemId: req.ItemID, Weekday: req.Weekday, Session: req.Session, StartTime: req.StartTime, BookingCutoffTime: req.BookingCutoffTime, EndTime: req.EndTime, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return itemWindow(v), nil
}
func disableItemWindow(ctx context.Context, s *svc.ServiceContext, req *types.ChangeAppointmentResourceStatusRequest) (*types.ItemWeeklyWindowResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.DisableItemWeeklyWindow(rpcCtx, &appointmentv1.ChangeResourceStatusRequest{ResourceId: req.ResourceID, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return itemWindow(v), nil
}
func listItemWindows(ctx context.Context, s *svc.ServiceContext, req *types.WeeklyWindowsPathRequest) (*types.ListItemWeeklyWindowsAPIResponse, error) {
	rpcCtx, requestID, err := rpcContext(ctx, s)
	if err != nil {
		return nil, err
	}
	v, err := s.Appointment.ListItemWeeklyWindows(rpcCtx, &appointmentv1.ListWeeklyWindowsRequest{ResourceId: req.ResourceID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	out := &types.ListItemWeeklyWindowsAPIResponse{}
	for _, x := range v.Windows {
		out.Windows = append(out.Windows, *itemWindow(x))
	}
	return out, nil
}
