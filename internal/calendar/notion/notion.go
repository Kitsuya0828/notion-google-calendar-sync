package notioncalendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/domain"
	"github.com/caarlos0/env/v9"
	"github.com/dstotijn/go-notion"
	"golang.org/x/exp/slog"
)

type Config struct {
	Token                   string `env:"NOTION_TOKEN,notEmpty"`
	DefaultTimeZone         string `env:"NOTION_DEFAULT_TIMEZONE,notEmpty"`
	DatabaseID              string `env:"NOTION_DATABASE_ID,notEmpty"`
	DescriptionPropertyName string `env:"NOTION_DESCRIPTION_PROPERTY_NAME" envDefault:"Description"`
	TagsPropertyName        string `env:"NOTION_TAGS_PROPERTY_NAME" envDefault:"Tags"`
	DatePropertyName        string `env:"NOTION_DATE_PROPERTY_NAME" envDefault:"Date"`
	UUIDPropertyName        string `env:"NOTION_UUID_PROPERTY_NAME" envDefault:"UUID"`
}

type CalendarService struct {
	client *notion.Client
	config Config
}

func NewService() (*CalendarService, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %v", err)
	}
	c := notion.NewClient(cfg.Token)
	cs := &CalendarService{
		client: c,
		config: cfg,
	}
	return cs, nil
}

