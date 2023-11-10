package v2

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	apiv2pb "github.com/boojack/slash/proto/gen/api/v2"
	storepb "github.com/boojack/slash/proto/gen/store"
	"github.com/boojack/slash/store"
)

func (s *APIV2Service) ListShortcuts(ctx context.Context, _ *apiv2pb.ListShortcutsRequest) (*apiv2pb.ListShortcutsResponse, error) {
	userID := ctx.Value(userIDContextKey).(int32)
	find := &store.FindShortcut{}
	find.VisibilityList = []store.Visibility{store.VisibilityWorkspace, store.VisibilityPublic}
	visibleShortcutList, err := s.Store.ListShortcuts(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch visible shortcut list, err: %v", err)
	}

	find.VisibilityList = []store.Visibility{store.VisibilityPrivate}
	find.CreatorID = &userID
	shortcutList, err := s.Store.ListShortcuts(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch private shortcut list, err: %v", err)
	}

	shortcutList = append(shortcutList, visibleShortcutList...)
	shortcuts := []*apiv2pb.Shortcut{}
	for _, shortcut := range shortcutList {
		shortcuts = append(shortcuts, convertShortcutFromStorepb(shortcut))
	}

	response := &apiv2pb.ListShortcutsResponse{
		Shortcuts: shortcuts,
	}
	return response, nil
}

func (s *APIV2Service) GetShortcut(ctx context.Context, request *apiv2pb.GetShortcutRequest) (*apiv2pb.GetShortcutResponse, error) {
	shortcut, err := s.Store.GetShortcut(ctx, &store.FindShortcut{
		Name: &request.Name,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get shortcut by name: %v", err)
	}
	if shortcut == nil {
		return nil, status.Errorf(codes.NotFound, "shortcut not found")
	}

	userID := ctx.Value(userIDContextKey).(int32)
	if shortcut.Visibility == storepb.Visibility_PRIVATE && shortcut.CreatorId != userID {
		return nil, status.Errorf(codes.PermissionDenied, "Permission denied")
	}
	shortcutMessage := convertShortcutFromStorepb(shortcut)
	response := &apiv2pb.GetShortcutResponse{
		Shortcut: shortcutMessage,
	}
	return response, nil
}

func (s *APIV2Service) CreateShortcut(ctx context.Context, request *apiv2pb.CreateShortcutRequest) (*apiv2pb.CreateShortcutResponse, error) {
	userID := ctx.Value(userIDContextKey).(int32)
	shortcut := &storepb.Shortcut{
		CreatorId:   userID,
		Name:        request.Shortcut.Name,
		Link:        request.Shortcut.Link,
		Title:       request.Shortcut.Title,
		Tags:        request.Shortcut.Tags,
		Description: request.Shortcut.Description,
		Visibility:  storepb.Visibility(request.Shortcut.Visibility),
		OgMetadata:  &storepb.OpenGraphMetadata{},
	}
	if request.Shortcut.OgMetadata != nil {
		shortcut.OgMetadata = &storepb.OpenGraphMetadata{
			Title:       request.Shortcut.OgMetadata.Title,
			Description: request.Shortcut.OgMetadata.Description,
			Image:       request.Shortcut.OgMetadata.Image,
		}
	}
	shortcut, err := s.Store.CreateShortcut(ctx, shortcut)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create shortcut, err: %v", err)
	}
	if err := s.createShortcutCreateActivity(ctx, shortcut); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create activity, err: %v", err)
	}

	response := &apiv2pb.CreateShortcutResponse{
		Shortcut: convertShortcutFromStorepb(shortcut),
	}
	return response, nil
}

