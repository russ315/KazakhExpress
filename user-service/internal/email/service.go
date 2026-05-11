package email

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type Service interface {
	SendWelcomeEmail(email, firstName string) error
	SendReceiptEmail(email string, orderDetails map[string]interface{}) error
}

type SMTPEmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
}

func NewSMTPEmailService() *SMTPEmailService {
	return &SMTPEmailService{
		smtpHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:     getEnv("SMTP_PORT", "587"),
		smtpUsername: getEnv("SMTP_USERNAME", ""),
		smtpPassword: getEnv("SMTP_PASSWORD", ""),
		fromEmail:    getEnv("FROM_EMAIL", "noreply@kazakhexpress.kz"),
	}
}

func (s *SMTPEmailService) SendWelcomeEmail(email, firstName string) error {
	if s.smtpUsername == "" || s.smtpPassword == "" {
		log.Printf("SMTP credentials not configured, skipping welcome email to %s", email)
		return nil
	}

	subject := "Қош келдіңіз, KazakhExpress-ке! 🎉"
	body := fmt.Sprintf(`
		Құрметті %s,

		KazakhExpress-ке тіркелгеніңізге рақмет!

		Біздің платформа арқылы Қазақстанның ең жақсы өнімдерін таба аласыз.
		Тіркелу арқылы сіз мына мүмкіндіктерге ие боласыз:
		- Жергілікті өндірушілердің өнімдері
		- Қауіпсіз төлем жүйесі
		- Жылдам жеткізу қызметі

		Біздің сайтқа кіріп, сатып алуды бастаңыз!

		Құрметпен,
		KazakhExpress командасы

		---
		Welcome, %s!

		Thank you for registering on KazakhExpress!

		Through our platform, you can find the best products from Kazakhstan.
		By registering, you gain access to:
		- Products from local manufacturers
		- Secure payment system
		- Fast delivery service

		Visit our website and start shopping!

		Best regards,
		KazakhExpress Team
	`, firstName, firstName)

	return s.sendEmail(email, subject, body)
}

func (s *SMTPEmailService) SendReceiptEmail(email string, orderDetails map[string]interface{}) error {
	if s.smtpUsername == "" || s.smtpPassword == "" {
		log.Printf("SMTP credentials not configured, skipping receipt email to %s", email)
		return nil
	}

	subject := "KazakhExpress - Заказ түбіртек / Order Receipt"
	
	orderID, _ := orderDetails["order_id"].(string)
	total, _ := orderDetails["total"].(float64)
	
	body := fmt.Sprintf(`
		Құрметті клиент,

		Сіздің заказыңыз сәтті орындалды!

		Заказ нөмірі: %s
		Жалпы соммасы: %.2f ₸

		Заказ туралы толық ақпаратты жеке кабинетіңізден көре аласыз.

		Құрметпен,
		KazakhExpress командасы

		---
		Dear customer,

		Your order has been successfully processed!

		Order ID: %s
		Total amount: %.2f KZT

		You can view full order details in your personal account.

		Best regards,
		KazakhExpress Team
	`, orderID, total, orderID, total)

	return s.sendEmail(email, subject, body)
}

func (s *SMTPEmailService) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)
	
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", 
		s.fromEmail, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	
	err := smtp.SendMail(addr, auth, s.fromEmail, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	log.Printf("Email sent successfully to %s", to)
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
