package usecase

import (
	"activeTver/internal/domain/consts"
	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/ports/external"
	"activeTver/internal/domain/ports/repository"
	"activeTver/internal/domain/types"
	filemanager "activeTver/internal/infrastructure/external/file_manager"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/image"
	"activeTver/internal/pkg/logs"
	ucports "activeTver/internal/usecase/ports"
	"bytes"
	"io"
	"time"

	"context"
	"fmt"
	"path/filepath"
)

// TODO: Проверить работу с файлами

var (
	dummyClaim = entity.Claim{}
)

type ClaimsService struct {
	Repository                 repository.ClaimsRepository
	UserRepository             repository.UserRepository
	FileManager                external.FileManager
	MaxPageSize                uint64
	CategoriesService          ucports.CategoriesUseCase
	MapService                 ucports.MapUseCase
	CompletedClaimsNoOlderThan time.Duration
}

const opClaimsPhotosPathToURLs = "usecase.Claims.claimPhotosPathToURLs"

func (cs ClaimsService) claimPhotosPathToURLs(ctx context.Context, claim *entity.Claim) {
	urls := make([]string, 0, len(claim.Photos))

	dir := fmt.Sprintf("claims/%v", claim.ID)

	for j := 0; j < len(claim.Photos); j++ {
		filePath := filepath.Join(dir, claim.Photos[j])

		photoURL, err := cs.FileManager.GetURLToFile(ctx, filePath)
		if err != nil {
			logs.Warn(
				err.Error(),
				common.GetCorrelationID(ctx),
				opClaimsPhotosPathToURLs,
			)
		}

		urls = append(urls, photoURL)
	}

	claim.Photos = urls
}

func (cs ClaimsService) GetByID(ctx context.Context, id uint64) (entity.Claim, error) {
	claim, err := cs.Repository.GetByID(ctx, id)
	if err != nil {
		return dummyClaim, err
	}

	cs.claimPhotosPathToURLs(ctx, &claim)

	return claim, nil
}

func (cs ClaimsService) GetByIDWithUserEmail(ctx context.Context, id uint64) (entity.Claim, string, error) {
	claim, email, err := cs.Repository.GetByIDWithUserEmail(ctx, id)
	if err != nil {
		return dummyClaim, "", err
	}

	cs.claimPhotosPathToURLs(ctx, &claim)

	return claim, email, nil
}

func (cs ClaimsService) GetFirstPage(ctx context.Context, pageSize uint64) ([]entity.Claim, error) {
	if pageSize > cs.MaxPageSize {
		pageSize = cs.MaxPageSize
	}

	claims, err := cs.Repository.GetFirstPage(ctx, pageSize)
	if err != nil {
		return nil, err
	}

	return claims, err
}

