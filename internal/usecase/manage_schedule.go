package usecase

import (
	"context"
	"errors"
	"multitrackticketing/internal/domain"
	"time"
)

type manageScheduleUseCase struct {
	sessionRepo    domain.SessionRepository
	contextTimeout time.Duration
}

func NewManageScheduleUseCase(repo domain.SessionRepository, timeout time.Duration) domain.ManageScheduleUseCase {
	return &manageScheduleUseCase{
		sessionRepo:    repo,
		contextTimeout: timeout,
	}
}

func (uc *manageScheduleUseCase) CreateSession(ctx context.Context, session *domain.Session) error {
	ctx, cancel := context.WithTimeout(ctx, uc.contextTimeout)
	defer cancel()

	if session.StartTime.After(session.EndTime) {
		return errors.New("start time must be before end time")
	}

	return uc.sessionRepo.Create(ctx, session)
}
