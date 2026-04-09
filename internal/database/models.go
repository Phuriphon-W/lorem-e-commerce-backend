package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatedAt time.Time
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
	UserID    uuid.UUID  `gorm:"not null"`
	CartItems []CartItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type CartItem struct {
	Base
	CartID    uuid.UUID
	ProductID uuid.UUID
	Product   Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Quantity  uint    `gorm:"not null"`
}

// Product
type Product struct {
	Base
	Name        string `gorm:"type:varchar(255);not null;unique"`
	Description string
	Price       float32   `gorm:"type:decimal(10,2);not null;check:price > 0"`
	Available   uint      `gorm:"default:0;not null"`
	ImageObjKey string    `gorm:"column:obj_key;not null"`
	CategoryID  uuid.UUID `gorm:"not null"`
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
)

type Order struct {
	Base
	UserID      uint        `gorm:"not null"`
	User        User        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	TotalPrice  float32     `gorm:"type:decimal(10,2);not null;check:total_price >= 0"`
	OrderStatus OrderStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	OrderItems  []OrderItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type OrderItem struct {
	Base
	OrderID         uint  `gorm:"primaryKey;not null"`
	Order           Order `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ProductID       uint
	Product         Product `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PriceAtPurchase float32 `gorm:"not null;check:price_at_purchase >= 0"`
	Quantity        uint    `gorm:"not null"`
}

// Payment
type Payment struct {
	Base
	OrderID       uint    `gorm:"not null"`
	UserID        uint    `gorm:"not null"`
	PaymentMethod string  `gorm:"type:varchar(255);not null"`
	PaymentAmount float64 `gorm:"type:decimal(10,2);not null;check:payment_amount >= 0"`
	PaymentStatus string  `gorm:"type:varchar(20);not null"`
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
