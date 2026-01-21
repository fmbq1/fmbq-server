package services

import (
	"database/sql"
	"fmt"
	"time"

	"fmbq-server/database"

	"github.com/google/uuid"
)

// NotificationScheduler handles scheduling and sending cart/wishlist reminders and product suggestions
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

		// Process each notification with delays to avoid overwhelming users
		lastUserSentTime := make(map[uuid.UUID]time.Time)
		delayBetweenNotifications := 30 * time.Minute // Minimum 30 minutes between notifications to same user
		
		for _, notif := range notifications {
			// Check if we should delay this notification (avoid overwhelming user)
			if lastSentTime, exists := lastUserSentTime[notif.UserID]; exists {
				timeSinceLastNotification := time.Since(lastSentTime)
				if timeSinceLastNotification < delayBetweenNotifications {
					// Skip this notification for now, reschedule it
					newScheduledTime := lastSentTime.Add(delayBetweenNotifications)
					ns.rescheduleNotification(notif.ID, newScheduledTime)
					continue
				}
			}
			
			// Check if cart/wishlist still has items (validation)
			shouldSend := false
			if notif.Type == "cart-reminder" {
				shouldSend = ns.validateCartHasItems(notif.UserID)
			} else if notif.Type == "wishlist-reminder" {
				if notif.ProductID.Valid {
					productUUID, _ := uuid.Parse(notif.ProductID.String)
					shouldSend = ns.validateWishlistHasProduct(notif.UserID, productUUID)
				}
			} else if notif.Type == "product-suggestion" {
				// Product suggestions are always valid to send
				shouldSend = true
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
			"type":        notif.Type,
			"product_id":  func() string {
				if notif.ProductID.Valid {
					return notif.ProductID.String
				}
				return ""
			}(),
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
		lastUserSentTime[notif.UserID] = time.Now()
		fmt.Printf("✅ Sent scheduled notification %s to user %s\n", notif.ID, notif.UserID)
	}

	return nil
}

// rescheduleNotification updates the scheduled_for time for a notification
func (ns *NotificationScheduler) rescheduleNotification(notificationID uuid.UUID, newTime time.Time) error {
	query := `
		UPDATE scheduled_notifications 
		SET scheduled_for = $1, updated_at = now()
		WHERE id = $2
	`
	_, err := database.Database.Exec(query, newTime, notificationID)
	return err
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
	case "product-suggestion":
		title = "Just for you! ✨"
		body = fmt.Sprintf("We found something you might love: %s", productName)
	default:
		title = "Reminder"
		body = fmt.Sprintf("Don't forget about %s", productName)
	}
	return title, body
}

// ScheduleSuggestedProductNotifications schedules notifications for suggested products based on view history
// This runs periodically to check users with view history and schedule product suggestions
func (ns *NotificationScheduler) ScheduleSuggestedProductNotifications() error {
	// Get users who have:
	// 1. Push token enabled
	// 2. Viewed products in the last 7 days
	// 3. Don't have a pending product-suggestion notification scheduled in the next 24 hours
	query := `
		WITH users_with_views AS (
			SELECT DISTINCT 
				COALESCE(pv.user_id, 
					(SELECT id FROM users WHERE phone = pv.phone_number LIMIT 1)
				) as user_id,
				pv.phone_number
			FROM product_views pv
			WHERE pv.view_timestamp > NOW() - INTERVAL '7 days'
			AND (
				pv.user_id IS NOT NULL 
				OR (pv.phone_number IS NOT NULL AND pv.phone_number != '')
			)
		),
		users_with_token AS (
			SELECT u.id, u.push_token
			FROM users u
			INNER JOIN users_with_views uvw ON u.id = uvw.user_id
			WHERE u.push_token IS NOT NULL 
			AND u.push_token != ''
		),
		users_needing_notifications AS (
			SELECT uwt.id, uwt.push_token
			FROM users_with_token uwt
			WHERE NOT EXISTS (
				SELECT 1 FROM scheduled_notifications sn
				WHERE sn.user_id = uwt.id
				AND sn.type = 'product-suggestion'
				AND sn.scheduled_for > NOW()
				AND sn.scheduled_for <= NOW() + INTERVAL '24 hours'
				AND sn.sent = FALSE
				AND sn.cancelled = FALSE
			)
			AND NOT EXISTS (
				SELECT 1 FROM scheduled_notifications sn2
				WHERE sn2.user_id = uwt.id
				AND sn2.type = 'product-suggestion'
				AND sn2.sent = FALSE
				AND sn2.cancelled = FALSE
				AND sn2.created_at > NOW() - INTERVAL '1 day'
			)
		)
		SELECT id FROM users_needing_notifications
		LIMIT 50
	`
	
	rows, err := database.Database.Query(query)
	if err != nil {
		return fmt.Errorf("failed to fetch users needing suggestions: %w", err)
	}
	defer rows.Close()
	
	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err == nil {
			userIDs = append(userIDs, userID)
		}
	}
	
	// For each user, get their suggested products and schedule notifications
	for _, userID := range userIDs {
		if err := ns.scheduleSuggestionsForUser(userID); err != nil {
			fmt.Printf("⚠️ Failed to schedule suggestions for user %s: %v\n", userID, err)
			continue
		}
	}
	
	fmt.Printf("✅ Scheduled product suggestion notifications for %d users\n", len(userIDs))
	return nil
}

