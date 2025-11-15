package services

import (
	"database/sql"
	"fmt"
	"time"

	"fmbq-server/database"
	"github.com/google/uuid"
)

// NotificationScheduler handles scheduling and sending cart/wishlist reminders
type NotificationScheduler struct {
	notificationService *NotificationService
}

// NewNotificationScheduler creates a new notification scheduler
func NewNotificationScheduler() *NotificationScheduler {
	return &NotificationScheduler{
		notificationService: NewNotificationService(),
	}
}

// ScheduleCartReminders schedules all cart reminder notifications for a user
func (ns *NotificationScheduler) ScheduleCartReminders(userID uuid.UUID) error {
	// Get all cart items with metadata
	query := `
		SELECT DISTINCT 
			ci.product_name,
			ci.product_image_url,
			ci.product_price,
			ci.added_at
		FROM cart_items ci
		JOIN carts c ON ci.cart_id = c.id
		WHERE c.user_id = $1
		AND ci.product_name IS NOT NULL
		AND ci.product_name != ''
		ORDER BY ci.added_at DESC
		LIMIT 3
	`
	
	rows, err := database.Database.Query(query, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch cart items: %w", err)
	}
	defer rows.Close()

	var items []struct {
		ProductName    string
		ProductImageURL string
		ProductPrice   float64
		AddedAt        time.Time
	}

	for rows.Next() {
		var item struct {
			ProductName     string
			ProductImageURL string
			ProductPrice    float64
			AddedAt         time.Time
		}
		err := rows.Scan(&item.ProductName, &item.ProductImageURL, &item.ProductPrice, &item.AddedAt)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		// No items in cart, cancel existing notifications
		return ns.CancelCartReminders(userID)
	}

	// Cancel existing cart reminders for this user
	if err := ns.CancelCartReminders(userID); err != nil {
		fmt.Printf("⚠️ Warning: Failed to cancel existing cart reminders: %v\n", err)
	}

	// Get the most recent item for personalized notifications
	mostRecent := items[0]
	now := time.Now()

	// Schedule 6-hour reminder
	scheduled6h := now.Add(6 * time.Hour)
	if err := ns.createScheduledNotification(userID, "cart-reminder", "6h", nil, 
		mostRecent.ProductName, mostRecent.ProductImageURL, mostRecent.ProductPrice, scheduled6h); err != nil {
		fmt.Printf("⚠️ Failed to schedule 6h cart reminder: %v\n", err)
	}

	// Schedule 24-hour reminder
	scheduled24h := now.Add(24 * time.Hour)
	if err := ns.createScheduledNotification(userID, "cart-reminder", "24h", nil,
		mostRecent.ProductName, mostRecent.ProductImageURL, mostRecent.ProductPrice, scheduled24h); err != nil {
		fmt.Printf("⚠️ Failed to schedule 24h cart reminder: %v\n", err)
	}

	// Schedule weekly reminder (7 days)
	scheduledWeekly := now.Add(7 * 24 * time.Hour)
	if err := ns.createScheduledNotification(userID, "cart-reminder", "weekly", nil,
		mostRecent.ProductName, mostRecent.ProductImageURL, mostRecent.ProductPrice, scheduledWeekly); err != nil {
		fmt.Printf("⚠️ Failed to schedule weekly cart reminder: %v\n", err)
	}

	return nil
}

// ScheduleWishlistReminders schedules all wishlist reminder notifications for a user
func (ns *NotificationScheduler) ScheduleWishlistReminders(userID uuid.UUID, productID uuid.UUID, productName, productImageURL string, productPrice float64) error {
	now := time.Now()

	// Schedule 24-hour reminder
	scheduled24h := now.Add(24 * time.Hour)
	if err := ns.createScheduledNotification(userID, "wishlist-reminder", "24h", &productID,
		productName, productImageURL, productPrice, scheduled24h); err != nil {
		fmt.Printf("⚠️ Failed to schedule 24h wishlist reminder: %v\n", err)
	}

	// Schedule 3-day reminder
	scheduled3d := now.Add(3 * 24 * time.Hour)
	if err := ns.createScheduledNotification(userID, "wishlist-reminder", "3d", &productID,
		productName, productImageURL, productPrice, scheduled3d); err != nil {
		fmt.Printf("⚠️ Failed to schedule 3d wishlist reminder: %v\n", err)
	}

	// Schedule weekly reminder (7 days)
	scheduledWeekly := now.Add(7 * 24 * time.Hour)
	if err := ns.createScheduledNotification(userID, "wishlist-reminder", "weekly", &productID,
		productName, productImageURL, productPrice, scheduledWeekly); err != nil {
		fmt.Printf("⚠️ Failed to schedule weekly wishlist reminder: %v\n", err)
	}

	return nil
}

