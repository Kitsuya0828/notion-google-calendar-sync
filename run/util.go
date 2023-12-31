package run

import (
	"github.com/Kitsuya0828/notion-google-calendar-sync/db"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/slog"
)

func getEventsIDMap(events []*db.Event) map[string]*db.Event {
	m := make(map[string]*db.Event)
	for _, event := range events {
		m[event.UUID] = event
	}
	return m
}

func updateEventField(dbEvent *db.Event, partiallyUpdatedEvent *db.Event) *db.Event {
	updatedEvent := dbEvent

	updatedEvent.Title = partiallyUpdatedEvent.Title
	if partiallyUpdatedEvent.NotionEventID != "" {
		updatedEvent.Color = partiallyUpdatedEvent.Color
	}
	updatedEvent.StartTime = partiallyUpdatedEvent.StartTime
	updatedEvent.EndTime = partiallyUpdatedEvent.EndTime
	updatedEvent.IsAllday = partiallyUpdatedEvent.IsAllday
	updatedEvent.Description = partiallyUpdatedEvent.Description

	return dbEvent
}

func getCorrectEvent(dbEvent *db.Event, notionEvent *db.Event, googleCalendarEvent *db.Event) (*db.Event, bool, bool) {
	correctEvent := dbEvent

	notionOpts := []cmp.Option{
		cmpopts.IgnoreFields(db.Event{}, "CreatedTime", "UpdatedTime", "NotionEventID", "GoogleCalendarEventID"),
	}

	isNotionUpdated := false
	diff := cmp.Diff(dbEvent, notionEvent, notionOpts...)
	if diff != "" {
		isNotionUpdated = true
		slog.Info("compare db event with notion event", diff)
	}

	googleCalendarOpts := []cmp.Option{
		cmpopts.IgnoreFields(db.Event{}, "Color", "CreatedTime", "UpdatedTime", "NotionEventID", "GoogleCalendarEventID"),
	}

	isGoogleCalendarUpdated := false
	diff = cmp.Diff(dbEvent, googleCalendarEvent, googleCalendarOpts...)
	if diff != "" {
		isGoogleCalendarUpdated = true
		slog.Info("compare db event with google calendar event", diff)
	}

	if isNotionUpdated && !isGoogleCalendarUpdated {
		correctEvent = updateEventField(dbEvent, notionEvent)
	} else if !isNotionUpdated && isGoogleCalendarUpdated {
		correctEvent = updateEventField(dbEvent, googleCalendarEvent)
	} else if isNotionUpdated && isGoogleCalendarUpdated {
		if notionEvent.UpdatedTime.After(googleCalendarEvent.UpdatedTime) {
			correctEvent = updateEventField(dbEvent, notionEvent)
		} else {
			correctEvent = updateEventField(dbEvent, googleCalendarEvent)
		}
	}

	return correctEvent, isNotionUpdated, isGoogleCalendarUpdated
}