// scheduleSuggestionsForUser schedules product suggestion notifications for a specific user
func (ns *NotificationScheduler) scheduleSuggestionsForUser(userID uuid.UUID) error {
	// Get user's suggested products (similar to GetSuggestedForYou API)
	suggestedQuery := `
		WITH viewed_products AS (
			SELECT DISTINCT product_id
			FROM product_views
			WHERE user_id = $1
			AND view_timestamp > NOW() - INTERVAL '30 days'
			ORDER BY MAX(view_timestamp) DESC
			LIMIT 50
		),
		viewed_product_data AS (
			SELECT DISTINCT
				vp.product_id,
				pm.brand_id,
				array_agg(DISTINCT c.id) FILTER (WHERE c.level = 1) as level1_categories,
				array_agg(DISTINCT c.id) FILTER (WHERE c.level = 2) as level2_categories,
				array_agg(DISTINCT c.id) FILTER (WHERE c.level = 3) as level3_categories
			FROM viewed_products vp
			JOIN product_models pm ON vp.product_id = pm.id
			LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
			LEFT JOIN categories c ON pmc.category_id = c.id
			GROUP BY vp.product_id, pm.brand_id
		),
		suggested_products AS (
			SELECT DISTINCT
				pm.id,
				pm.title,
				COALESCE(pi.url, '') as image_url,
				COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) as price
			FROM product_models pm
			LEFT JOIN skus s ON pm.id = s.product_model_id
			LEFT JOIN prices pr ON s.id = pr.sku_id AND pr.currency = 'MRO'
			LEFT JOIN LATERAL (
				SELECT url 
				FROM product_images 
				WHERE product_model_id = pm.id 
				ORDER BY position, created_at
				LIMIT 1
			) pi ON true
			CROSS JOIN viewed_product_data vpd
			WHERE pm.is_active = true
			AND pm.id != ALL(SELECT product_id FROM viewed_products)
			AND pi.url IS NOT NULL AND pi.url != ''
			GROUP BY pm.id, pm.title, pi.url, vpd.level1_categories, vpd.level2_categories, vpd.level3_categories, vpd.brand_id
			HAVING (
				EXISTS (
					SELECT 1 FROM viewed_product_data vpd2
					JOIN product_model_categories pmc2 ON pmc2.product_model_id = pm.id
					JOIN categories c2 ON pmc2.category_id = c2.id
					WHERE (c2.id = ANY(vpd2.level1_categories) OR c2.id = ANY(vpd2.level2_categories) OR c2.id = ANY(vpd2.level3_categories))
				) OR EXISTS (
					SELECT 1 FROM viewed_product_data vpd3
					WHERE vpd3.brand_id = pm.brand_id AND pm.brand_id IS NOT NULL
				)
			)
			ORDER BY price ASC
			LIMIT 3
		)
		SELECT id, title, image_url, price FROM suggested_products
	`
	
	rows, err := database.Database.Query(suggestedQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch suggested products: %w", err)
	}
	defer rows.Close()
	
	type SuggestedProduct struct {
		ID       uuid.UUID
		Title    string
		ImageURL string
		Price    float64
	}
	
	var products []SuggestedProduct
	for rows.Next() {
		var p SuggestedProduct
		if err := rows.Scan(&p.ID, &p.Title, &p.ImageURL, &p.Price); err == nil {
			products = append(products, p)
		}
	}
	
	if len(products) == 0 {
		return nil // No suggestions for this user
	}
	
	// Schedule notifications with delays between them (24h, 48h, 72h) to avoid overwhelming users
	baseTime := time.Now().Add(24 * time.Hour)
	for i, product := range products {
		scheduledFor := baseTime.Add(time.Duration(i) * 24 * time.Hour)
		productID := &product.ID
		
		err := ns.createScheduledNotification(
			userID,
			"product-suggestion",
			"daily",
			productID,
			product.Title,
			product.ImageURL,
			product.Price,
			scheduledFor,
		)
		
		if err != nil {
			fmt.Printf("⚠️ Failed to schedule suggestion notification for user %s, product %s: %v\n", 
				userID, product.ID, err)
			continue
		}
	}
	
	return nil
}