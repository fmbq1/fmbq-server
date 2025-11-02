package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ExpoPushMessage represents a push notification message
type ExpoPushMessage struct {
	To    string                 `json:"to"`
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Data  map[string]interface{} `json:"data,omitempty"`
	Sound string                 `json:"sound,omitempty"`
	Badge int                    `json:"badge,omitempty"`
}

// ExpoPushResponseItem represents a single push notification response item
type ExpoPushResponseItem struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Error  string `json:"message,omitempty"`
}

// ExpoPushResponse represents the response from Expo push service
// Data can be either a single object or an array of objects
type ExpoPushResponse struct {
	Data json.RawMessage `json:"data"`
}

// NotificationService handles push notifications
type NotificationService struct {
	ExpoPushURL string
}

// NewNotificationService creates a new notification service
func NewNotificationService() *NotificationService {
	return &NotificationService{
		ExpoPushURL: "https://exp.host/--/api/v2/push/send",
	}
}

// SendPushNotification sends a push notification to a user
func (ns *NotificationService) SendPushNotification(pushToken string, title, body string, data map[string]interface{}) error {
	if pushToken == "" {
		return fmt.Errorf("push token is empty")
	}

	// Expo Push API expects messages in an array format
	message := ExpoPushMessage{
		To:    pushToken,
		Title: title,
		Body:  body,
		Data:  data,
		Sound: "default",
		Badge: 1,
	}

	// Send as array (batch format)
	messages := []ExpoPushMessage{message}
	jsonData, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal push message: %w", err)
	}

    fmt.Printf("🔔 Sending push → token=%s title=%q body=%q\n", pushToken, title, body)
    req, err := http.NewRequest("POST", ns.ExpoPushURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

    resp, err := client.Do(req)
	if err != nil {
        return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
        fmt.Printf("🔔 Expo push error status=%d body=%s\n", resp.StatusCode, string(responseBody))
        return fmt.Errorf("push notification failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var pushResponse ExpoPushResponse
	if err := json.Unmarshal(responseBody, &pushResponse); err != nil {
        fmt.Printf("🔔 Expo push parse error body=%s\n", string(responseBody))
        return fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse data field - can be either a single object or an array
	var results []ExpoPushResponseItem
	
	// Try parsing as array first
	var dataArray []ExpoPushResponseItem
	if err := json.Unmarshal(pushResponse.Data, &dataArray); err == nil {
		results = dataArray
	} else {
		// Try parsing as single object
		var singleResult ExpoPushResponseItem
		if err := json.Unmarshal(pushResponse.Data, &singleResult); err == nil {
			results = []ExpoPushResponseItem{singleResult}
		} else {
			fmt.Printf("🔔 Expo push parse error: data is neither array nor object. Raw data: %s\n", string(pushResponse.Data))
			return fmt.Errorf("failed to parse response data: %w", err)
		}
	}

	// Check if any notifications failed
	for _, result := range results {
		if result.Status == "error" {
            fmt.Printf("🔔 Expo push result error id=%s message=%s\n", result.ID, result.Error)
            return fmt.Errorf("push notification failed: %s", result.Error)
		}
		if result.Status == "ok" {
			fmt.Printf("🔔 Expo push result success id=%s\n", result.ID)
		}
	}

    fmt.Printf("✅ Push sent successfully via Expo\n")
	return nil
}

// SendOrderStatusNotification sends a notification about order status change
func (ns *NotificationService) SendOrderStatusNotification(pushToken, orderNumber, status, customerName string) error {
	var title, body string

	switch status {
	case "pending":
		title = "Order Confirmed! 🎉"
		body = fmt.Sprintf("Hi %s! Your order #%s has been confirmed and is being processed.", customerName, orderNumber)
	case "processing":
		title = "Order Processing 📦"
		body = fmt.Sprintf("Your order #%s is being prepared for shipment.", orderNumber)
	case "shipped":
		title = "Order Shipped! 🚚"
		body = fmt.Sprintf("Great news! Your order #%s has been shipped and is on its way to you.", orderNumber)
	case "delivered":
		title = "Order Delivered! ✅"
		body = fmt.Sprintf("Your order #%s has been successfully delivered. Thank you for shopping with us!", orderNumber)
	case "cancelled":
		title = "Order Cancelled ❌"
		body = fmt.Sprintf("Your order #%s has been cancelled. If you have any questions, please contact our support team.", orderNumber)
	case "refunded":
		title = "Order Refunded 💰"
		body = fmt.Sprintf("Your order #%s has been refunded. The amount will be credited to your account within 3-5 business days.", orderNumber)
	default:
		title = "Order Update 📱"
		body = fmt.Sprintf("Your order #%s status has been updated to: %s", orderNumber, status)
	}

	data := map[string]interface{}{
		"type":         "order_update",
		"order_number": orderNumber,
		"status":       status,
		"timestamp":    time.Now().Unix(),
	}

	return ns.SendPushNotification(pushToken, title, body, data)
}

// SendOrderCreatedNotification sends a notification when a new order is created
func (ns *NotificationService) SendOrderCreatedNotification(pushToken, orderNumber, customerName string, totalAmount float64) error {
	title := "Order Placed Successfully! 🛍️"
	body := fmt.Sprintf("Hi %s! Your order #%s has been placed successfully. Total: %.2f MRU", customerName, orderNumber, totalAmount)

	data := map[string]interface{}{
		"type":         "order_created",
		"order_number": orderNumber,
		"total_amount": totalAmount,
		"timestamp":    time.Now().Unix(),
	}

	return ns.SendPushNotification(pushToken, title, body, data)
}

// SendPaymentConfirmationNotification sends a notification when payment is confirmed
func (ns *NotificationService) SendPaymentConfirmationNotification(pushToken, orderNumber, customerName string, amount float64) error {
	title := "Payment Confirmed! 💳"
	body := fmt.Sprintf("Hi %s! Payment for order #%s has been confirmed. Amount: %.2f MRU", customerName, orderNumber, amount)

	data := map[string]interface{}{
		"type":         "payment_confirmed",
		"order_number": orderNumber,
		"amount":       amount,
		"timestamp":    time.Now().Unix(),
	}

	return ns.SendPushNotification(pushToken, title, body, data)
}

// SendDeliveryUpdateNotification sends a notification about delivery updates
func (ns *NotificationService) SendDeliveryUpdateNotification(pushToken, orderNumber, message string) error {
	title := "Delivery Update 🚚"
	body := fmt.Sprintf("Update for order #%s: %s", orderNumber, message)

	data := map[string]interface{}{
		"type":         "delivery_update",
		"order_number": orderNumber,
		"message":      message,
		"timestamp":    time.Now().Unix(),
	}

	return ns.SendPushNotification(pushToken, title, body, data)
}