// CancelCartReminders cancels all pending cart reminders for a user
func (ns *NotificationScheduler) CancelCartReminders(userID uuid.UUID) error {
	query := `
		UPDATE scheduled_notifications 
		SET cancelled = TRUE, updated_at = now()
		WHERE user_id = $1 
		AND type = 'cart-reminder' 
		AND sent = FALSE 
		AND cancelled = FALSE
	`
	_, err := database.Database.Exec(query, userID)
	return err
}

// CancelWishlistReminders cancels all pending wishlist reminders for a specific product
func (ns *NotificationScheduler) CancelWishlistReminders(userID uuid.UUID, productID uuid.UUID) error {
	query := `
		UPDATE scheduled_notifications 
		SET cancelled = TRUE, updated_at = now()
		WHERE user_id = $1 
		AND type = 'wishlist-reminder' 
		AND product_id = $2
		AND sent = FALSE 
		AND cancelled = FALSE
	`
	_, err := database.Database.Exec(query, userID, productID)
	return err
}

// createScheduledNotification creates a scheduled notification record
func (ns *NotificationScheduler) createScheduledNotification(
	userID uuid.UUID,
	notificationType, reminderType string,
	productID *uuid.UUID,
	productName, productImageURL string,
	productPrice float64,
	scheduledFor time.Time,
) error {
	query := `
		INSERT INTO scheduled_notifications 
		(id, user_id, type, reminder_type, product_id, product_name, product_image_url, product_price, scheduled_for, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, now(), now())
	`
	_, err := database.Database.Exec(query, userID, notificationType, reminderType, productID,
		productName, productImageURL, productPrice, scheduledFor)
	return err
}

// ProcessScheduledNotifications processes and sends due notifications
func (ns *NotificationScheduler) ProcessScheduledNotifications() error {
	now := time.Now()
	
	// Get all notifications that are due and not sent/cancelled
	query := `
		SELECT id, user_id, type, reminder_type, product_id, product_name, product_image_url, product_price
		FROM scheduled_notifications
		WHERE scheduled_for <= $1
		AND sent = FALSE
		AND cancelled = FALSE
		ORDER BY scheduled_for ASC
		LIMIT 100
	`
	
	rows, err := database.Database.Query(query, now)
	if err != nil {
		return fmt.Errorf("failed to fetch scheduled notifications: %w", err)
	}
	defer rows.Close()

	var notifications []struct {
		ID              uuid.UUID
		UserID          uuid.UUID
		Type            string
		ReminderType    string
		ProductID       sql.NullString
		ProductName     string
		ProductImageURL string
		ProductPrice    float64
	}

	for rows.Next() {
		var notif struct {
			ID              uuid.UUID
			UserID          uuid.UUID
			Type            string
			ReminderType    string
			ProductID       sql.NullString
			ProductName     string
			ProductImageURL string
			ProductPrice    float64
		}
		err := rows.Scan(&notif.ID, &notif.UserID, &notif.Type, &notif.ReminderType,
			&notif.ProductID, &notif.ProductName, &notif.ProductImageURL, &notif.ProductPrice)
		if err != nil {
			continue
		}
		notifications = append(notifications, notif)
	}

	// Process each notification
	for _, notif := range notifications {
		// Check if cart/wishlist still has items (validation)
		shouldSend := false
		if notif.Type == "cart-reminder" {
			shouldSend = ns.validateCartHasItems(notif.UserID)
		} else if notif.Type == "wishlist-reminder" {
			if notif.ProductID.Valid {
				productUUID, _ := uuid.Parse(notif.ProductID.String)
				shouldSend = ns.validateWishlistHasProduct(notif.UserID, productUUID)
			}
		}

		if !shouldSend {
			// Mark as cancelled since item no longer exists
			ns.markNotificationCancelled(notif.ID)
			continue
		}

		// Get user's push token
		var pushToken sql.NullString
		err := database.Database.QueryRow(
			"SELECT push_token FROM users WHERE id = $1",
			notif.UserID,
		).Scan(&pushToken)

		if err != nil || !pushToken.Valid || pushToken.String == "" {
			// No push token, mark as sent to avoid retrying
			ns.markNotificationSent(notif.ID)
			continue
		}

		// Generate notification message
		title, body := ns.generateNotificationMessage(notif.Type, notif.ReminderType, notif.ProductName)

		// Send notification
		data := map[string]interface{}{
			"type":       notif.Type,
			"product_id": notif.ProductID.String,
			"product_name": notif.ProductName,
		}

		err = ns.notificationService.SendPushNotification(
			pushToken.String,
			title,
			body,
			data,
		)

		if err != nil {
			fmt.Printf("❌ Failed to send scheduled notification %s: %v\n", notif.ID, err)
			// Don't mark as sent if it failed, so it can be retried
			continue
		}

		// Mark as sent
		ns.markNotificationSent(notif.ID)
		fmt.Printf("✅ Sent scheduled notification %s to user %s\n", notif.ID, notif.UserID)
	}

	return nil
}

