package repository

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/domain"
	"github.com/caarlos0/env/v9"
	"golang.org/x/exp/slog"
	"google.golang.org/api/iterator"
)

const (
	collectionID = "events"
)

type Config struct {
	ProjectID string `env:"GOOGLE_CLOUD_PROJECT_ID,notEmpty"`
}

type DatabaseService struct {
	client *firestore.Client
}

func CreateService(ctx context.Context) (*DatabaseService, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %v", err)
	}
	c, err := firestore.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	ds := &DatabaseService{
		client: c,
	}
	return ds, nil
}

func (ds *DatabaseService) AddEvent(ctx context.Context, event *domain.Event) error {
	uuid := event.UUID

	_, err := ds.client.Collection(collectionID).Doc(uuid).Create(ctx, event)
	if err != nil {
		return fmt.Errorf("create a document: %v", err)
	}
	slog.Info("added an event to db", "uuid", event.UUID)
	return nil
}

func (ds *DatabaseService) SetEvent(ctx context.Context, event *domain.Event) error {
	_, err := ds.client.Collection(collectionID).Doc(event.UUID).Set(ctx, event)
	if err != nil {
		return fmt.Errorf("overwrite a document: %v", err)
	}
	slog.Info("set an event on db", "uuid", event.UUID)
	return nil
}

func (ds *DatabaseService) DeleteEvent(ctx context.Context, event *domain.Event) error {
	_, err := ds.client.Collection(collectionID).Doc(event.UUID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("delete a document: %v", err)
	}
	slog.Info("delete an event on db", "uuid", event.UUID)
	return nil
}

func (ds *DatabaseService) ListEvents(ctx context.Context) ([]*domain.Event, error) {
	iter := ds.client.Collection(collectionID).Documents(ctx)
	events := []*domain.Event{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate document: %v", err)
		}

		var event domain.Event
		err = doc.DataTo(&event)
		if err != nil {
			return nil, fmt.Errorf("convert from document to event type: %v", err)
		}
		events = append(events, &event)
	}
	slog.Info("listed db events", "num", len(events))
	return events, nil
}

func (ds *DatabaseService) Close() error {
	return ds.client.Close()
}
