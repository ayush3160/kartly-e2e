// Package domain holds the shapes orders-api serves and stores.
package domain

// Money is an amount in minor units with its currency.
type Money struct {
	Amount   int64  `json:"amount"` // minor units (paise, cents)
	Currency string `json:"currency"`
}

// Product is a catalog item as the storefront shows it.
type Product struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    Money  `json:"price"`
	InStock  bool   `json:"inStock"`
}

// CartItem is one line in a cart.
type CartItem struct {
	ProductID string `json:"productId"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	UnitPrice Money  `json:"unitPrice"`
}

// Cart is a shopper's cart, kept in Mongo.
type Cart struct {
	ID         string     `json:"id" bson:"_id"`
	CustomerID string     `json:"customerId,omitempty" bson:"customerId,omitempty"`
	Items      []CartItem `json:"items" bson:"items"`
	Currency   string     `json:"currency" bson:"currency"`
}

// Quote is what pricing-svc returns for a cart.
type Quote struct {
	Subtotal Money  `json:"subtotal"`
	Discount Money  `json:"discount"`
	Shipping Money  `json:"shipping"`
	Tax      Money  `json:"tax"`
	Total    Money  `json:"total"`
	Coupon   string `json:"coupon,omitempty"`
}

// OrderItem is one line of a placed order.
type OrderItem struct {
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	UnitPrice Money  `json:"unitPrice"`
}

// Order is a placed order.
type Order struct {
	ID          string      `json:"id"`
	CustomerID  string      `json:"customerId"`
	Status      string      `json:"status"`
	Items       []OrderItem `json:"items"`
	Total       Money       `json:"total"`
	Coupon      string      `json:"coupon,omitempty"`
	ShippingFee Money       `json:"shippingFee"`
	CreatedAt   string      `json:"createdAt"`
	Notes       []string    `json:"notes,omitempty"`
	// PointsEarned is the loyalty points this order earned (KART-405).
	PointsEarned int64 `json:"pointsEarned"`
}

// Account is a customer profile from the legacy accounts database.
type Account struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Tier      string    `json:"tier"`
	Addresses []Address `json:"addresses"`
}

// Address is a shipping address.
type Address struct {
	ID      int64  `json:"id"`
	Line1   string `json:"line1"`
	City    string `json:"city"`
	Pincode string `json:"pincode"`
	Default bool   `json:"default"`
}

// Return is a return request for an order.
type Return struct {
	ID      string `json:"id"`
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
	Refund  Money  `json:"refund"`
	Label   string `json:"label,omitempty"`
	Reason  string `json:"reason"`
}
