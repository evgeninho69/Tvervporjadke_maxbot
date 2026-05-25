package usecase

import (
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/ports/repository"
	"context"
)

var dummyAnnouncement = entity.Announcement{}

type AnnouncementService struct {
	Repository repository.AnnouncementRepository
	MaxPageSize uint64
}

func (as AnnouncementService) Create(ctx context.Context, announcement entity.Announcement) (uint64, error) {
	id, err := as.Repository.Create(ctx, announcement)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (as AnnouncementService) GetByID(ctx context.Context, id uint64) (entity.Announcement, error) {
	announcement, err := as.Repository.GetByID(ctx, id)
	if err != nil {
		return dummyAnnouncement, err
	}

	return announcement, nil
}

func (as AnnouncementService) Delete(ctx context.Context, id uint64) error {
	if err := as.Repository.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (as AnnouncementService) GetFirstPage(ctx context.Context, pageSize uint64) ([]entity.Announcement, error) {
	if pageSize > as.MaxPageSize {
		pageSize = as.MaxPageSize
	}
	
	announcements, err := as.Repository.GetFirstPage(ctx, pageSize)
	if err != nil {
		return nil, err
	}

	return announcements, nil
}

func (as AnnouncementService) GetPage(ctx context.Context, cursor, pageSize uint64) ([]entity.Announcement, error) {
	if pageSize > as.MaxPageSize {
		pageSize = as.MaxPageSize
	}
	
	announcements, err := as.Repository.GetPage(ctx, cursor, pageSize)
	if err != nil {
		return nil, err
	}

	return announcements, nil
}

func (as AnnouncementService) Update(ctx context.Context, announcement entity.Announcement) error {
	if err := as.Repository.Update(ctx, announcement); err != nil {
		return err
	}

	return nil
}