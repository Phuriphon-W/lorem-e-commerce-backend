package dto

import "github.com/google/uuid"

// Create Checkout Session
type (
	CreateCheckoutInputDtoBody struct {
		OrderID uuid.UUID `json:"orderId" required:"true" doc:"ID of the order to pay for"`
		UserID  uuid.UUID `json:"userId" required:"true" doc:"ID of the user who make checkout"`
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
