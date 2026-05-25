package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"sync"
	"time"

	"activeTver/internal/domain/entity"
	"activeTver/internal/domain/ports/external"
	"activeTver/internal/domain/ports/repository"
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"activeTver/internal/pkg/tokens"

	"activeTver/internal/usecase/ports"
)

type MailCode struct {
	Code     uint64
	Attempts uint8
}

type AuthService struct {
	TokensRepository repository.RefreshTokensRepository
	UserService      ucports.UserUseCase

	EmailClient      external.EmailSender
	TemplateRenderer external.TemplateRenderer
	CacheStorage     external.CacheStorage[string, *MailCode]

	TokenManager tokens.TokensManager

	RefreshTokenExpirationTime time.Duration

	MailCodeExpirationTime time.Duration
	MailCodeMaxAttempts    uint8
	MailCodeLen            int

	SupportEmail string

	Transactor repository.Transactor
}

var (
	dummyRT         = entity.RefreshToken{}
	dummyMC         = &MailCode{}
	mailCodePrepare = sync.Once{}
	mailCodeMin     uint64
	mailCodeMax     int64
)

func (as AuthService) newMailCode() (*MailCode, error) {
	mailCodePrepare.Do(func() {
		mailCodeMin = uint64(math.Pow(10., float64(as.MailCodeLen-1)))
		mailCodeMax = int64(mailCodeMin) * 9
	})

	res, err := rand.Int(rand.Reader, big.NewInt(mailCodeMax))
	if err != nil {
		return dummyMC, err
	}

	mailCode := &MailCode{
		Code:     mailCodeMin + res.Uint64(),
		Attempts: 0,
	}

	return mailCode, nil
}

func (as AuthService) registerUser(ctx context.Context, user entity.User) (uint64, error) {
	uid, err := as.UserService.Create(ctx, user)
	if err != nil {
		return uid, err
	}

	return uid, nil
}

const opAuthRequestLogin = "usecase.Auth.RequestLogin"

var authEmailSubject = "Код для авторизации"
var authEmailType = "text/html"

type EmailValues struct {
	Code           uint64
	ExpirationTime float64
	SupportEmail   string
}

func (as AuthService) RequestLogin(ctx context.Context, email string) error {
	_, err := as.UserService.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, apperror.ErrUserNotExists) {
			return err
		}

		if _, err = as.registerUser(ctx, entity.User{Email: email}); err != nil {
			return err
		}
	}

	mailCode, err := as.newMailCode()
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAuthRequestLogin,
		)

		return apperror.ErrUseCase
	}

	if err := as.CacheStorage.Add(ctx, email, mailCode, time.Now().Add(as.MailCodeExpirationTime)); err != nil {
		return err
	}

	msg, err := as.TemplateRenderer.RenderLoginEmailTemplate(ctx, EmailValues{
		Code:           mailCode.Code,
		ExpirationTime: as.MailCodeExpirationTime.Minutes(),
		SupportEmail:   as.SupportEmail,
	})
	if err != nil {
		if err := as.CacheStorage.Remove(ctx, email); err != nil {
			logs.Warn(
				err.Error(),
				common.GetCorrelationID(ctx),
				opAuthRequestLogin,
			)
		}

		return err
	}

	if err := as.EmailClient.SendMessage(ctx, []string{email}, authEmailSubject, authEmailType, msg); err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAuthRequestLogin,
		)

		if err := as.CacheStorage.Remove(ctx, email); err != nil {
			logs.Warn(
				err.Error(),
				common.GetCorrelationID(ctx),
				opAuthRequestLogin,
			)
		}

		return err
	}

	return nil
}

func (as AuthService) newPairTokens(ctx context.Context, userID uint64) (string, entity.RefreshToken, error) {
	jwt, err := as.TokenManager.NewJWT(userID)
	if err != nil {
		return "", dummyRT, err
	}

	refreshToken, err := as.TokenManager.NewToken()
	if err != nil {
		return "", dummyRT, err
	}

	hash, salt, err := as.TokenManager.GetTokenHash(refreshToken)
	if err != nil {
		return "", dummyRT, err
	}

	rt := entity.RefreshToken{
		Token:  refreshToken,
		Hash:   hash,
		Salt:   salt,
		UserID: userID,
		Expire: time.Now().Add(as.RefreshTokenExpirationTime),
	}

	tokenID, err := as.TokensRepository.Create(ctx, rt)
	if err != nil {
		return "", dummyRT, err
	}

	rt.ID = tokenID

	return jwt, rt, nil
}

const opAuthLogin = "usecase.Auth.Login"

