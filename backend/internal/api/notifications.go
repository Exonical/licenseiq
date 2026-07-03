package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/notify"
	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"
)

type NotificationDispatchResult struct {
	Channel string `json:"channel" example:"slack"`
	Success bool   `json:"success" example:"true"`
	Error   string `json:"error,omitempty" example:"slack returned status 500"`
}

type NotificationTestResponse struct {
	Results []NotificationDispatchResult `json:"results"`
}

type NotificationTestOutput struct{ Body NotificationTestResponse }

func registerNotificationRoutes(api huma.API, dispatcher *notify.Dispatcher, logger *zap.Logger) {
	huma.Post(api, "/notifications/test", func(ctx context.Context, _ *struct{}) (*NotificationTestOutput, error) {
		if dispatcher == nil || dispatcher.Empty() {
			return &NotificationTestOutput{Body: NotificationTestResponse{Results: nil}}, nil
		}
		message := notificationTestMessage()
		results := dispatcher.Dispatch(ctx, message)
		out := make([]NotificationDispatchResult, 0, len(results))
		for _, result := range results {
			out = append(out, NotificationDispatchResult{Channel: result.Channel, Success: result.Success, Error: result.Error})
		}
		return &NotificationTestOutput{Body: NotificationTestResponse{Results: out}}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "testNotifications"
		o.Summary = "Test notifications"
		o.Description = "Send a test notification through all configured channels."
		o.Tags = []string{"Notifications"}
		o.DefaultStatus = http.StatusOK
		o.Errors = operationErrors()
		protectedOperation("notifications", "admin")(o)
	})
	_ = logger
}

func notificationTestMessage() notify.Message {
	data := notify.RenewalReminderData{
		VendorName:  "LicenseIQ",
		ProductName: "Notification Test",
		LicenseName: "Test License",
		RenewalDate: time.Now().UTC().AddDate(0, 0, 30),
		DaysUntil:   30,
	}
	message, err := notify.RenderRenewalReminder(data)
	if err != nil {
		return notify.TestMessage()
	}
	message.Subject = "LicenseIQ notification test"
	message.Fields["role"] = "Administrator"
	return message
}
