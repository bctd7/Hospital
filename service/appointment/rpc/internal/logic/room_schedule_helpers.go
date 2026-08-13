package logic

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
	"hospital/service/appointment/rpc/internal/svc"
)

func roomScheduleRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, manager.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid room or schedule request")
	case errors.Is(err, manager.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, manager.ErrNotFound):
		return status.Error(codes.NotFound, "room or schedule data not found")
	case errors.Is(err, manager.ErrConflict):
		return status.Error(codes.AlreadyExists, "room or schedule conflict")
	case errors.Is(err, manager.ErrVersionConflict):
		return status.Error(codes.Aborted, "room or schedule version conflict")
	case errors.Is(err, manager.ErrWindowConflict):
		return status.Error(codes.FailedPrecondition, "item window must be fully contained by every active room window")
	case errors.Is(err, manager.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "room or schedule state does not allow the operation")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func operationMeta(operationID, requestID string) staffmanager.OperationMeta {
	return staffmanager.OperationMeta{OperationID: operationID, RequestID: requestID}
}
func statusCommand(in *appointmentv1.ChangeResourceStatusRequest) staffmanager.ChangeStatusCommand {
	return staffmanager.ChangeStatusCommand{ResourceID: in.ResourceId, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)}
}

func roomResponse(v manager.Room) *appointmentv1.Room {
	return &appointmentv1.Room{RoomId: v.RoomID, DepartmentId: v.DepartmentID, CampusId: v.CampusID, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, DisplayName: v.DisplayName, Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func relationResponse(v manager.RoomItem) *appointmentv1.RoomExaminationItem {
	return &appointmentv1.RoomExaminationItem{RelationId: v.RelationID, RoomId: v.RoomID, ItemId: v.ItemID, RoomDisplayName: v.RoomDisplayName, CampusId: v.CampusID, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, ItemName: v.ItemName, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func roomWindowResponse(v manager.RoomWeeklyWindow) *appointmentv1.RoomWeeklyWindow {
	return &appointmentv1.RoomWeeklyWindow{WindowId: v.WindowID, RoomId: v.RoomID, Weekday: v.Weekday, Session: string(v.Session), OpenTime: v.OpenTime, CloseTime: v.CloseTime, ActiveCapacity: v.ActiveCapacity, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func itemWindowResponse(v manager.ItemWeeklyWindow) *appointmentv1.ItemWeeklyWindow {
	return &appointmentv1.ItemWeeklyWindow{WindowId: v.WindowID, ItemId: v.ItemID, Weekday: v.Weekday, Session: string(v.Session), StartTime: v.StartTime, BookingCutoffTime: v.BookingCutoffTime, EndTime: v.EndTime, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}

func createRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.CreateRoomRequest) (*appointmentv1.Room, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.CreateRoom(ctx, p, staffmanager.CreateRoomCommand{DepartmentID: in.DepartmentId, CampusID: in.CampusId, Building: in.Building, FloorNumber: in.FloorNumber, RoomNumber: in.RoomNumber, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomScheduleRPCError(err)
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
		return nil, roomScheduleRPCError(err)
	}
	return roomResponse(v), nil
}
func listRooms(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListRoomsRequest) (*appointmentv1.ListRoomsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.ListRooms(ctx, p, staffmanager.ListRoomsQuery{DepartmentID: in.DepartmentId, Page: in.Page, PageSize: in.PageSize})
	if err != nil {
		return nil, roomScheduleRPCError(err)
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
	v, err := svcCtx.StaffManager.UpdateRoom(ctx, p, staffmanager.UpdateRoomCommand{RoomID: in.RoomId, CampusID: in.CampusId, Building: in.Building, FloorNumber: in.FloorNumber, RoomNumber: in.RoomNumber, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	return roomResponse(v), nil
}
func retireRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.RetireRoomRequest) (*appointmentv1.Room, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.RetireRoom(ctx, p, staffmanager.RetireRoomCommand{RoomID: in.RoomId, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	return roomResponse(v), nil
}
func addRoomItem(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.AddRoomExaminationItemRequest) (*appointmentv1.RoomExaminationItem, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.AddRoomItem(ctx, p, staffmanager.AddRoomItemCommand{RoomID: in.RoomId, ItemID: in.ItemId, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	return relationResponse(v), nil
}
func changeRoomItem(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest, enable bool) (*appointmentv1.RoomExaminationItem, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var v manager.RoomItem
	if enable {
		v, err = svcCtx.StaffManager.EnableRoomItem(ctx, p, statusCommand(in))
	} else {
		v, err = svcCtx.StaffManager.DisableRoomItem(ctx, p, statusCommand(in))
	}
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	return relationResponse(v), nil
}
func listRoomItems(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListRoomExaminationItemsRequest) (*appointmentv1.ListRoomExaminationItemsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.ListRoomItems(ctx, p, in.RoomId, staffmanager.ListRelationsQuery{Status: manager.Status(in.Status), Page: in.Page, PageSize: in.PageSize})
	if err != nil {
		return nil, roomScheduleRPCError(err)
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
	v, err := svcCtx.PatientManager.ListAvailableRooms(ctx, p, in.ItemId)
	if err != nil {
		return nil, roomScheduleRPCError(err)
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
	v, err := svcCtx.StaffManager.SetRoomWindow(ctx, p, staffmanager.SetRoomWindowCommand{WindowID: in.WindowId, RoomID: in.RoomId, Weekday: in.Weekday, Session: manager.Session(in.Session), OpenTime: in.OpenTime, CloseTime: in.CloseTime, ActiveCapacity: in.ActiveCapacity, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	return roomWindowResponse(v), nil
}
func disableRoomWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.RoomWeeklyWindow, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.DisableRoomWindow(ctx, p, statusCommand(in))
	if err != nil {
		return nil, roomScheduleRPCError(err)
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
		return nil, roomScheduleRPCError(err)
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
	v, err := svcCtx.StaffManager.SetItemWindow(ctx, p, staffmanager.SetItemWindowCommand{WindowID: in.WindowId, ItemID: in.ItemId, Weekday: in.Weekday, Session: manager.Session(in.Session), StartTime: in.StartTime, BookingCutoffTime: in.BookingCutoffTime, EndTime: in.EndTime, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	return itemWindowResponse(v), nil
}
func disableItemWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.ItemWeeklyWindow, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.StaffManager.DisableItemWindow(ctx, p, statusCommand(in))
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	return itemWindowResponse(v), nil
}
func listItemWindows(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListWeeklyWindowsRequest) (*appointmentv1.ListItemWeeklyWindowsResponse, error) {
	p, err := appointmentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var v []manager.ItemWeeklyWindow
	if p.AccountType == authn.AccountTypePatient {
		v, err = svcCtx.PatientManager.ListProjectWindows(ctx, p, in.ResourceId)
	} else {
		v, err = svcCtx.StaffManager.ListItemWindows(ctx, p, in.ResourceId, in.ActiveOnly)
	}
	if err != nil {
		return nil, roomScheduleRPCError(err)
	}
	out := &appointmentv1.ListItemWeeklyWindowsResponse{}
	for _, item := range v {
		out.Windows = append(out.Windows, itemWindowResponse(item))
	}
	return out, nil
}
