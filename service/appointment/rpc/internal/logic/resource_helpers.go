package logic

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/resource"
	"hospital/service/appointment/rpc/internal/svc"
)

func resourceRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, resource.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid appointment resource request")
	case errors.Is(err, resource.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, resource.ErrNotFound):
		return status.Error(codes.NotFound, "appointment resource not found")
	case errors.Is(err, resource.ErrConflict):
		return status.Error(codes.AlreadyExists, "appointment resource conflict")
	case errors.Is(err, resource.ErrVersionConflict):
		return status.Error(codes.Aborted, "appointment resource version conflict")
	case errors.Is(err, resource.ErrWindowConflict):
		return status.Error(codes.FailedPrecondition, "item window must be fully contained by every active room window")
	case errors.Is(err, resource.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "appointment resource state does not allow the operation")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func operationMeta(operationID, requestID string) resource.OperationMeta {
	return resource.OperationMeta{OperationID: operationID, RequestID: requestID}
}
func statusCommand(in *appointmentv1.ChangeResourceStatusRequest) resource.ChangeStatusCommand {
	return resource.ChangeStatusCommand{ResourceID: in.ResourceId, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)}
}

func roomResponse(v resource.Room) *appointmentv1.Room {
	return &appointmentv1.Room{RoomId: v.RoomID, DepartmentId: v.DepartmentID, CampusId: v.CampusID, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, DisplayName: v.DisplayName, Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func relationResponse(v resource.RoomItem) *appointmentv1.RoomExaminationItem {
	return &appointmentv1.RoomExaminationItem{RelationId: v.RelationID, RoomId: v.RoomID, ItemId: v.ItemID, RoomDisplayName: v.RoomDisplayName, CampusId: v.CampusID, Building: v.Building, FloorNumber: v.FloorNumber, RoomNumber: v.RoomNumber, ItemName: v.ItemName, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func roomWindowResponse(v resource.RoomWeeklyWindow) *appointmentv1.RoomWeeklyWindow {
	return &appointmentv1.RoomWeeklyWindow{WindowId: v.WindowID, RoomId: v.RoomID, Weekday: v.Weekday, Session: string(v.Session), OpenTime: v.OpenTime, CloseTime: v.CloseTime, ActiveCapacity: v.ActiveCapacity, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}
func itemWindowResponse(v resource.ItemWeeklyWindow) *appointmentv1.ItemWeeklyWindow {
	return &appointmentv1.ItemWeeklyWindow{WindowId: v.WindowID, ItemId: v.ItemID, Weekday: v.Weekday, Session: string(v.Session), StartTime: v.StartTime, BookingCutoffTime: v.BookingCutoffTime, EndTime: v.EndTime, Status: string(v.Status), Version: v.Version, CreatedAt: v.CreatedAt.UTC().Format(timeLayout), UpdatedAt: v.UpdatedAt.UTC().Format(timeLayout)}
}

func createRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.CreateRoomRequest) (*appointmentv1.Room, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.CreateRoom(ctx, p, resource.CreateRoomCommand{DepartmentID: in.DepartmentId, CampusID: in.CampusId, Building: in.Building, FloorNumber: in.FloorNumber, RoomNumber: in.RoomNumber, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return roomResponse(v), nil
}
func getRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.GetRoomRequest) (*appointmentv1.Room, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.GetRoom(ctx, p, in.RoomId)
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return roomResponse(v), nil
}
func listRooms(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListRoomsRequest) (*appointmentv1.ListRoomsResponse, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.ListRooms(ctx, p, resource.ListRoomsQuery{DepartmentID: in.DepartmentId, Page: in.Page, PageSize: in.PageSize})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	out := &appointmentv1.ListRoomsResponse{Page: v.Page, PageSize: v.PageSize, Total: v.Total}
	for _, item := range v.Items {
		out.Rooms = append(out.Rooms, roomResponse(item))
	}
	return out, nil
}
func updateRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.UpdateRoomRequest) (*appointmentv1.Room, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.UpdateRoom(ctx, p, resource.UpdateRoomCommand{RoomID: in.RoomId, CampusID: in.CampusId, Building: in.Building, FloorNumber: in.FloorNumber, RoomNumber: in.RoomNumber, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return roomResponse(v), nil
}
func retireRoom(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.RetireRoomRequest) (*appointmentv1.Room, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.RetireRoom(ctx, p, resource.RetireRoomCommand{RoomID: in.RoomId, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return roomResponse(v), nil
}
func addRoomItem(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.AddRoomExaminationItemRequest) (*appointmentv1.RoomExaminationItem, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.AddRoomItem(ctx, p, resource.AddRoomItemCommand{RoomID: in.RoomId, ItemID: in.ItemId, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return relationResponse(v), nil
}
func changeRoomItem(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest, enable bool) (*appointmentv1.RoomExaminationItem, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var v resource.RoomItem
	if enable {
		v, err = svcCtx.ResourceManager.EnableRoomItem(ctx, p, statusCommand(in))
	} else {
		v, err = svcCtx.ResourceManager.DisableRoomItem(ctx, p, statusCommand(in))
	}
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return relationResponse(v), nil
}
func listRoomItems(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListRoomExaminationItemsRequest) (*appointmentv1.ListRoomExaminationItemsResponse, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.ListRoomItems(ctx, p, in.RoomId, resource.ListRelationsQuery{Status: resource.Status(in.Status), Page: in.Page, PageSize: in.PageSize})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	out := &appointmentv1.ListRoomExaminationItemsResponse{Page: v.Page, PageSize: v.PageSize, Total: v.Total}
	for _, item := range v.Items {
		out.Relations = append(out.Relations, relationResponse(item))
	}
	return out, nil
}
func listItemRooms(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListAvailableRoomsByExaminationItemRequest) (*appointmentv1.ListRoomExaminationItemsResponse, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.ListItemRooms(ctx, p, in.ItemId, in.ActiveOnly)
	if err != nil {
		return nil, resourceRPCError(err)
	}
	out := &appointmentv1.ListRoomExaminationItemsResponse{Page: 1, PageSize: int64(len(v)), Total: int64(len(v))}
	for _, item := range v {
		out.Relations = append(out.Relations, relationResponse(item))
	}
	return out, nil
}
func setRoomWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.SetRoomWeeklyWindowRequest) (*appointmentv1.RoomWeeklyWindow, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.SetRoomWindow(ctx, p, resource.SetRoomWindowCommand{WindowID: in.WindowId, RoomID: in.RoomId, Weekday: in.Weekday, Session: resource.Session(in.Session), OpenTime: in.OpenTime, CloseTime: in.CloseTime, ActiveCapacity: in.ActiveCapacity, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return roomWindowResponse(v), nil
}
func disableRoomWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.RoomWeeklyWindow, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.DisableRoomWindow(ctx, p, statusCommand(in))
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return roomWindowResponse(v), nil
}
func listRoomWindows(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListWeeklyWindowsRequest) (*appointmentv1.ListRoomWeeklyWindowsResponse, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.ListRoomWindows(ctx, p, in.ResourceId, in.ActiveOnly)
	if err != nil {
		return nil, resourceRPCError(err)
	}
	out := &appointmentv1.ListRoomWeeklyWindowsResponse{}
	for _, item := range v {
		out.Windows = append(out.Windows, roomWindowResponse(item))
	}
	return out, nil
}
func setItemWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.SetItemWeeklyWindowRequest) (*appointmentv1.ItemWeeklyWindow, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.SetItemWindow(ctx, p, resource.SetItemWindowCommand{WindowID: in.WindowId, ItemID: in.ItemId, Weekday: in.Weekday, Session: resource.Session(in.Session), StartTime: in.StartTime, BookingCutoffTime: in.BookingCutoffTime, EndTime: in.EndTime, ExpectedVersion: in.ExpectedVersion, OperationMeta: operationMeta(in.OperationId, in.RequestId)})
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return itemWindowResponse(v), nil
}
func disableItemWindow(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.ItemWeeklyWindow, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.DisableItemWindow(ctx, p, statusCommand(in))
	if err != nil {
		return nil, resourceRPCError(err)
	}
	return itemWindowResponse(v), nil
}
func listItemWindows(ctx context.Context, svcCtx *svc.ServiceContext, in *appointmentv1.ListWeeklyWindowsRequest) (*appointmentv1.ListItemWeeklyWindowsResponse, error) {
	p, err := catalogPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	v, err := svcCtx.ResourceManager.ListItemWindows(ctx, p, in.ResourceId, in.ActiveOnly)
	if err != nil {
		return nil, resourceRPCError(err)
	}
	out := &appointmentv1.ListItemWeeklyWindowsResponse{}
	for _, item := range v {
		out.Windows = append(out.Windows, itemWindowResponse(item))
	}
	return out, nil
}
