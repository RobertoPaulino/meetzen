package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type contact struct {
	Name  string
	Email string
}

type InviteRequest struct {
	Sender      contact
	Recipients  []contact
	Title       string
	DateTime    string
	MeetingLink string
	Message     string
}

func main() {
	// Load environment variables from sendgrid.env
	loadEnvFile("sendgrid.env")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/invite", createInvite)
	log.Println("Server starting on :8080")
	log.Printf("Template ID: %s", os.Getenv("SENDGRID_TEMPLATE_ID"))
	http.ListenAndServe(":8080", mux)
}

func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Warning: Could not open %s: %v", filename, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle export statements
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
		}

		// Split on first = sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove surrounding quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}

			os.Setenv(key, value)
			log.Printf("Loaded env var: %s", key)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading %s: %v", filename, err)
	}
}

func createInvite(w http.ResponseWriter, r *http.Request) {
	var invite InviteRequest
	err := json.NewDecoder(r.Body).Decode(&invite)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if invite.Sender.Name == "" || invite.Sender.Email == "" || len(invite.Recipients) == 0 || invite.DateTime == "" || invite.Title == "" || invite.MeetingLink == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	for _, recipient := range invite.Recipients {
		if recipient.Name == "" || recipient.Email == "" {
			http.Error(w, "All recipients must have name and email", http.StatusBadRequest)
			return
		}
	}

	for _, recipient := range invite.Recipients {
		err := sendEmail(invite, recipient)
		if err != nil {
			log.Printf("Failed to send email to %s: %v", recipient.Email, err)
		}
	}

	err = sendEmail(invite, invite.Sender)
	if err != nil {
		log.Printf("Failed to send email to sender %s: %v", invite.Sender.Email, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Invites sent successfully"})
}

func sendEmail(invite InviteRequest, recipient contact) error {
	templateID := os.Getenv("SENDGRID_TEMPLATE_ID")
	if templateID == "" {
		return fmt.Errorf("SENDGRID_TEMPLATE_ID not set")
	}

	log.Printf("Sending email to %s using template ID: %s", recipient.Email, templateID)

	from := mail.NewEmail("MeetZen", "invite@meetzen.me")
	to := mail.NewEmail(recipient.Name, recipient.Email)

	message := mail.NewV3Mail()
	message.SetFrom(from)
	message.SetTemplateID(templateID)

	personalization := mail.NewPersonalization()
	personalization.AddTos(to)

	googleCalLink := createGoogleCalendarLink(invite)
	outlookCalLink := createOutlookCalendarLink(invite)

	dynamicData := map[string]interface{}{
		"recipient_name":        recipient.Name,
		"sender_name":           invite.Sender.Name,
		"title":                 invite.Title,
		"datetime":              invite.DateTime,
		"meeting_link":          invite.MeetingLink,
		"google_calendar_link":  googleCalLink,
		"outlook_calendar_link": outlookCalLink,
	}

	if invite.Message != "" {
		dynamicData["message"] = invite.Message
	}

	log.Printf("Dynamic template data for %s: %+v", recipient.Email, dynamicData)

	personalization.DynamicTemplateData = dynamicData
	message.AddPersonalizations(personalization)

	log.Printf("Sending email via SendGrid to %s with template %s", recipient.Email, templateID)

	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Printf("SendGrid API error for %s: %v", recipient.Email, err)
		return err
	}

	log.Printf("SendGrid response for %s - Status: %d - Body: %s", recipient.Email, response.StatusCode, response.Body)

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		log.Printf("✅ Email successfully sent to %s using template %s", recipient.Email, templateID)
	} else {
		log.Printf("⚠️ Email sending may have failed for %s - Status: %d", recipient.Email, response.StatusCode)
	}
	return nil
}

func createGoogleCalendarLink(invite InviteRequest) string {
	baseURL := "https://calendar.google.com/calendar/render"
	params := url.Values{}
	params.Add("action", "TEMPLATE")
	params.Add("text", invite.Title)
	messageText := fmt.Sprintf("Meeting Link: %s - Message: %s", invite.MeetingLink, invite.Message)
	params.Add("details", messageText)
	params.Add("location", invite.MeetingLink)
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func createOutlookCalendarLink(invite InviteRequest) string {
	baseURL := "https://outlook.live.com/calendar/0/deeplink/compose"
	params := url.Values{}
	params.Add("subject", invite.Title)
	bodyText := fmt.Sprintf("Date & Time: %s - Meeting Link: %s - Message: %s", invite.DateTime, invite.MeetingLink, invite.Message)
	params.Add("body", bodyText)
	params.Add("location", invite.MeetingLink)
	params.Add("path", "/calendar/action/compose")
	params.Add("rru", "addevent")
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}
