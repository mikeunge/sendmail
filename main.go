package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"regexp"
	"time"

	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"
)

type EmailConfig struct {
	SmtpServer   string
	SmtpPort     string
	SenderEmail  string
	SenderPass   string
	Subject      string
	TemplateFile string
	CsvFile      string
}

type EmailRecipient struct {
	Email string
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./emails.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS emails (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"email" TEXT,
		"status" TEXT,
		"timestamp" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
}

func emailSent(email string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM emails WHERE email = ? AND status = 'sent'", email).Scan(&count)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
	}
	return count > 0
}

func saveEmailStatus(email, status string) {
	_, err := db.Exec("INSERT INTO emails (email, status) VALUES (?, ?)", email, status)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
	}
}

func loadRecipientsFromCSV(filePath string) ([]EmailRecipient, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var recipients []EmailRecipient
	for _, record := range records {
		recipients = append(recipients, EmailRecipient{Email: record[0]})
	}

	return recipients, nil
}

func loadEmailTemplate(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func isValidEmail(email string) bool {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}

func sendEmailWithTLS(config EmailConfig, recipient EmailRecipient, template string, tlsConfig *tls.Config) error {
	conn, err := smtp.Dial(fmt.Sprintf("%s:%s", config.SmtpServer, config.SmtpPort))
	if err != nil {
		log.Printf("dial error: %v", err)
		return fmt.Errorf("dial error: %v", err)
	}
	defer conn.Close()

	if err = conn.StartTLS(tlsConfig); err != nil {
		log.Printf("starttls error: %v", err)
		return fmt.Errorf("starttls error: %v", err)
	}

	auth := smtp.PlainAuth("", config.SenderEmail, config.SenderPass, config.SmtpServer)
	if err = conn.Auth(auth); err != nil {
		log.Printf("auth error: %v", err)
		return fmt.Errorf("auth error: %v", err)
	}

	if err = conn.Mail(config.SenderEmail); err != nil {
		log.Printf("mail error: %v", err)
		return fmt.Errorf("mail error: %v", err)
	}
	if err = conn.Rcpt(recipient.Email); err != nil {
		log.Printf("rcpt error: %v", err)
		return fmt.Errorf("rcpt error: %v", err)
	}

	writer, err := conn.Data()
	if err != nil {
		log.Printf("data error: %v", err)
		return fmt.Errorf("data error: %v", err)
	}
	defer writer.Close()

	headers := make(map[string]string)
	headers["From"] = config.SenderEmail
	headers["To"] = recipient.Email
	headers["Subject"] = config.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	message := ""
	for key, value := range headers {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	message += "\r\n" + template

	_, err = writer.Write([]byte(message))
	if err != nil {
		log.Printf("write error: %v", err)
		return fmt.Errorf("write error: %v", err)
	}

	return nil
}

func main() {
	csvFile := flag.String("csv", "recipients.csv", "Path to the CSV file with email recipients")
	templateFile := flag.String("html", "email.html", "Path to the HTML file that gets sent")
	flag.Parse()

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	initDB()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	config := EmailConfig{
		SmtpServer:   os.Getenv("SMTP_SERVER"),
		SmtpPort:     os.Getenv("SMTP_PORT"),
		SenderEmail:  os.Getenv("SENDER_EMAIL"),
		SenderPass:   os.Getenv("SENDER_PASS"),
		Subject:      os.Getenv("EMAIL_SUBJECT"),
		TemplateFile: *templateFile,
		CsvFile:      *csvFile,
	}

	recipients, err := loadRecipientsFromCSV(config.CsvFile)
	if err != nil {
		log.Printf("Error loading recipients: %v", err)
	}
	log.Println("Loaded", len(recipients), "recipients from", config.CsvFile)
	maxSeconds := len(recipients) * 60
	maxMinutes := maxSeconds / 60
	maxHours := maxMinutes / 60
	log.Printf("Sending will take maximum %d seconds (%d minutes / %d hours)", maxSeconds, maxMinutes, maxHours)

	template, err := loadEmailTemplate(config.TemplateFile)
	if err != nil {
		log.Printf("Error loading template: %v", err)
	}

	tlsConfig := &tls.Config{
		ServerName: config.SmtpServer,
		MinVersion: tls.VersionTLS12,
	}

	// Send emails with random delays
	for i, recipient := range recipients {
		delay := time.Duration(rand.Intn(40)+20) * time.Second
		log.Printf("Sending email to %s (recipient %d/%d)",
			recipient.Email, i+1, len(recipients))

		if emailSent(recipient.Email) {
			log.Printf("Email to %s was already sent. Skipping.", recipient.Email)
			continue
		}

		if !isValidEmail(recipient.Email) {
			log.Printf("Invalid email format: %s", recipient.Email)
			saveEmailStatus(recipient.Email, "invalid")
			continue
		}

		err := sendEmailWithTLS(config, recipient, template, tlsConfig)
		if err != nil {
			log.Printf("Error sending email to %s: %v", recipient.Email, err)
			saveEmailStatus(recipient.Email, "failed")
			continue
		}

		saveEmailStatus(recipient.Email, "sent")
		log.Printf("Email sent successfully. Waiting %v before next send", delay)
		time.Sleep(delay)
	}
}
