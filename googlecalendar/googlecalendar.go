package googlecalendar

import (
	"context"
	"time"

	"github.com/Kitsuya0828/notion-googlecalendar-sync/firestore"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func NewService(ctx context.Context) (*calendar.Service, error) {
	return calendar.NewService(ctx, option.WithCredentialsFile("credentials.json"))
}

func ListEvents(service *calendar.Service, calendarID string) ([]*firestore.Event, error) {
	events := []*firestore.Event{}
	result, err := service.Events.List(calendarID).TimeMin(time.Now().Format(time.RFC3339)).Do()
	if err != nil {
		return nil, err
	}

	tz, err := time.LoadLocation(result.TimeZone)
	if err != nil {
		return nil, err
	}
	time.Local = tz

	for _, item := range result.Items {
		event := &firestore.Event{
			Title:                 item.Summary,
			GoogleCalendarEventID: item.Id,
			Description:           item.Description,
		}

		createdTime, err := time.Parse(time.RFC3339, item.Created)
		if err != nil {
			return nil, err
		}
		event.CreatedTime = createdTime

		updatedTime, err := time.Parse(time.RFC3339, item.Updated)
		if err != nil {
			return nil, err
		}
		event.UpdatedTime = updatedTime

		startTime := time.Time{}
		if item.Start.DateTime == "" {
			startTime, err = time.ParseInLocation("2006-01-02", item.Start.Date, tz)
			if err != nil {
				return nil, err
			}
		} else {
			startTime, err = time.ParseInLocation(time.RFC3339, item.Start.DateTime, tz)
			if err != nil {
				return nil, err
			}
		}
		event.StartTime = startTime

		endTime := time.Time{}
		if item.End.DateTime == "" {
			endTime, err = time.Parse("2006-01-02", item.End.Date)
			if err != nil {
				return nil, err
			}
		} else {
			endTime, err = time.Parse(time.RFC3339, item.End.DateTime)
			if err != nil {
				return nil, err
			}
		}
		event.EndTime = endTime
		events = append(events, event)
	}
	return events, nil
}

func InsertEvent(service *calendar.Service, calendarID string, event *firestore.Event) (string, error) {
	e := &calendar.Event{
		Summary: event.Title,
		Description: event.Description,
		Start: &calendar.EventDateTime{
			DateTime: event.StartTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: event.EndTime.Format(time.RFC3339),
		},
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"uuid": event.UUID,
			},
		},
		ColorId: firestore.ColorMap[event.Color],
	}

	result, err := service.Events.Insert(calendarID, e).Do()
	if err != nil {
		return "", err
	}
	return result.Id, nil
}
