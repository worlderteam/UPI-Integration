/*
package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

// Razorpay API URLs
const razorpayUPIURL = "https://api.razorpay.com/v1/orders"
const razorpayPayoutURL = "https://api.razorpay.com/v1/payouts"

// Load environment variables
func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
	log.Println("Environment variables loaded successfully")
}

// Function to handle UPI Collect Request (Top-Up)
func upiCollect(c echo.Context) error {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")

	amount := c.QueryParam("amount")
	userID := c.QueryParam("user_id")

	if amount == "" || userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required parameters"})
	}

	// Convert amount to paise
	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		log.Println("Invalid amount format:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid amount format, must be a whole number in INR"})
	}
	amountPaise := amountInt * 100

	log.Println("UPI Collect Request - Amount in Paise:", amountPaise)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"amount":          amountPaise,
		"currency":        "INR",
		"payment_capture": 1,
		"notes": map[string]string{
			"user_id": userID,
		},
	})

	// Create HTTP request
	req, _ := http.NewRequest("POST", razorpayUPIURL, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in UPI Collect API request:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to initiate UPI Collect request"})
	}
	defer resp.Body.Close()

	// Decode response
	var razorpayResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&razorpayResponse)

	log.Println("UPI Collect request successful:", razorpayResponse)
	return c.JSON(http.StatusOK, razorpayResponse)
}

// Function to handle UPI PayOut (Withdrawals)
func upiPayout(c echo.Context) error {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")
	accountNumber := os.Getenv("RAZORPAY_ACCOUNT_NUMBER")

	amount := c.QueryParam("amount")
	userID := c.QueryParam("user_id")
	upiID := c.QueryParam("upi_id") // User's UPI ID for withdrawal

	if amount == "" || userID == "" || upiID == "" || accountNumber == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required parameters"})
	}

	// Convert amount to paise
	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		log.Println("Invalid amount format:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid amount format, must be a whole number in INR"})
	}
	amountPaise := amountInt * 100

	log.Println("UPI PayOut Request - Amount in Paise:", amountPaise)

	// Step 1: Created a Contact in RazorpayX
	contactBody, _ := json.Marshal(map[string]interface{}{
		"name":    "User " + userID,
		"email":   "user@example.com",
		"contact": "9999999999",
		"type":    "customer",
	})

	req, _ := http.NewRequest("POST", "https://api.razorpay.com/v1/contacts", bytes.NewBuffer(contactBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in Contact Creation:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create contact"})
	}
	defer resp.Body.Close()

	// Decode Contact Response
	var contactResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&contactResponse)
	log.Println("Razorpay Contact Response:", contactResponse)

	contactID, exists := contactResponse["id"].(string)
	if !exists {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve Contact ID"})
	}

	// Step 2: Created a Fund Account for UPI
	fundAccountBody, _ := json.Marshal(map[string]interface{}{
		"contact_id":   contactID,
		"account_type": "vpa",
		"vpa": map[string]string{
			"address": upiID,
		},
	})

	req, _ = http.NewRequest("POST", "https://api.razorpay.com/v1/fund_accounts", bytes.NewBuffer(fundAccountBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	resp, err = client.Do(req)
	if err != nil {
		log.Println("Error in Fund Account Creation:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create fund account"})
	}
	defer resp.Body.Close()

	// Decode Fund Account Response
	var fundAccountResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&fundAccountResponse)

	fundAccountID, exists := fundAccountResponse["id"].(string)
	if !exists {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve Fund Account ID"})
	}

	// Step 3: Simulate or Initiate a Payout with RazorpayX
	var payoutResponse map[string]interface{}

	//Mock successful response in Test Mode
	if os.Getenv("RAZORPAY_KEY") == "rzp_test_EnxaFjzhDsvCiY" {
		payoutResponse = map[string]interface{}{
			"status":   "processed",
			"amount":   amountPaise,
			"currency": "INR",
			"mode":     "UPI",
			"purpose":  "refund",
			"id":       "pout_TEST123456",
		}
	} else {
		// Initiate a Real Payout
		payoutBody, _ := json.Marshal(map[string]interface{}{
			"account_number":       accountNumber,
			"fund_account_id":      fundAccountID,
			"amount":               amountPaise,
			"currency":             "INR",
			"mode":                 "UPI",
			"purpose":              "refund",
			"queue_if_low_balance": true,
			"notes": map[string]string{
				"user_id": userID,
			},
		})

		req, _ = http.NewRequest("POST", "https://api.razorpay.com/v1/payouts", bytes.NewBuffer(payoutBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(apiKey, apiSecret)

		resp, err = client.Do(req)
		if err != nil {
			log.Println("Error in UPI PayOut API request:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process UPI PayOut"})
		}
		defer resp.Body.Close()

		json.NewDecoder(resp.Body).Decode(&payoutResponse)
	}

	log.Println("UPI PayOut request successful:", payoutResponse)
	return c.JSON(http.StatusOK, payoutResponse)
}

func main() {
	loadEnv()
	e := echo.New()

	e.GET("/upi/collect", upiCollect)
	e.GET("/upi/payout", upiPayout)

	log.Println("Starting server on port 8080...")
	e.Logger.Fatal(e.Start(":8080"))
}

*/



package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

// Razorpay API URLs
const razorpayUPIURL = "https://api.razorpay.com/v1/orders"
const razorpayPayoutURL = "https://api.razorpay.com/v1/payouts"

// Load environment variables
func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
	log.Println("Environment variables loaded successfully")
}