func (s *APIV2Service) UpdateShortcut(ctx context.Context, request *apiv2pb.UpdateShortcutRequest) (*apiv2pb.UpdateShortcutResponse, error) {
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "updateMask is required")
	}

	userID := ctx.Value(userIDContextKey).(int32)
	currentUser, err := s.Store.GetUser(ctx, &store.FindUser{
		ID: &userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user, err: %v", err)
	}
	shortcut, err := s.Store.GetShortcut(ctx, &store.FindShortcut{
		Name: &request.Shortcut.Name,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get shortcut by name: %v", err)
	}
	if shortcut == nil {
		return nil, status.Errorf(codes.NotFound, "shortcut not found")
	}
	if shortcut.CreatorId != userID && currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "Permission denied")
	}

	update := &store.UpdateShortcut{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "link":
			update.Link = &request.Shortcut.Link
		case "title":
			update.Title = &request.Shortcut.Title
		case "tags":
			tag := strings.Join(request.Shortcut.Tags, " ")
			update.Tag = &tag
		case "description":
			update.Description = &request.Shortcut.Description
		case "visibility":
			visibility := store.Visibility(request.Shortcut.Visibility)
			update.Visibility = &visibility
		case "og_metadata":
			if request.Shortcut.OgMetadata != nil {
				update.OpenGraphMetadata = &store.OpenGraphMetadata{
					Title:       request.Shortcut.OgMetadata.Title,
					Description: request.Shortcut.OgMetadata.Description,
					Image:       request.Shortcut.OgMetadata.Image,
				}
			}
		}
	}
	shortcut, err = s.Store.UpdateShortcut(ctx, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update shortcut, err: %v", err)
	}

	response := &apiv2pb.UpdateShortcutResponse{
		Shortcut: convertShortcutFromStorepb(shortcut),
	}
	return response, nil
}

func (s *APIV2Service) DeleteShortcut(ctx context.Context, request *apiv2pb.DeleteShortcutRequest) (*apiv2pb.DeleteShortcutResponse, error) {
	userID := ctx.Value(userIDContextKey).(int32)
	currentUser, err := s.Store.GetUser(ctx, &store.FindUser{
		ID: &userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user, err: %v", err)
	}
	shortcut, err := s.Store.GetShortcut(ctx, &store.FindShortcut{
		Name: &request.Name,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get shortcut by name: %v", err)
	}
	if shortcut == nil {
		return nil, status.Errorf(codes.NotFound, "shortcut not found")
	}
	if shortcut.CreatorId != userID && currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "Permission denied")
	}

	err = s.Store.DeleteShortcut(ctx, &store.DeleteShortcut{
		ID: shortcut.Id,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete shortcut, err: %v", err)
	}
	response := &apiv2pb.DeleteShortcutResponse{}
	return response, nil
}

func (s *APIV2Service) createShortcutCreateActivity(ctx context.Context, shortcut *storepb.Shortcut) error {
	payload := &storepb.ActivityShorcutCreatePayload{
		ShortcutId: shortcut.Id,
	}
	payloadStr, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal activity payload")
	}
	activity := &store.Activity{
		CreatorID: shortcut.CreatorId,
		Type:      store.ActivityShortcutCreate,
		Level:     store.ActivityInfo,
		Payload:   string(payloadStr),
	}
	_, err = s.Store.CreateActivity(ctx, activity)
	if err != nil {
		return errors.Wrap(err, "Failed to create activity")
	}
	return nil
}

func convertShortcutFromStorepb(shortcut *storepb.Shortcut) *apiv2pb.Shortcut {
	return &apiv2pb.Shortcut{
		Id:          shortcut.Id,
		CreatorId:   shortcut.CreatorId,
		CreatedTs:   shortcut.CreatedTs,
		UpdatedTs:   shortcut.UpdatedTs,
		RowStatus:   apiv2pb.RowStatus(shortcut.RowStatus),
		Name:        shortcut.Name,
		Link:        shortcut.Link,
		Title:       shortcut.Title,
		Tags:        shortcut.Tags,
		Description: shortcut.Description,
		Visibility:  apiv2pb.Visibility(shortcut.Visibility),
		OgMetadata: &apiv2pb.OpenGraphMetadata{
			Title:       shortcut.OgMetadata.Title,
			Description: shortcut.OgMetadata.Description,
			Image:       shortcut.OgMetadata.Image,
		},
	}
}
