package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// User
type User struct {
	Base
	Username     string `gorm:"type:varchar(255);not null;unique"`
	FirstName    string `gorm:"type:varchar(255);not null"`
	LastName     string `gorm:"type:varchar(255);not null"`
	Email        string `gorm:"type:varchar(255);not null;unique"`
	PasswordHash string `gorm:"type:varchar(255);not null"`
	Telephone    *string
	ZipCode      *string `gorm:"type:varchar(10)"`
	Road         *string `gorm:"type:varchar(255)"`
	District     *string `gorm:"type:varchar(255)"`
	SubDistrict  *string `gorm:"type:varchar(255)"`
	HouseNumber  *string `gorm:"type:varchar(10)"`
	Province     *string `gorm:"type:varchar(255)"`
	IsAdmin      *bool   `gorm:"not null;default:false"`
	Cart         Cart
	Orders       []Order
	Payments     []Payment
}

// Cart
type Cart struct {
	Base
	UserID    uuid.UUID  `gorm:"not null;index"`
	CartItems []CartItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type CartItem struct {
	Base
	CartID    uuid.UUID `gorm:"index"`
	ProductID uuid.UUID
	Product   Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Quantity  uint    `gorm:"not null"`
}

// Product
type Product struct {
	Base
	Name        string `gorm:"type:varchar(255);not null;unique"`
	Description string
	Price       float32   `gorm:"type:decimal(10,2);not null;check:price > 0;index"`
	Available   uint      `gorm:"default:0;not null;check:available >= 0"`
	ImageObjKey string    `gorm:"column:obj_key;not null"`
	CategoryID  uuid.UUID `gorm:"not null;index"`
	Category    Category
}

// Category
type Category struct {
	Base
	Name     string    `gorm:"type:varchar(255);not null;unique"`
	Products []Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// Order
type OrderStatus string

const (
	Pending   OrderStatus = "pending"
	Paid      OrderStatus = "paid"
	Shipping  OrderStatus = "shipping"
	Completed OrderStatus = "completed"
	Failed    OrderStatus = "failed"
)

type Order struct {
	Base
	UserID                 uuid.UUID   `gorm:"not null;index"`
	User                   User        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	TotalPrice             float32     `gorm:"type:decimal(10,2);not null;check:total_price >= 0"`
	OrderStatus            OrderStatus `gorm:"type:varchar(20);not null;default:'pending';index"`
	StripeSessionID        *string     `gorm:"type:varchar(255)"`
	StripeSessionURL       *string     `gorm:"type:text"`
	StripeSessionExpiresAt *time.Time
	OrderItems             []OrderItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type OrderItem struct {
	Base
	OrderID         uuid.UUID `gorm:"primaryKey;not null;index"`
	Order           Order     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ProductID       uuid.UUID
	Product         Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PriceAtPurchase float32 `gorm:"not null;check:price_at_purchase >= 0"`
	Quantity        uint    `gorm:"not null"`
}

// Payment
type Payment struct {
	Base
	OrderID       uuid.UUID `gorm:"not null"`
	UserID        uuid.UUID `gorm:"not null"`
	PaymentMethod string    `gorm:"type:varchar(255);not null"`
	PaymentAmount float64   `gorm:"type:decimal(10,2);not null;check:payment_amount >= 0"`
	PaymentStatus string    `gorm:"type:varchar(20);not null"`
}

// File Metadata
type File struct {
	Base
	OriginalName string `gorm:"type:varchar(255);not null"`
	Name         string `gorm:"type:varchar(255);not null"`
	Size         int64
	ContentType  string `gorm:"type:text"`
	ObjectKey    string `gorm:"type:varchar(255);not null"`
}