// Function to generate a unique User ID
func generateUserID() string {
	return uuid.New().String()
}

// Function to create a Contact in RazorpayX
func createContact(name, email, phone string) (string, error) {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")

	contactBody, _ := json.Marshal(map[string]interface{}{
		"name":    name,
		"email":   email,
		"contact": phone,
		"type":    "customer",
	})

	req, _ := http.NewRequest("POST", "https://api.razorpay.com/v1/contacts", bytes.NewBuffer(contactBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error creating contact:", err)
		return "", err
	}
	defer resp.Body.Close()

	var contactResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&contactResponse)

	contactID, exists := contactResponse["id"].(string)
	if !exists {
		return "", err
	}

	log.Println("Contact Created Successfully:", contactID)
	return contactID, nil
}

// Function to create a Fund Account for UPI
func createFundAccount(contactID, upiID string) (string, error) {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")

	fundAccountBody, _ := json.Marshal(map[string]interface{}{
		"contact_id":  contactID,
		"account_type": "vpa",
		"vpa": map[string]string{
			"address": upiID,
		},
	})

	req, _ := http.NewRequest("POST", "https://api.razorpay.com/v1/fund_accounts", bytes.NewBuffer(fundAccountBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error creating fund account:", err)
		return "", err
	}
	defer resp.Body.Close()

	var fundAccountResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&fundAccountResponse)

	fundAccountID, exists := fundAccountResponse["id"].(string)
	if !exists {
		return "", err
	}

	log.Println("Fund Account Created Successfully:", fundAccountID)
	return fundAccountID, nil
}

// Function to handle UPI Collect (Top-Up)
func upiCollect(c echo.Context) error {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")

	amount := c.QueryParam("amount")
	userID := c.QueryParam("user_id")

	if amount == "" || userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required parameters"})
	}

	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		log.Println("Invalid amount format:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid amount format, must be a whole number in INR"})
	}
	amountPaise := amountInt * 100

	log.Println("UPI Collect Request - Amount in Paise:", amountPaise)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"amount":   amountPaise,
		"currency": "INR",
		"notes": map[string]string{
			"user_id": userID,
		},
	})

	req, _ := http.NewRequest("POST", razorpayUPIURL, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in UPI Collect API request:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to initiate UPI Collect request"})
	}
	defer resp.Body.Close()

	var razorpayResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&razorpayResponse)

	log.Println("UPI Collect request successful:", razorpayResponse)
	return c.JSON(http.StatusOK, razorpayResponse)
}


// API to generate User ID
func generateUserIDHandler(c echo.Context) error {
	userID := generateUserID()
	return c.JSON(http.StatusOK, map[string]string{"user_id": userID})
}

// API to create Contact in RazorpayX
func createContactHandler(c echo.Context) error {
	name := c.QueryParam("name")
	email := c.QueryParam("email")
	phone := c.QueryParam("phone")

	if name == "" || email == "" || phone == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required parameters"})
	}

	contactID, err := createContact(name, email, phone)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create contact"})
	}

	return c.JSON(http.StatusOK, map[string]string{"contact_id": contactID})
}

// API to create Fund Account for UPI
func createFundAccountHandler(c echo.Context) error {
	contactID := c.QueryParam("contact_id")
	upiID := c.QueryParam("upi_id")

	if contactID == "" || upiID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required parameters"})
	}

	fundAccountID, err := createFundAccount(contactID, upiID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create fund account"})
	}

	return c.JSON(http.StatusOK, map[string]string{"fund_account_id": fundAccountID})
}



// Function to handle UPI PayOut (Withdrawals)
func upiPayout(c echo.Context) error {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")
	accountNumber := os.Getenv("RAZORPAY_ACCOUNT_NUMBER")

	amount := c.QueryParam("amount")
	userID := c.QueryParam("user_id")
	upiID := c.QueryParam("upi_id")

	if amount == "" || userID == "" || upiID == "" || accountNumber == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required parameters"})
	}

	amountInt, err := strconv.Atoi(amount)
	if err != nil {
		log.Println("Invalid amount format:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid amount format, must be a whole number in INR"})
	}
	amountPaise := amountInt * 100

	log.Println("UPI PayOut Request - Amount in Paise:", amountPaise)

	contactID, err := createContact("User "+userID, "user@example.com", "9999999999")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create contact"})
	}

	fundAccountID, err := createFundAccount(contactID, upiID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create fund account"})
	}

	payoutBody, _ := json.Marshal(map[string]interface{}{
		"account_number":  accountNumber,
		"fund_account_id": fundAccountID,
		"amount":          amountPaise,
		"currency":        "INR",
		"mode":            "UPI",
		"purpose":         "refund",
		"notes": map[string]string{
			"user_id": userID,
		},
	})

	req, _ := http.NewRequest("POST", razorpayPayoutURL, bytes.NewBuffer(payoutBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in UPI PayOut API request:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process UPI PayOut"})
	}
	defer resp.Body.Close()

	var payoutResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&payoutResponse)

	log.Println("UPI PayOut request successful:", payoutResponse)
	return c.JSON(http.StatusOK, payoutResponse)
}

func main() {
	loadEnv()
	e := echo.New()

	e.POST("/create/user", generateUserIDHandler)
	e.POST("/create/contact", createContactHandler)
	e.POST("/create/fund_account", createFundAccountHandler)
	e.GET("/upi/collect", upiCollect)
	e.GET("/upi/payout", upiPayout)

	log.Println("Starting server on port 8080...")
	e.Logger.Fatal(e.Start(":8080"))
}
