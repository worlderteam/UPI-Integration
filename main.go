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
