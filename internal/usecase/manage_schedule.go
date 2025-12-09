package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"multitrackticketing/internal/domain"
)

type manageScheduleUseCase struct {
	eventRepo      domain.EventRepository
	sessionRepo    domain.SessionRepository
	contextTimeout time.Duration
}

func NewManageScheduleUseCase(eventRepo domain.EventRepository, sessionRepo domain.SessionRepository, timeout time.Duration) domain.ManageScheduleUseCase {
	return &manageScheduleUseCase{
		eventRepo:      eventRepo,
		sessionRepo:    sessionRepo,
		contextTimeout: timeout,
	}
}

func (uc *manageScheduleUseCase) CreateEvent(ctx context.Context, event *domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, uc.contextTimeout)
	defer cancel()

	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	return uc.eventRepo.Create(ctx, event)
}

func (uc *manageScheduleUseCase) ImportSessionizeData(ctx context.Context, eventID string, sessionizeID string) error {
	ctx, cancel := context.WithTimeout(ctx, uc.contextTimeout)
	defer cancel()

	// 1. Fetch data from Sessionize
	url := fmt.Sprintf("https://sessionize.com/api/v2/%s/view/GridSmart", sessionizeID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch from sessionize: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sessionize api returned status: %d", resp.StatusCode)
	}

	var sessionizeData SessionizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionizeData); err != nil {
		return fmt.Errorf("failed to decode sessionize response: %w", err)
	}

	// 2. Clear existing schedule
	if err := uc.sessionRepo.DeleteScheduleByEventID(ctx, eventID); err != nil {
		return fmt.Errorf("failed to delete existing schedule: %w", err)
	}

	// 3. Import new data
	// Group rooms by ID to avoid duplicates across dates (if any)
	uniqueRooms := make(map[int]string) // id -> name

	for _, grid := range sessionizeData {
		for _, room := range grid.Rooms {
			uniqueRooms[room.ID] = room.Name
		}
	}

	// Insert Rooms
	roomMap := make(map[int]string) // sessionize_id -> domain_id
	for sID, name := range uniqueRooms {
		r := &domain.Room{
			EventID:          eventID,
			Name:             name,
			SessionizeRoomID: sID,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		if err := uc.sessionRepo.CreateRoom(ctx, r); err != nil {
			return fmt.Errorf("failed to create room %s: %w", name, err)
		}
		roomMap[sID] = r.ID
	}

	// Insert Sessions
	for _, grid := range sessionizeData {
		for _, room := range grid.Rooms {
			domainRoomID, ok := roomMap[room.ID]
			if !ok {
				continue // Should not happen
			}

			for _, session := range room.Sessions {
				desc := ""
				if session.Description != nil {
					desc = *session.Description
				}
				s := &domain.Session{
					RoomID:              domainRoomID,
					SessionizeSessionID: session.ID,
					Title:               session.Title,
					StartTime:           session.StartsAt,
					EndTime:             session.EndsAt,
					Description:         desc,
					CreatedAt:           time.Now(),
					UpdatedAt:           time.Now(),
				}

				if err := uc.sessionRepo.CreateSession(ctx, s); err != nil {
					return fmt.Errorf("failed to create session %s: %w", session.Title, err)
				}
			}
		}
	}

	return nil
}