func (as AuthService) Login(ctx context.Context, email string, mailCode uint64) (jwt string, rt entity.RefreshToken, err error) {
	cachedMailCode, err := as.CacheStorage.Get(ctx, email)
	if err != nil {
		if errors.Is(err, apperror.ErrValueWasNotCached) {
			return "", dummyRT, apperror.ErrUserNotRequestLogin
		}

		return "", dummyRT, err
	}

	// Проверяем полученный код с сохраненным в кэше
	// Если коды одинаковые, то продолжаем, иначе возвращаем ошибку
	if cachedMailCode.Code != mailCode {
		logs.Warn(
			apperror.ErrInvalidMailCode.Error(),
			common.GetCorrelationID(ctx),
			opAuthLogin,
		)

		// Увеличиваем количество попыток входа на один
		// и проверяем не превысили ли максимум
		// Если превысили, то удаляем запись из кэша
		cachedMailCode.Attempts++
		if cachedMailCode.Attempts >= as.MailCodeMaxAttempts {
			if err := as.CacheStorage.Remove(ctx, email); err != nil {
				logs.Error(
					err.Error(),
					common.GetCorrelationID(ctx),
					opAuthLogin,
				)
			}

			return "", dummyRT, apperror.ErrLoginAttemptsLimitExceeded
		}

		return "", dummyRT, apperror.ErrInvalidMailCode
	}

	if err := as.CacheStorage.Remove(ctx, email); err != nil {
		return "", dummyRT, err
	}

	// Получаем id пользователя по почте
	user, err := as.UserService.GetByEmail(ctx, email)
	if err != nil {
		return "", dummyRT, err
	}

	jwt, rt, err = as.newPairTokens(ctx, user.ID)
	if err != nil {
		logs.Error(
			err.Error(),
			common.GetCorrelationID(ctx),
			opAuthLogin,
		)

		return "", dummyRT, err
	}

	return jwt, rt, nil
}

// TODO: При попытке авторизации с устаревшим кодов сделать выход со всех устройств

const opAuthRefreshPairTokens = "usecase.Auth.RefreshPairTokens"

func (as AuthService) RefreshPairTokens(ctx context.Context, oldRT entity.RefreshToken) (jwt string, rt entity.RefreshToken, err error) {
	var (
		newJWT string
		newRT  entity.RefreshToken
	)
	if err := as.Transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		oldRTFromDB, err := as.TokensRepository.GetDataByTokenID(ctx, oldRT.ID)
		if err != nil {
			return err
		}

		if as.TokenManager.TokenExpired(oldRTFromDB.Expire) {
			logs.Warn(
				apperror.ErrRefreshTokenExpired.Error(),
				common.GetCorrelationID(ctx),
				opAuthRefreshPairTokens,
			)

			if err := as.TokensRepository.DeleteByTokenID(ctx, oldRT.ID); err != nil {
				return err
			}

			return apperror.ErrRefreshTokenExpired
		}

		// ? Нужен alert
		if oldRTFromDB.Revoked {
			logs.Error(
				apperror.ErrRefreshTokenRevoked.Error(),
				common.GetCorrelationID(ctx),
				opAuthRefreshPairTokens,
			)

			if err := as.LogoutAllUserDevices(ctx, oldRTFromDB.UserID); err != nil {
				return err
			}

			return apperror.ErrRefreshTokenRevoked
		}

		if err := as.TokenManager.CompareHashAndToken(oldRT.Token, oldRTFromDB.Hash, oldRTFromDB.Salt); err != nil {
			logs.Warn(
				err.Error(),
				common.GetCorrelationID(ctx),
				opAuthRefreshPairTokens,
			)

			return apperror.ErrInvalidRefreshToken
		}

		newJWT, newRT, err = as.newPairTokens(ctx, oldRTFromDB.UserID)
		if err != nil {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(ctx),
				opAuthRefreshPairTokens,
			)

			return apperror.ErrUseCase
		}

		if err := as.TokensRepository.Revoke(ctx, oldRTFromDB.ID); err != nil {
			if err := as.TokensRepository.DeleteByTokenID(ctx, newRT.ID); err != nil {
				return err
			}

			return err
		}

		return nil
	}); err != nil {
		return "", dummyRT, err
	}

	return newJWT, newRT, nil
}

const opAuthLogout = "usecase.Auth.Logout"

func (as AuthService) Logout(ctx context.Context, rt entity.RefreshToken) error {
	if err := as.Transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		tokenData, err := as.TokensRepository.GetDataByTokenID(ctx, rt.ID)
		if err != nil {
			return err
		}

		if err := as.TokenManager.CompareHashAndToken(rt.Token, tokenData.Hash, tokenData.Salt); err != nil {
			logs.Error(
				err.Error(),
				common.GetCorrelationID(ctx),
				opAuthLogout,
			)

			return apperror.ErrInvalidRefreshToken
		}

		if err := as.TokensRepository.Revoke(ctx, rt.ID); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (as AuthService) LogoutAllUserDevices(ctx context.Context, userID uint64) error {
	if err := as.TokensRepository.DeleteAllByUserID(ctx, userID); err != nil {
		return err
	}

	return nil
}
