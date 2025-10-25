package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminOrderSummary struct {
    ID            string    `json:"id"`
    OrderNumber   string    `json:"order_number"`
    Status        string    `json:"status"`
    TotalAmount   float64   `json:"total_amount"`
    Currency      string    `json:"currency"`
    Source        string    `json:"source"`
    PaymentMethod string    `json:"payment_method"`
    CustomerName  *string   `json:"customer_name"`
    CreatedAt     time.Time `json:"created_at"`
}

// GET /api/v1/admin/pos/orders
func AdminListPOSOrders(c *gin.Context) {
    start := c.Query("start")
    end := c.Query("end")
    q := `
        SELECT o.id, o.order_number, o.status, o.total_amount, o.currency, COALESCE(o.source,'web') as source,
               COALESCE(pm.name,'') as payment_method,
               (SELECT c.contact_name FROM customers c WHERE c.user_id = o.user_id LIMIT 1) as customer_name,
               o.created_at
        FROM orders o
        LEFT JOIN payment_methods pm ON pm.id = o.payment_method_id
        WHERE ($1::timestamptz IS NULL OR o.created_at >= $1::timestamptz)
          AND ($2::timestamptz IS NULL OR o.created_at <= $2::timestamptz)
        ORDER BY o.created_at DESC
        LIMIT 500
    `
    var rows *sql.Rows
    var err error
    if start == "" && end == "" {
        rows, err = DB.Query(q, nil, nil)
    } else if start != "" && end == "" {
        rows, err = DB.Query(q, start, nil)
    } else if start == "" && end != "" {
        rows, err = DB.Query(q, nil, end)
    } else {
        rows, err = DB.Query(q, start, end)
    }
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"}); return }
    defer rows.Close()

    var out []AdminOrderSummary
    for rows.Next() {
        var rec AdminOrderSummary
        var customerName sql.NullString
        if err := rows.Scan(&rec.ID, &rec.OrderNumber, &rec.Status, &rec.TotalAmount, &rec.Currency, &rec.Source, &rec.PaymentMethod, &customerName, &rec.CreatedAt); err == nil {
            if customerName.Valid { rec.CustomerName = &customerName.String }
            out = append(out, rec)
        }
    }
    c.JSON(http.StatusOK, gin.H{"orders": out})
}

