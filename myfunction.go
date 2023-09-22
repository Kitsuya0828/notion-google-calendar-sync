package mycloudeventfunction

import (
	"context"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/Kitsuya0828/notion-google-calendar-sync/internal/task"
	"github.com/cloudevents/sdk-go/v2/event"
	"golang.org/x/exp/slog"
)

func init() {
	// Register a CloudEvent function with the Functions Framework
	functions.CloudEvent("MyCloudEventFunction", myCloudEventFunction)
}

// Function myCloudEventFunction accepts and handles a CloudEvent object
func myCloudEventFunction(ctx context.Context, e event.Event) error {
	// Your code here
	// Access the CloudEvent data payload via e.Data() or e.DataAs(...)
	opt := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelWarn,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opt))
	slog.SetDefault(logger)

	err := task.Exec()
	if err != nil {
		return err
	}

	// Return nil if no error occurred
	return nil
}
