package task

import (
	"context"
	"fmt"

	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/repository"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/calendar/google"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/calendar/notion"
	"golang.org/x/exp/slog"
)

func Exec() error {
	ctx := context.Background()

	// List future events in Notion database
	slog.Debug("list notion events")
	notionCalendarService, err := notioncalendar.NewService()
	if err != nil {
		return fmt.Errorf("initialize notion calendar service: %v", err)
	}
	notionEvents, err := notionCalendarService.ListEvents(ctx)
	if err != nil {
		return fmt.Errorf("list notion events: %v", err)
	}
	// List future events in Google Calendar
	slog.Debug("list google calendar events")
	googleCalendarService, err := googlecalendar.NewService(ctx)
	if err != nil {
		return fmt.Errorf("initialize google calendar service: %v", err)
	}
	googleCalendarEvents, err := googleCalendarService.ListEvents()
	if err != nil {
		return fmt.Errorf("list google calendar events: %v", err)
	}

	// Initialize Firestore client
	slog.Debug("initialize firestore client")
	databaseService, err := repository.CreateService(ctx)
	if err != nil {
		return fmt.Errorf("initialize database service: %v", err)
	}
	defer databaseService.Close()

	// Check if new events have been added
	slog.Debug("check for added events")
	err = checkAdd(ctx, notionCalendarService, googleCalendarService, notionEvents, googleCalendarEvents, databaseService)
	if err != nil {
		return fmt.Errorf("check for added events: %v", err)
	}

	// List future events again
	slog.Debug("list notion events again")
	notionEvents, err = notionCalendarService.ListEvents(ctx)
	if err != nil {
		return fmt.Errorf("list notion events again: %v", err)
	}
	slog.Debug("list google calendar events again")
	googleCalendarEvents, err = googleCalendarService.ListEvents()
	if err != nil {
		return fmt.Errorf("list google calendar events again: %v", err)
	}

	// Check if events have been updated or deleted
	err = checkUpdate(ctx, notionCalendarService, googleCalendarService, notionEvents, googleCalendarEvents, databaseService)
	if err != nil {
		return fmt.Errorf("check for updated or deleted events: %v", err)
	}

	return nil
}
