package dto

import (
	"github.com/google/uuid"
)

type PaymentDto struct {
	ID            uuid.UUID `json:"id" doc:"Payment ID"`
	OrderID       uuid.UUID `json:"orderId" doc:"Order ID of this Payment"`
	PaymentMethod string    `json:"method" doc:"Payment Method"`
	PaymentAmount float64   `json:"amount" doc:"Payment Amount"`
	PaymentStatus string    `json:"status" doc:"Payment Status"`
	CreatedAt     string    `json:"createdAt" doc:"Payment creation date"`
}

// Create Checkout Session
type (
	CreateCheckoutInputDtoBody struct {
		UserID  uuid.UUID `json:"userId" required:"true" doc:"ID of the user who make checkout"`
		OrderID uuid.UUID `json:"orderId" required:"true" doc:"ID of the order to pay"`
	}

	CreateCheckoutInputDto struct {
		Body CreateCheckoutInputDtoBody
	}

	CreateCheckoutOutputDtoBody struct {
		CheckoutURL string `json:"checkoutUrl" doc:"Payment Service Checkout URL to redirect the user to"`
	}

	CreateCheckoutOutputDto struct {
		Body CreateCheckoutOutputDtoBody
	}
)

// Verify Session
type (
	VerifySessionInputDto struct {
		SessionID string `query:"sessionId" required:"true" doc:"The payment service checkout session ID"`
	}

	VerifySessionOutputDtoBody struct {
		Valid bool `json:"valid" doc:"True if the session is fully paid and legitimate"`
	}

	VerifySessionOutputDto struct {
		Body VerifySessionOutputDtoBody
	}
)

// Get Payments By User ID

type (
	GetPaymentsByUserIdInputDto struct {
		UserID     uuid.UUID `path:"userId" required:"true" doc:"User ID"`
		PageNumber int64     `query:"pageNumber" default:"1" minimum:"1" doc:"Page number"`
		PageSize   int64     `query:"pageSize" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
		Status     string    `query:"status" doc:"Status of the payment"`
		OrderBy    string    `query:"orderBy" doc:"Query Ordering"`
	}

	GetPaymentsByUserIdOutputDtoBody struct {
		Payments []PaymentDto `json:"payments"`
		Total    int64        `json:"total"`
	}

	GetPaymentsByUserIdOutputDto struct {
		Body GetPaymentsByUserIdOutputDtoBody
	}
)
