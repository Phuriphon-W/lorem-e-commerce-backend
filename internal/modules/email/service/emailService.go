package service

type EmailService interface {
	SendResetPasswordEmail(toEmail, userName, resetLink string) error
}
