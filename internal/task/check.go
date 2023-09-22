package task

import (
	"context"
	"fmt"

	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/calendar/google"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/calendar/notion"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/domain"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

func checkAdd(
	ctx context.Context,
	notionCalendarService *notioncalendar.CalendarService,
	googleCalendarService *googlecalendar.CalendarService,
	notionEvents []*domain.Event,
	googleCalendarEvents []*domain.Event,
	databaseService *repository.DatabaseService,
) error {
	events := append(notionEvents, googleCalendarEvents...)
	for _, event := range events {
		if event.UUID == "" { // Not yet added to the database
			uuid, err := uuid.NewRandom()
			if err != nil {
				return fmt.Errorf("randomly generate a UUID: %v", err)
			}
			event.UUID = uuid.String()

			if event.NotionEventID == "" { // Not yet added to Notion
				notionEventID, err := notionCalendarService.CreateEvent(ctx, event)
				if err != nil {
					return fmt.Errorf("create notion event for newly added google calendar event: %v", err)
				}
				event.NotionEventID = notionEventID
				err = googleCalendarService.UpdateEvent(event)
				if err != nil {
					return fmt.Errorf("update uuid for newly added google calendar event: %v", err)
				}
			} else if event.GoogleCalendarEventID == "" { // // Not yet added to Google Calendar
				googleCalendarEventID, err := googleCalendarService.InsertEvent(event)
				if err != nil {
					return fmt.Errorf("create google calendar event for newly added notion event: %v", err)
				}
				event.GoogleCalendarEventID = googleCalendarEventID
				err = notionCalendarService.UpdateEvent(ctx, event)
				if err != nil {
					return fmt.Errorf("update uuid for newly added notion event: %v", err)
				}
			}

			err = databaseService.AddEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("add a newly added event to db: %v", err)
			}
		}
	}
	return nil
}

func checkUpdate(
	ctx context.Context,
	notionCalendarService *notioncalendar.CalendarService,
	googleCalendarService *googlecalendar.CalendarService,
	notionEvents []*domain.Event,
	googleCalendarEvents []*domain.Event,
	databaseService *repository.DatabaseService,
) error {
	events, err := databaseService.ListEvents(ctx)
	if err != nil {
		return fmt.Errorf("list db events before checking update: %v", err)
	}
	for _, event := range events {
		isNotionDeleted := false
		notionEventsIDMap := getEventsIDMap(notionEvents)
		// Check if the event has been deleted on Notion
		notionEvent, ok := notionEventsIDMap[event.UUID]
		if !ok {
			isNotionDeleted = true
		}

		isGoogleCalendarDeleted := false
		googleCalendarEventsIDMap := getEventsIDMap(googleCalendarEvents)
		// Check if the event has been deleted on Google Calendar
		googelCalendarEvent, ok := googleCalendarEventsIDMap[event.UUID]
		if !ok {
			isGoogleCalendarDeleted = true
		}

		// If the event is deleted either on Notion or Google Calendar
		// TODO: Maintain consistency of events
		if isNotionDeleted && !isGoogleCalendarDeleted {
			err := googleCalendarService.DeleteEvent(event)
			if err != nil {
				return fmt.Errorf("delete google calendar event for deleted notion event: %v", err)
			}
			err = databaseService.DeleteEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("delete db event for deleted notion event: %v", err)
			}
			continue
		} else if !isNotionDeleted && isGoogleCalendarDeleted {
			err := notionCalendarService.DeleteEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("delete notion event for deleted google calendar event: %v", err)
			}
			err = databaseService.DeleteEvent(ctx, event)
			if err != nil {
				return fmt.Errorf("delete db event for deleted google calendar event: %v", err)
			}
			continue
		}

		correctEvent, isNotionUpdated, isGoogleCalendarUpdated := getCorrectEvent(event, notionEvent, googelCalendarEvent)
		slog.Debug("check update", "notion", isNotionUpdated, "google calendar", isGoogleCalendarUpdated, "uuid", event.UUID)
		if isNotionUpdated || isGoogleCalendarUpdated {
			err := databaseService.SetEvent(ctx, correctEvent)
			if err != nil {
				return fmt.Errorf("set correct event to db while checking update: %v", err)
			}
			if isNotionUpdated {
				err := googleCalendarService.UpdateEvent(correctEvent)
				if err != nil {
					return fmt.Errorf("set correct event to google calendar while checking update: %v", err)
				}
			}
			if isGoogleCalendarUpdated {
				err := notionCalendarService.UpdateEvent(ctx, correctEvent)
				if err != nil {
					return fmt.Errorf("set correct event to notion while checking update: %v", err)
				}
			}
		}
	}
	return nil
}