func (cs *CalendarService) ListEvents(ctx context.Context) ([]*domain.Event, error) {
	now := time.Now()
	req := &notion.DatabaseQuery{
		Filter: &notion.DatabaseQueryFilter{
			Property: cs.config.DatePropertyName,
			DatabaseQueryPropertyFilter: notion.DatabaseQueryPropertyFilter{
				Date: &notion.DatePropertyFilter{
					After: &now,
				},
			},
		},
	}

	events := []*domain.Event{}

	loc, err := time.LoadLocation(cs.config.DefaultTimeZone)
	if err != nil {
		return nil, fmt.Errorf("load location: %v", err)
	}
	time.Local = loc

	for {
		response, err := cs.client.QueryDatabase(ctx, cs.config.DatabaseID, req)
		if err != nil {
			return nil, fmt.Errorf("query database: %v", err)
		}
		result := response.Results

		for _, page := range result {
			event := &domain.Event{NotionEventID: page.ID}

			props, ok := page.Properties.(notion.DatabasePageProperties)
			if !ok {
				slog.Error("type assertion failed", "page.Properties", page.Properties)
				continue
			}

			for key, prop := range props {
				switch pt := prop.Type; pt {
				case "title":
					titles := []string{}
					for _, rt := range prop.Title {
						titles = append(titles, rt.Text.Content)
					}
					event.Title = strings.Join(titles, "\n")
				case "multi_select":
					for _, o := range prop.MultiSelect {
						if key == cs.config.TagsPropertyName {
							event.Color = string(o.Color)
							break
						}
					}
				case "created_time":
					event.CreatedTime = *prop.CreatedTime
				case "last_edited_time":
					event.UpdatedTime = *prop.LastEditedTime
				case "rich_text":
					descriptions := []string{}
					for _, rt := range prop.RichText {
						if key == cs.config.UUIDPropertyName {
							event.UUID = rt.Text.Content
							break
						} else if key == cs.config.DescriptionPropertyName {
							descriptions = append(descriptions, rt.Text.Content)
						}
					}
					if len(descriptions) > 0 {
						event.Description = strings.Join(descriptions, "\n")
					}
				case "date":
					event.StartTime = prop.Date.Start.Time
					if !prop.Date.Start.HasTime() { // All day
						st := event.StartTime
						event.StartTime = time.Date(st.Year(), st.Month(), st.Day(), 0, 0, 0, 0, loc)
						event.IsAllday = true
					}
					if prop.Date.End != nil {
						event.EndTime = prop.Date.End.Time
						if !prop.Date.End.HasTime() { // All day (more than 2 days)
							et := event.EndTime
							event.EndTime = time.Date(et.Year(), et.Month(), et.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
						}
					} else {
						if event.IsAllday { // All day (1 day)
							event.EndTime = event.StartTime.AddDate(0, 0, 1)
						} else {
							// If no end time is specified and it is not an all day event, set the duration to 1 hour
							event.EndTime = event.StartTime.Add(time.Hour)
						}
					}
					// if prop.Date.TimeZone != nil {
					// 	fmt.Println(prop.Date.TimeZone)
					// }
				default:
					slog.Debug("property type unsupported", "type", pt)
				}
			}
			slog.Debug("parsed notion event", "event", event)
			events = append(events, event)
		}

		if response.HasMore {
			req.StartCursor = *response.NextCursor
		} else {
			break
		}
	}
	slog.Info("listed notion events", "num", len(events))
	return events, nil
}

func (cs *CalendarService) CreateEvent(ctx context.Context, event *domain.Event) (string, error) {
	date := &notion.Date{
		Start: notion.NewDateTime(event.StartTime, !event.IsAllday),
	}

	if event.IsAllday { // All day event
		endTime := notion.NewDateTime(event.EndTime.AddDate(0, 0, -1), false)
		if date.Start != endTime { // All day event (more than 2 days)
			date.End = &endTime
		}
	} else {
		endTime := notion.NewDateTime(event.EndTime, true)
		date.End = &endTime
	}

	params := notion.CreatePageParams{
		ParentType: notion.ParentTypeDatabase,
		ParentID:   cs.config.DatabaseID,
		DatabasePageProperties: &notion.DatabasePageProperties{
			"title": notion.DatabasePageProperty{
				Title: []notion.RichText{
					{
						Text: &notion.Text{
							Content: event.Title,
						},
					},
				},
			},
			cs.config.DescriptionPropertyName: notion.DatabasePageProperty{
				RichText: []notion.RichText{
					{
						Text: &notion.Text{
							Content: event.Description,
						},
					},
				},
			},
			cs.config.DatePropertyName: notion.DatabasePageProperty{
				Date: date,
			},
			cs.config.UUIDPropertyName: notion.DatabasePageProperty{
				RichText: []notion.RichText{
					{
						Text: &notion.Text{
							Content: event.UUID,
						},
					},
				},
			},
		},
	}

	page, err := cs.client.CreatePage(ctx, params)
	if err != nil {
		return "", fmt.Errorf("call api to create a page: %v", err)
	}
	slog.Info("created notion event", "page", page)
	return page.ID, nil
}

func (cs *CalendarService) UpdateEvent(ctx context.Context, event *domain.Event) error {
	date := &notion.Date{
		Start: notion.NewDateTime(event.StartTime, !event.IsAllday),
	}

	if event.IsAllday { // All day event
		endTime := notion.NewDateTime(event.EndTime.AddDate(0, 0, -1), false)
		if date.Start != endTime { // All day event (more than 2 days)
			date.End = &endTime
		}
	} else {
		endTime := notion.NewDateTime(event.EndTime, true)
		date.End = &endTime
	}

	params := notion.UpdatePageParams{
		DatabasePageProperties: notion.DatabasePageProperties{
			"title": notion.DatabasePageProperty{
				Title: []notion.RichText{
					{
						Text: &notion.Text{
							Content: event.Title,
						},
					},
				},
			},
			cs.config.DescriptionPropertyName: notion.DatabasePageProperty{
				RichText: []notion.RichText{
					{
						Text: &notion.Text{
							Content: event.Description,
						},
					},
				},
			},
			cs.config.DatePropertyName: notion.DatabasePageProperty{
				Date: date,
			},
			cs.config.UUIDPropertyName: notion.DatabasePageProperty{
				RichText: []notion.RichText{
					{
						Text: &notion.Text{
							Content: event.UUID,
						},
					},
				},
			},
		},
	}

	result, err := cs.client.UpdatePage(ctx, event.NotionEventID, params)
	if err != nil {
		return fmt.Errorf("call api to update a page: %v", err)
	}
	slog.Info("updated notion event", "page", result)
	return nil
}

func (cs *CalendarService) DeleteEvent(ctx context.Context, event *domain.Event) error {
	archived := true
	params := notion.UpdatePageParams{
		Archived: &archived,
	}
	result, err := cs.client.UpdatePage(ctx, event.NotionEventID, params)
	if err != nil {
		return fmt.Errorf("call api to archive a page: %v", err)
	}
	slog.Info("deleted notion event", "page", result)
	return nil
}
