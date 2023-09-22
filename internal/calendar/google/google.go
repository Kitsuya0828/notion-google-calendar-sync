package googlecalendar

import (
	"context"
	"fmt"
	"time"

	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/domain"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/repository"
	"github.com/caarlos0/env/v9"
	"golang.org/x/exp/slog"
	"google.golang.org/api/calendar/v3"
)

type Config struct {
	CalendarID string `env:"GOOGLE_CALENDAR_ID,notEmpty"`
}

type CalendarService struct {
	service *calendar.Service
	config  Config
}

func NewService(ctx context.Context) (*CalendarService, error) {
	srv, err := calendar.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create a new service: %v", err)
	}
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %v", err)
	}
	cs := &CalendarService{
		service: srv,
		config:  cfg,
	}
	return cs, nil
}

func (cs *CalendarService) ListEvents() ([]*domain.Event, error) {
	events := []*domain.Event{}
	result, err := cs.service.Events.List(cs.config.CalendarID).TimeMin(time.Now().Format(time.RFC3339)).Do()
	if err != nil {
		return nil, fmt.Errorf("execute calendar.events.list call: %v", err)
	}

	tz, err := time.LoadLocation(result.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("load location: %v", err)
	}
	time.Local = tz

	for _, item := range result.Items {
		event := &domain.Event{
			Title:                 item.Summary,
			GoogleCalendarEventID: item.Id,
			Description:           item.Description,
		}

		createdTime, err := time.Parse(time.RFC3339, item.Created)
		if err != nil {
			return nil, fmt.Errorf("parse created time: %v", err)
		}
		event.CreatedTime = createdTime

		updatedTime, err := time.Parse(time.RFC3339, item.Updated)
		if err != nil {
			return nil, fmt.Errorf("parse updated time: %v", err)
		}
		event.UpdatedTime = updatedTime

		startTime := time.Time{}
		if item.Start.DateTime == "" {
			startTime, err = time.ParseInLocation("2006-01-02", item.Start.Date, tz)
			if err != nil {
				return nil, fmt.Errorf("parse start time: %v", err)
			}
			event.IsAllday = true
		} else {
			startTime, err = time.Parse(time.RFC3339, item.Start.DateTime)
			if err != nil {
				return nil, fmt.Errorf("parse start time: %v", err)
			}
		}
		event.StartTime = startTime

		endTime := time.Time{}
		if item.End.DateTime == "" {
			endTime, err = time.ParseInLocation("2006-01-02", item.End.Date, tz)
			if err != nil {
				return nil, fmt.Errorf("parse end time: %v", err)
			}
		} else {
			endTime, err = time.Parse(time.RFC3339, item.End.DateTime)
			if err != nil {
				return nil, fmt.Errorf("parse end time: %v", err)
			}
		}
		event.EndTime = endTime

		if item.ExtendedProperties != nil {
			uuid, ok := item.ExtendedProperties.Private["uuid"]
			if ok {
				event.UUID = uuid
			}
		}

		for k, v := range repository.ColorMap {
			if v == item.ColorId {
				event.Color = k
				break
			}
		}
		slog.Debug("parsed google calendar event", "event", event)
		events = append(events, event)
	}
	slog.Info("listed google calendar events", "num", len(events))
	return events, nil
}

func (cs *CalendarService) InsertEvent(event *domain.Event) (string, error) {
	startDateTime := &calendar.EventDateTime{
		DateTime: event.StartTime.Format(time.RFC3339),
	}
	endDateTime := &calendar.EventDateTime{
		DateTime: event.EndTime.Format(time.RFC3339),
	}
	if event.IsAllday {
		startDateTime = &calendar.EventDateTime{
			Date: event.StartTime.Format("2006-01-02"),
		}
		endDateTime = &calendar.EventDateTime{
			Date: event.EndTime.Format("2006-01-02"),
		}
	}

	e := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Start:       startDateTime,
		End:         endDateTime,
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"uuid": event.UUID,
			},
		},
		ColorId: repository.ColorMap[event.Color],
	}

	result, err := cs.service.Events.Insert(cs.config.CalendarID, e).Do()
	if err != nil {
		return "", fmt.Errorf("execute calendar.events.insert call: %v", err)
	}
	slog.Info("inserted google calendar event", "event", result)
	return result.Id, nil
}

func (cs *CalendarService) UpdateEvent(event *domain.Event) error {
	startDateTime := &calendar.EventDateTime{
		DateTime: event.StartTime.Format(time.RFC3339),
	}
	endDateTime := &calendar.EventDateTime{
		DateTime: event.EndTime.Format(time.RFC3339),
	}
	if event.IsAllday {
		startDateTime = &calendar.EventDateTime{
			Date: event.StartTime.Format("2006-01-02"),
		}
		endDateTime = &calendar.EventDateTime{
			Date: event.EndTime.Format("2006-01-02"),
		}
	}

	e := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Start:       startDateTime,
		End:         endDateTime,
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"uuid": event.UUID,
			},
		},
	}
	if event.Color != "" {
		e.ColorId = repository.ColorMap[event.Color]
	}

	result, err := cs.service.Events.Update(cs.config.CalendarID, event.GoogleCalendarEventID, e).Do()
	if err != nil {
		return fmt.Errorf("execute calendar.events.update call: %v", err)
	}
	slog.Info("updated google calendar event", "event", result)
	return nil
}

func (cs *CalendarService) DeleteEvent(event *domain.Event) error {
	err := cs.service.Events.Delete(cs.config.CalendarID, event.GoogleCalendarEventID).Do()
	if err != nil {
		return fmt.Errorf("execute calendar.events.delete call: %v", err)
	}
	slog.Info("deleted google calendar event", "id", event.GoogleCalendarEventID)
	return nil
}