// validateCartHasItems checks if user still has items in cart
func (ns *NotificationScheduler) validateCartHasItems(userID uuid.UUID) bool {
	var count int
	err := database.Database.QueryRow(`
		SELECT COUNT(*)
		FROM cart_items ci
		JOIN carts c ON ci.cart_id = c.id
		WHERE c.user_id = $1
	`, userID).Scan(&count)
	return err == nil && count > 0
}

// validateWishlistHasProduct checks if product is still in user's wishlist
func (ns *NotificationScheduler) validateWishlistHasProduct(userID, productID uuid.UUID) bool {
	var exists bool
	err := database.Database.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM wishlist_items WHERE user_id = $1 AND product_id = $2)
	`, userID, productID).Scan(&exists)
	return err == nil && exists
}

// markNotificationSent marks a notification as sent
func (ns *NotificationScheduler) markNotificationSent(notificationID uuid.UUID) {
	database.Database.Exec(
		"UPDATE scheduled_notifications SET sent = TRUE, updated_at = now() WHERE id = $1",
		notificationID,
	)
}

// markNotificationCancelled marks a notification as cancelled
func (ns *NotificationScheduler) markNotificationCancelled(notificationID uuid.UUID) {
	database.Database.Exec(
		"UPDATE scheduled_notifications SET cancelled = TRUE, updated_at = now() WHERE id = $1",
		notificationID,
	)
}

// generateNotificationMessage generates title and body for notification
func (ns *NotificationScheduler) generateNotificationMessage(notificationType, reminderType, productName string) (title, body string) {
	switch notificationType {
	case "cart-reminder":
		switch reminderType {
		case "6h":
			title = "Don't forget your cart! 🛒"
			body = fmt.Sprintf("You still have items waiting in your cart. Complete your order for %s!", productName)
		case "24h":
			title = "Your cart is saved 💾"
			body = fmt.Sprintf("Don't miss %s! Your cart is saved and waiting for you.", productName)
		case "weekly":
			title = "Your saved items are still available 📦"
			body = fmt.Sprintf("Your cart is waiting! Check out %s and complete your order.", productName)
		default:
			title = "Cart reminder"
			body = fmt.Sprintf("You have items in your cart, including %s", productName)
		}
	case "wishlist-reminder":
		switch reminderType {
		case "24h":
			title = "Don't forget this! ⭐"
			body = fmt.Sprintf("You added %s to your wishlist. Don't miss out!", productName)
		case "3d":
			title = "Your wishlist is waiting for you 💝"
			body = fmt.Sprintf("Discover %s again from your wishlist!", productName)
		case "weekly":
			title = "Your wishlist is full of things you might love ❤️"
			body = fmt.Sprintf("Check out your wishlist! %s is waiting for you.", productName)
		default:
			title = "Wishlist reminder"
			body = fmt.Sprintf("You saved %s to your wishlist", productName)
		}
	default:
		title = "Reminder"
		body = fmt.Sprintf("Don't forget about %s", productName)
	}
	return title, body
}

