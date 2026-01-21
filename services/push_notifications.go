package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"fmbq-server/database"
)

// SendProductAvailabilityNotification sends a push notification when a product becomes available
func SendProductAvailabilityNotification(userID string, productTitle string, productID string) error {
	// Get user's push token from database
	var pushToken sql.NullString
	err := database.Database.QueryRow(
		"SELECT push_token FROM users WHERE id = $1",
		userID,
	).Scan(&pushToken)

	if err != nil || !pushToken.Valid || pushToken.String == "" {
		return fmt.Errorf("no push token found for user %s", userID)
	}

	// Check if it's an Expo push token (starts with ExponentPushToken)
	if len(pushToken.String) < 20 {
		return fmt.Errorf("invalid push token format")
	}

	// Prepare notification payload for Expo
	notification := map[string]interface{}{
		"to":    pushToken.String,
		"sound": "default",
		"title": "Product Back in Stock! 🎉",
		"body":  fmt.Sprintf("%s is now available. Shop now!", productTitle),
		"data": map[string]interface{}{
			"type":        "product_available",
			"product_id":  productID,
			"product_title": productTitle,
			"screen":      "product",
			"params": map[string]string{
				"id": productID,
			},
		},
		"priority": "high",
		"channelId": "default",
	}

	// Send to Expo Push Notification service
	jsonData, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	resp, err := http.Post(
		"https://exp.host/--/api/v2/push/send",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("push notification service returned status %d", resp.StatusCode)
	}

	fmt.Printf("✅ Push notification sent successfully to user %s for product %s\n", userID, productID)
	return nil
}
