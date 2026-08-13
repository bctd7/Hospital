package logic

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"
)

func roomProjectRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, common.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid room or project configuration request")
	case errors.Is(err, common.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, common.ErrNotFound):
		return status.Error(codes.NotFound, "room or project configuration data not found")
	case errors.Is(err, common.ErrConflict):
		return status.Error(codes.AlreadyExists, "room or project configuration conflict")
	case errors.Is(err, common.ErrVersionConflict):
		return status.Error(codes.Aborted, "room or project configuration version conflict")
	case errors.Is(err, common.ErrWindowConflict):
		return status.Error(codes.FailedPrecondition, "item window must be fully contained by every active room window")
	case errors.Is(err, common.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "room or project configuration state does not allow the operation")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func operationInput(operationID, requestID string) staffinput.Operation {
	return staffinput.Operation{OperationID: operationID, RequestID: requestID}
}
func statusInput(in *appointmentv1.ChangeResourceStatusRequest) staffinput.ChangeStatus {
	return staffinput.ChangeStatus{ResourceID: in.ResourceId, ExpectedVersion: in.ExpectedVersion, Operation: operationInput(in.OperationId, in.RequestId)}
}

func roomResponse(v common.Room) *appointmentv1.Room {
	return &appointmentv1.Room{RoomId: v.RoomID, DepartmentId: v.DepartmentID, CampusId: v.CampusID, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, DisplayName: v.DisplayName, Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func relationResponse(v common.RoomItem) *appointmentv1.RoomExaminationItem {
	return &appointmentv1.RoomExaminationItem{RelationId: v.RelationID, RoomId: v.RoomID, ItemId: v.ItemID, RoomDisplayName: v.RoomDisplayName, CampusId: v.CampusID, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, ItemName: v.ItemName, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func roomWindowResponse(v common.RoomWeeklyWindow) *appointmentv1.RoomWeeklyWindow {
	return &appointmentv1.RoomWeeklyWindow{WindowId: v.WindowID, RoomId: v.RoomID, Weekday: v.Weekday, Session: string(v.Session), OpenTime: v.OpenTime, CloseTime: v.CloseTime, ActiveCapacity: v.ActiveCapacity, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func itemWindowResponse(v common.ItemWeeklyWindow) *appointmentv1.ItemWeeklyWindow {
	return &appointmentv1.ItemWeeklyWindow{WindowId: v.WindowID, ItemId: v.ItemID, Weekday: v.Weekday, Session: string(v.Session), StartTime: v.StartTime, BookingCutoffTime: v.BookingCutoffTime, EndTime: v.EndTime, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}

func createRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.CreateRoomRequest) (*appointmentv1.Room, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.CreateRoom(ctx, p, staffinput.CreateRoom{DepartmentID: in.DepartmentId, CampusID: in.CampusId, Building: in.Building, FloorNumber: in.FloorNumber, RoomNumber: in.RoomNumber, Operation: operationInput(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return roomResponse(v), nil
}
func getRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.GetRoomRequest) (*appointmentv1.Room, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.GetRoom(ctx, p, in.RoomId)
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return roomResponse(v), nil
}
func listRooms(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListRoomsRequest) (*appointmentv1.ListRoomsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.ListRooms(ctx, p, staffinput.ListRooms{DepartmentID: in.DepartmentId, Page: in.Page, PageSize: in.PageSize})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	out := &appointmentv1.ListRoomsResponse{Page: v.Page, PageSize: v.PageSize, Total: v.Total}
	for _, item := range v.Items {
		out.Rooms = append(out.Rooms, roomResponse(item))
	}
	return out, nil
}
func updateRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.UpdateRoomRequest) (*appointmentv1.Room, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.UpdateRoom(ctx, p, staffinput.UpdateRoom{RoomID: in.RoomId, CampusID: in.CampusId, Building: in.Building, FloorNumber: in.FloorNumber, RoomNumber: in.RoomNumber, ExpectedVersion: in.ExpectedVersion, Operation: operationInput(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return roomResponse(v), nil
}
func retireRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.RetireRoomRequest) (*appointmentv1.Room, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.RetireRoom(ctx, p, staffinput.RetireRoom{RoomID: in.RoomId, ExpectedVersion: in.ExpectedVersion, Operation: operationInput(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return roomResponse(v), nil
}
func addRoomItem(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.AddRoomExaminationItemRequest) (*appointmentv1.RoomExaminationItem, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.AddRoomItem(ctx, p, staffinput.AddRoomItem{RoomID: in.RoomId, ItemID: in.ItemId, Operation: operationInput(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return relationResponse(v), nil
}
func changeRoomItem(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest, enable bool) (*appointmentv1.RoomExaminationItem, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var v common.RoomItem
	if enable {
		v, err = svcCtx.StaffManager.EnableRoomItem(ctx, p, statusInput(in))
	} else {
		v, err = svcCtx.StaffManager.DisableRoomItem(ctx, p, statusInput(in))
	}
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return relationResponse(v), nil
}
func listRoomItems(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListRoomExaminationItemsRequest) (*appointmentv1.ListRoomExaminationItemsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.ListRoomItems(ctx, p, in.RoomId, staffinput.ListRelations{Status: common.Status(in.Status), Page: in.Page, PageSize: in.PageSize})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	out := &appointmentv1.ListRoomExaminationItemsResponse{Page: v.Page, PageSize: v.PageSize, Total: v.Total}
	for _, item := range v.Items {
		out.Relations = append(out.Relations, relationResponse(item))
	}
	return out, nil
}
func listItemRooms(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListAvailableRoomsByExaminationItemRequest) (*appointmentv1.ListRoomExaminationItemsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.SharedManager.ListPatientAvailableRooms(ctx, p, in.ItemId)
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	out := &appointmentv1.ListRoomExaminationItemsResponse{Page: 1, PageSize: int64(len(v)), Total: int64(len(v))}
	for _, item := range v {
		out.Relations = append(out.Relations, relationResponse(item))
	}
	return out, nil
}
func setRoomWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.SetRoomWeeklyWindowRequest) (*appointmentv1.RoomWeeklyWindow, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.SetRoomWindow(ctx, p, staffinput.SetRoomWindow{WindowID: in.WindowId, RoomID: in.RoomId, Weekday: in.Weekday, Session: common.Session(in.Session), OpenTime: in.OpenTime, CloseTime: in.CloseTime, ActiveCapacity: in.ActiveCapacity, ExpectedVersion: in.ExpectedVersion, Operation: operationInput(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return roomWindowResponse(v), nil
}
func disableRoomWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.RoomWeeklyWindow, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.DisableRoomWindow(ctx, p, statusInput(in))
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return roomWindowResponse(v), nil
}
func listRoomWindows(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListWeeklyWindowsRequest) (*appointmentv1.ListRoomWeeklyWindowsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.ListRoomWindows(ctx, p, in.ResourceId, in.ActiveOnly)
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	out := &appointmentv1.ListRoomWeeklyWindowsResponse{}
	for _, item := range v {
		out.Windows = append(out.Windows, roomWindowResponse(item))
	}
	return out, nil
}
func setItemWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.SetItemWeeklyWindowRequest) (*appointmentv1.ItemWeeklyWindow, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.SetItemWindow(ctx, p, staffinput.SetItemWindow{WindowID: in.WindowId, ItemID: in.ItemId, Weekday: in.Weekday, Session: common.Session(in.Session), StartTime: in.StartTime, BookingCutoffTime: in.BookingCutoffTime, EndTime: in.EndTime, ExpectedVersion: in.ExpectedVersion, Operation: operationInput(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return itemWindowResponse(v), nil
}
func disableItemWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.ItemWeeklyWindow, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.DisableItemWindow(ctx, p, statusInput(in))
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	return itemWindowResponse(v), nil
}
func listItemWindows(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListWeeklyWindowsRequest) (*appointmentv1.ListItemWeeklyWindowsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var v []common.ItemWeeklyWindow
	if p.AccountType == authn.AccountTypePatient {
		v, err = svcCtx.SharedManager.ListPatientProjectWindows(ctx, p, in.ResourceId)
	} else {
		v, err = svcCtx.StaffManager.ListItemWindows(ctx, p, in.ResourceId, in.ActiveOnly)
	}
	if err != nil {
		return nil, roomProjectRPCError(err)
	}
	out := &appointmentv1.ListItemWeeklyWindowsResponse{}
	for _, item := range v {
		out.Windows = append(out.Windows, itemWindowResponse(item))
	}
	return out, nil
}
