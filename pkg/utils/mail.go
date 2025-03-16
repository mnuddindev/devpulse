package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/mnuddindev/devpulse/pkg/logger"
	"gopkg.in/gomail.v2"
)

// EmailConfig holds SMTP and app settings—passed in from app config
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	AppURL       string
	FromEmail    string
}

// SendActivationEmail sends a professional activation email with OTP and link
func SendActivationEmail(ctx context.Context, config EmailConfig, email, username, token string, otp int64, logger *logger.Logger) error {
	activationLink := fmt.Sprintf("%s/activate?token=%s", config.AppURL, token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to BlogBlaze</title>
    <style>
        body {
            font-family: 'Arial', sans-serif;
            background-color: #f4f4f4;
            margin: 0;
            padding: 0;
            color: #333;
        }
        .container {
            max-width: 600px;
            margin: 40px auto;
            background-color: #ffffff;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background-color: #1a73e8;
            padding: 20px;
            text-align: center;
            color: #ffffff;
        }
        .header h1 {
            margin: 0;
            font-size: 24px;
        }
        .content {
            padding: 30px;
            line-height: 1.6;
        }
        .otp {
            font-size: 28px;
            font-weight: bold;
            color: #1a73e8;
            text-align: center;
            margin: 20px 0;
        }
        .button {
            display: inline-block;
            padding: 12px 24px;
            background-color: #1a73e8;
            color: #ffffff;
            text-decoration: none;
            border-radius: 5px;
            font-weight: bold;
            text-align: center;
        }
        .button:hover {
            background-color: #1557b0;
        }
        .footer {
            background-color: #f4f4f4;
            padding: 20px;
            text-align: center;
            font-size: 12px;
            color: #777;
        }
        .footer a {
            color: #1a73e8;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to BlogBlaze!</h1>
        </div>
        <div class="content">
            <p>Hello %s,</p>
            <p>Thanks for joining BlogBlaze—the ultimate platform for creators and readers. To get started, please activate your account using the OTP code below or click the activation link.</p>
            <div class="otp">%06d</div>
            <p style="text-align: center;">
                <a href="%s" class="button">Activate Your Account</a>
            </p>
            <p><strong>Instructions:</strong></p>
            <ul>
                <li>Use the OTP code above if the link doesn’t work.</li>
                <li>Enter it on the activation page at <a href="%s">%s</a>.</li>
                <li>This code expires in 24 hours for your security.</li>
            </ul>
            <p>If you didn’t sign up, please ignore this email or contact our support team.</p>
            <p>Happy blogging!</p>
            <p>The BlogBlaze Team</p>
        </div>
        <div class="footer">
            <p>&copy; %d BlogBlaze. All rights reserved.</p>
            <p><a href="%s/support">Contact Support</a> | <a href="%s/privacy">Privacy Policy</a></p>
        </div>
    </div>
</body>
</html>
`, username, otp, activationLink, activationLink, activationLink, time.Now().Year(), config.AppURL, config.AppURL)

	// Plain text fallback
	textBody := fmt.Sprintf(`
Hello %s,

Welcome to BlogBlaze! Your activation OTP is: %06d

Activate your account here: %s

Instructions:
- Use the OTP if the link doesn’t work.
- Enter it at %s/activate.
- This code expires in 24 hours.

If you didn’t sign up, ignore this email or contact support@blogblaze.com.

Happy blogging!
The BlogBlaze Team
© %d BlogBlaze
`, username, otp, activationLink, config.AppURL, time.Now().Year())

	// Setup email
	msg := gomail.NewMessage()
	msg.SetHeader("From", config.FromEmail)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "Activate Your BlogBlaze Account")
	msg.SetBody("text/plain", textBody)
	msg.AddAlternative("text/html", htmlBody)

	// Dial and send
	dialer := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)
	if err := dialer.DialAndSend(msg); err != nil {
		logger.Warn(ctx).WithFields("email", email).Logs(fmt.Sprintf("Failed to send activation email: %v, email: %s, user: %s", err, email, username))
		return WrapError(err, ErrInternalServerError.Code, "Failed to send activation email")
	}

	logger.Info(ctx).WithFields("email", email).Logs(fmt.Sprintf("Activation email sent to: %s", email))
	return nil
}
