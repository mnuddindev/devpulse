package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

func StructToMap(i interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(i)

	// Dereference pointer if necessary.
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		result[field.Name] = v.Field(i).Interface()
	}
	return result
}

// SendActivationEmail sends an account activation email to the specified address.
func SendActivationEmail(code int64, email, username string) error {
	// In production these values should be configurable.
	activationURL := "http://localhost:3010/activate-user"
	smtpHost := "0.0.0.0"
	smtpPort := 1025
	smtpUsername := ""
	smtpPassword := ""

	// Construct a clean HTML email body.
	emailBody := fmt.Sprintf(`<!DOCTYPE html><html><body><p>Hello %s,</p><p>Your activation code is <strong>%d</strong>.</p><p>Please activate your account by clicking <a href="%s">here</a>.</p></body></html>`, username, code, activationURL)

	mail := gomail.NewMessage()
	mail.SetHeader("From", "support@devpulse.com")
	mail.SetHeader("To", email)
	mail.SetHeader("Subject", "Activate Your Account")
	mail.SetBody("text/html", emailBody)

	dialer := gomail.NewDialer(smtpHost, smtpPort, smtpUsername, smtpPassword)
	if err := dialer.DialAndSend(mail); err != nil {
		logger.Log.WithError(err).WithFields(logrus.Fields{
			"email": email,
			"user":  username,
		}).Error("failed to send activation email")
		return fmt.Errorf("failed to send activation email: %w", err)
	}

	logrus.WithField("email", email).Info("activation email sent successfully")
	return nil
}

// UUIDsFromStrings converts a slice of strings to a slice of UUIDs.
func UUIDsFromStrings(strUUIDs []string) []uuid.UUID {
	uuids := make([]uuid.UUID, len(strUUIDs))
	for i, strUUID := range strUUIDs {
		parsedUUID, err := uuid.Parse(strUUID)
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}
		uuids[i] = parsedUUID
	}
	return uuids
}

// StrictBodyParser parses the request body strictly and returns an error if the body contains unknown fields.
func StrictBodyParser(c *fiber.Ctx, out interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields() // Reject unknown fields
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

// Contains checks if a string exists in a slice of strings.
func Contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// TruncateString truncates a string to the given length
func CreateExcerpt(s string, length int) string {
	if len(s) > length {
		return s[:length] + "..."
	}
	return s
}

// truncateString ensures the string fits within the specified character limit
func TruncateString(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	// Truncate to maxLen, ensuring we don't cut mid-word
	words := strings.Split(s, " ")
	result := ""
	for _, word := range words {
		if utf8.RuneCountInString(result+" "+word) <= maxLen {
			result += " " + word
		} else {
			break
		}
	}
	return strings.TrimSpace(result)
}