func (cs ClaimsService) GetPage(ctx context.Context, cursor, pageSize uint64) ([]entity.Claim, error) {
	if pageSize > cs.MaxPageSize {
		pageSize = cs.MaxPageSize
	}

	claims, err := cs.Repository.GetPage(ctx, cursor, pageSize)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func (cs ClaimsService) GetFirstUserPage(ctx context.Context, uid, pageSize uint64) ([]entity.Claim, error) {
	if pageSize > cs.MaxPageSize {
		pageSize = cs.MaxPageSize
	}

	claims, err := cs.Repository.GetFirstUserPage(ctx, pageSize, uid)
	if err != nil {
		return nil, err
	}

	return claims, err
}

func (cs ClaimsService) GetUserPage(ctx context.Context, cursor, uid, pageSize uint64) ([]entity.Claim, error) {
	if pageSize > cs.MaxPageSize {
		pageSize = cs.MaxPageSize
	}

	claims, err := cs.Repository.GetUserPage(ctx, cursor, pageSize, uid)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func (cs ClaimsService) deletePhotos(ctx context.Context, op string, claim entity.Claim) {
	dir := fmt.Sprintf("claims/%d", claim.ID)

	for _, filename := range claim.Photos {
		if err := cs.FileManager.Delete(ctx, filepath.Join(dir, filename)); err != nil {
			logs.Warn(
				err.Error(),
				common.GetCorrelationID(ctx),
				op,
			)
		}
	}
}

const opClaimsCreateProblem = "usecase.Claims.CreateProblem"

func (cs ClaimsService) CreateProblem(ctx context.Context, claim entity.Claim, files [][]byte) (uint64, error) {
	if !cs.CategoriesService.SubcategoryExist(ctx, claim.Category) {
		logs.Error(
			apperror.ErrUnknownClaimCategory.Error(),
			common.GetCorrelationID(ctx),
			opClaimsCreateProblem,
		)

		return 0, apperror.ErrUnknownClaimCategory
	}

	if !cs.MapService.IsPointInPolygon(claim.Longitude, claim.Latitude) {
		return 0, apperror.ErrPointIsNotInPolygon
	}

	if len(files) != 0 {
		claim.Photos = make([]string, 0, len(files))
	}

	fileCount := 0
	preparedFiles := make(map[string]io.Reader)
	for _, file := range files {
		ok, imageType := image.IsImage(file)
		if !ok {
			logs.Warn(
				apperror.ErrIsNotImage.Error(),
				common.GetCorrelationID(ctx),
				opClaimsCreateProblem,
			)

			continue // Пропускаем файл, если он не является jpeg/png
		}

		if imageType == "image/png" {
			bytes, err := image.ConvertToJpeg(file)
			if err != nil {
				logs.Warn(
					err.Error(),
					common.GetCorrelationID(ctx),
					opClaimsCreateProblem,
				)

				continue // Пропускаем файл (png), если не получилось конвертировать
			}

			file = bytes
		}

		r := bytes.NewReader(file)

		name := filemanager.NewFileName(".jpeg")
		preparedFiles[name] = r

		claim.Photos = append(claim.Photos, name)

		if fileCount >= 2 {
			break
		}
		fileCount++
	}

	id, err := cs.Repository.Create(ctx, claim)
	if err != nil {
		cs.deletePhotos(ctx, opClaimsCreateProblem, claim)

		return id, err
	}

	for name, file := range preparedFiles {
		dir := fmt.Sprintf("claims/%d", id)

		if err := cs.FileManager.Save(ctx, filepath.Join(dir, name), file); err != nil {
			cs.deletePhotos(ctx, opClaimsCreateProblem, claim)

			if err := cs.Repository.Delete(ctx, id); err != nil {
				logs.Warn(
					err.Error(),
					common.GetCorrelationID(ctx),
					opClaimsCreateProblem,
				)
			}
			return 0, err
		}
	}

	return id, nil
}

const opClaimsCreateProposal = "usecase.Claims.CreateProposal"

func (cs ClaimsService) CreateProposal(ctx context.Context, claim entity.Claim) (uint64, error) {
	claim.Category = consts.ProposalCategory

	if !cs.MapService.IsPointInPolygon(claim.Longitude, claim.Latitude) {
		logs.Warn(
			"point is not in polygon",
			common.GetCorrelationID(ctx),
			opClaimsCreateProposal,
		)

		return 0, apperror.ErrPointIsNotInPolygon
	}

	id, err := cs.Repository.Create(ctx, claim)
	if err != nil {
		return id, err
	}

	return id, nil
}

const opClaimsDelete = "usecase.Claims.Delete"

func (cs ClaimsService) Delete(ctx context.Context, id, uid uint64, isAdmin bool) error {
	claim, err := cs.Repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Заявка может быть удалена её владельцем в случае, если она находится на стадии обработки,
	// либо администратором системы
	if (claim.Status != types.ClaimStatusPending || uid != claim.CreatedBy) && !isAdmin {
		logs.Warn(
			apperror.ErrUserCannotPerformOperation.Error(),
			common.GetCorrelationID(ctx),
			opClaimsDelete,
		)

		return apperror.ErrUserCannotPerformOperation
	}

	if err := cs.Repository.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

const opClaimsUpdate = "usecase.Claims.Update"

func (cs ClaimsService) Update(ctx context.Context, updatedClaim entity.Claim) error {
	claim, err := cs.Repository.GetByID(ctx, updatedClaim.ID)
	if err != nil {
		return err
	}

	if updatedClaim.CreatedBy != claim.CreatedBy {
		logs.Warn(
			apperror.ErrUserCannotPerformOperation.Error(),
			common.GetCorrelationID(ctx),
			opClaimsUpdate,
		)

		return apperror.ErrUserCannotPerformOperation
	}

	if err := cs.Repository.Update(ctx, updatedClaim); err != nil {
		return err
	}

	return nil
}

func (cs ClaimsService) AddFeedback(ctx context.Context, id uint64, feedback string) error {
	if err := cs.Repository.AddFeedback(ctx, id, feedback); err != nil {
		return err
	}

	return nil
}

const opClaimsChangeStatus = "usecase.Claims.ChangeStatus"

func (cs ClaimsService) ChangeStatus(ctx context.Context, id uint64, status types.ClaimStatus) error {
	if status == types.ClaimStatusUnknown {
		logs.Warn(
			apperror.ErrUnknownClaimStatus.Error(),
			common.GetCorrelationID(ctx),
			opClaimsChangeStatus,
		)

		return apperror.ErrUnknownClaimStatus
	}

	if err := cs.Repository.ChangeStatus(ctx, id, status); err != nil {
		return err
	}

	return nil
}

func (cs ClaimsService) GetByArea(ctx context.Context, lat1, long1, lat2, long2 float64) ([]entity.Claim, error) {
	noOlderThan := time.Now().Add(-cs.CompletedClaimsNoOlderThan)

	claims, err := cs.Repository.GetByArea(
		ctx,
		lat1,
		long1,
		lat2,
		long2,
		types.ClaimStatusAccepted.ToString(),
		types.ClaimStatusCompleted.ToString(),
		noOlderThan,
	)
	if err != nil {
		return nil, err
	}

	return claims, nil
}
