package database

import (
	"time"

	"gorm.io/gorm"
)

// User
type User struct {
	gorm.Model
	UserName     string `gorm:"type:varchar(255);not null;unique"`
	FirstName    string `gorm:"type:varchar(255);not null"`
	LastName     string `gorm:"type:varchar(255);not null"`
	Email        string `gorm:"type:varchar(255);not null;unique"`
	PasswordHash string `gorm:"type:varchar(255);not null"`
	Telephone    string
	ZipCode      string `gorm:"type:varchar(10)"`
	Road         string `gorm:"type:varchar(255)"`
	District     string `gorm:"type:varchar(255)"`
	SubDistrict  string `gorm:"type:varchar(255)"`
	HouseNumber  string `gorm:"type:varchar(10)"`
	Province     string `gorm:"type:varchar(255)"`
	IsAdmin      bool   `gorm:"not null;default:false"`
	Cart         Cart
	Orders       []Order
	Payments     []Payment
}

// Cart
type Cart struct {
	gorm.Model
	UserID    string     `gorm:"not null"`
	CartItems []CartItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type CartItem struct {
	gorm.Model
	CartID    uint
	ProductID uint
	Product   Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Quantity  uint    `gorm:"not null"`
}

// Product
type Product struct {
	gorm.Model
	Name        string `gorm:"type:varchar(255);not null"`
	Description *string
	Price       float32 `gorm:"type:decimal(10,2);not null;check:price > 0"`
	Available   uint16  `gorm:"default:0;not null"`
	ImageURL    string  `gorm:"column:image_url;not null"`
	CategoryID  uint    `gorm:"not null"`
	Category    Category
}

type AdminProductLog struct {
	UserID    uint `gorm:"primaryKey;not null"`
	ProductID uint `gorm:"primaryKey;not null"`
	CreatedAt time.Time
	Action    string `gorm:"varchar(255);not null"`
}

// Category
type Category struct {
	gorm.Model
	Name     string    `gorm:"type:varchar(255);not null"`
	Products []Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Order
type OrderStatus string

const (
	Pending   OrderStatus = "pending"
	Paid      OrderStatus = "paid"
	Shipping  OrderStatus = "shipping"
	Completed OrderStatus = "completed"
)

type Order struct {
	gorm.Model
	UserID      uint        `gorm:"not null"`
	User        User        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	TotalPrice  float32     `gorm:"type:decimal(10,2);not null;check:total_price >= 0"`
	OrderStatus OrderStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	OrderItems  []OrderItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type OrderItem struct {
	gorm.Model
	OrderID         uint  `gorm:"primaryKey;not null"`
	Order           Order `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ProductID       uint
	Product         Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PriceAtPurchase float32 `gorm:"not null;check:price_at_purchase >= 0"`
	Quantity        uint    `gorm:"not null"`
}

// Payment
type Payment struct {
	gorm.Model
	OrderID       uint    `gorm:"not null"`
	CustomerID    uint    `gorm:"not null"`
	PaymentMethod string  `gorm:"type:varchar(255);not null"`
	PaymentAmount float64 `gorm:"type:decimal(10,2);not null;check:payment_amount >= 0"`
	PaymentStatus string  `gorm:"type:varchar(20);not null"`
}