// GET /api/v1/admin/pos/orders/:id
func AdminGetPOSOrder(c *gin.Context) {
    id := c.Param("id")
    var order struct {
        ID string `json:"id"`
        OrderNumber string `json:"order_number"`
        Status string `json:"status"`
        TotalAmount float64 `json:"total_amount"`
        Currency string `json:"currency"`
        Tendered float64 `json:"tendered"`
        ChangeDue float64 `json:"change_due"`
        PaymentMethod string `json:"payment_method"`
        CustomerName *string `json:"customer_name"`
        CreatedAt time.Time `json:"created_at"`
    }
    err := DB.QueryRow(`
        SELECT o.id, o.order_number, o.status, o.total_amount, o.currency, COALESCE(o.tendered_amount,0), COALESCE(o.change_due,0),
               COALESCE(pm.name,''),
               (SELECT c.contact_name FROM customers c WHERE c.user_id = o.user_id LIMIT 1) as customer_name,
               o.created_at
        FROM orders o
        LEFT JOIN payment_methods pm ON pm.id = o.payment_method_id
        WHERE o.id = $1`, id).Scan(&order.ID, &order.OrderNumber, &order.Status, &order.TotalAmount, &order.Currency, &order.Tendered, &order.ChangeDue, &order.PaymentMethod, &order.CustomerName, &order.CreatedAt)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"}); return }

    // items
    rows, err := DB.Query(`
        SELECT oi.id, oi.quantity, oi.unit_price, oi.total_price,
               s.id as sku_id,
               COALESCE(pm2.name,'') as product_name,
               COALESCE(pc.color,'') as color,
               COALESCE(s.size,'') as size
        FROM order_items oi
        JOIN skus s ON s.id = oi.sku_id
        LEFT JOIN product_models pm2 ON pm2.id = s.product_model_id
        LEFT JOIN product_colors pc ON pc.id = s.product_color_id
        WHERE oi.order_id = $1`, id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"}); return }
    defer rows.Close()
    type Item struct {
        ID string `json:"id"`
        SKU string `json:"sku_id"`
        Title string `json:"title"`
        Color string `json:"color"`
        Size string `json:"size"`
        Quantity int `json:"quantity"`
        UnitPrice float64 `json:"unit_price"`
        TotalPrice float64 `json:"total_price"`
    }
    var items []Item
    for rows.Next() {
        var it Item
        if err := rows.Scan(&it.ID, &it.Quantity, &it.UnitPrice, &it.TotalPrice, &it.SKU, &it.Title, &it.Color, &it.Size); err == nil {
            items = append(items, it)
        }
    }
    c.JSON(http.StatusOK, gin.H{"order": order, "items": items})
}

// GET /api/v1/admin/pos/stats
func AdminPOSStats(c *gin.Context) {
    start := c.Query("start")
    end := c.Query("end")
    q := `SELECT COALESCE(SUM(total_amount),0), COUNT(*) FROM orders WHERE ($1::timestamptz IS NULL OR created_at >= $1::timestamptz) AND ($2::timestamptz IS NULL OR created_at <= $2::timestamptz)`
    var total float64
    var count int64
    var err error
    if start == "" && end == "" { err = DB.QueryRow(q, nil, nil).Scan(&total, &count) }
    if start != "" && end == "" { err = DB.QueryRow(q, start, nil).Scan(&total, &count) }
    if start == "" && end != "" { err = DB.QueryRow(q, nil, end).Scan(&total, &count) }
    if start != "" && end != "" { err = DB.QueryRow(q, start, end).Scan(&total, &count) }
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"}); return }
    c.JSON(http.StatusOK, gin.H{"total_amount": total, "orders_count": count})
}

// GET /api/v1/admin/inventory/low-stock
// Lists SKUs where available <= reorder_point
func AdminLowStock(c *gin.Context) {
    rows, err := DB.Query(`
        SELECT s.id, s.sku_code, COALESCE(pm.name,''), COALESCE(pc.color,''), COALESCE(s.size,''), i.available, i.reorder_point
        FROM inventory i
        JOIN skus s ON s.id = i.sku_id
        LEFT JOIN product_models pm ON pm.id = s.product_model_id
        LEFT JOIN product_colors pc ON pc.id = s.product_color_id
        WHERE i.available <= i.reorder_point
        ORDER BY i.available ASC
        LIMIT 200`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch low stock"})
        return
    }
    defer rows.Close()
    type Low struct {
        SKUID string `json:"sku_id"`
        SKUCode string `json:"sku_code"`
        Title string `json:"title"`
        Color string `json:"color"`
        Size string `json:"size"`
        Available int `json:"available"`
        ReorderPoint int `json:"reorder_point"`
    }
    var out []Low
    for rows.Next() {
        var l Low
        var title string
        if err := rows.Scan(&l.SKUID, &l.SKUCode, &title, &l.Color, &l.Size, &l.Available, &l.ReorderPoint); err == nil {
            l.Title = strings.TrimSpace(title)
            out = append(out, l)
        }
    }
    c.JSON(http.StatusOK, gin.H{"items": out})
}

// PUT /api/v1/admin/inventory/:sku_id/reorder-point
func AdminSetReorderPoint(c *gin.Context) {
    skuID := c.Param("sku_id")
    var body struct{ ReorderPoint int `json:"reorder_point"` }
    if err := c.ShouldBindJSON(&body); err != nil || body.ReorderPoint < 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reorder point"})
        return
    }
    if _, err := DB.Exec(`UPDATE inventory SET reorder_point = $1, updated_at = now() WHERE sku_id = $2`, body.ReorderPoint, skuID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reorder point"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /api/v1/admin/inventory/all
// Lists all inventory with product details
func AdminAllInventory(c *gin.Context) {
    search := c.Query("search")
    var query string
    var args []interface{}
    
    if search != "" {
        query = `
            SELECT s.id, s.sku_code, COALESCE(pm.name,''), COALESCE(pc.color,''), COALESCE(s.size,''), 
                   i.available, i.reserved, i.reorder_point, i.updated_at
            FROM inventory i
            JOIN skus s ON s.id = i.sku_id
            LEFT JOIN product_models pm ON pm.id = s.product_model_id
            LEFT JOIN product_colors pc ON pc.id = s.product_color_id
            WHERE pm.name ILIKE $1 OR s.sku_code ILIKE $1 OR pc.color ILIKE $1
            ORDER BY pm.name, pc.color, s.size
            LIMIT 500`
        args = []interface{}{"%" + search + "%"}
    } else {
        query = `
            SELECT s.id, s.sku_code, COALESCE(pm.name,''), COALESCE(pc.color,''), COALESCE(s.size,''), 
                   i.available, i.reserved, i.reorder_point, i.updated_at
            FROM inventory i
            JOIN skus s ON s.id = i.sku_id
            LEFT JOIN product_models pm ON pm.id = s.product_model_id
            LEFT JOIN product_colors pc ON pc.id = s.product_color_id
            ORDER BY pm.name, pc.color, s.size
            LIMIT 500`
        args = []interface{}{}
    }
    
    rows, err := DB.Query(query, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch inventory"})
        return
    }
    defer rows.Close()
    
    type InventoryItem struct {
        SKUID string `json:"sku_id"`
        SKUCode string `json:"sku_code"`
        Title string `json:"title"`
        Color string `json:"color"`
        Size string `json:"size"`
        Available int `json:"available"`
        Reserved int `json:"reserved"`
        ReorderPoint int `json:"reorder_point"`
        UpdatedAt time.Time `json:"updated_at"`
    }
    
    var out []InventoryItem
    for rows.Next() {
        var item InventoryItem
        var title string
        if err := rows.Scan(&item.SKUID, &item.SKUCode, &title, &item.Color, &item.Size, 
                           &item.Available, &item.Reserved, &item.ReorderPoint, &item.UpdatedAt); err == nil {
            item.Title = strings.TrimSpace(title)
            out = append(out, item)
        }
    }
    c.JSON(http.StatusOK, gin.H{"items": out})
}

// PUT /api/v1/admin/inventory/:sku_id/quantity
func AdminUpdateQuantity(c *gin.Context) {
    skuID := c.Param("sku_id")
    var body struct{ 
        Available int `json:"available"`
        Reserved int `json:"reserved"`
    }
    if err := c.ShouldBindJSON(&body); err != nil || body.Available < 0 || body.Reserved < 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quantities"})
        return
    }
    if _, err := DB.Exec(`UPDATE inventory SET available = $1, reserved = $2, updated_at = now() WHERE sku_id = $3`, 
                        body.Available, body.Reserved, skuID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quantities"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"ok": true})
}


