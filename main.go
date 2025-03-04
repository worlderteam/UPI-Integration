package main

import (
	"bytes"
	"encoding/json"
	
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

// Razorpay API URLs
const razorpayUPIURL = "https://api.razorpay.com/v1/payment_links"
const razorpayPayoutURL = "https://api.razorpay.com/v1/payouts"

// Load environment variables
func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

// Function to handle UPI Collect Request (Top-Up)
func upiCollect(c echo.Context) error {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")

	amount := c.QueryParam("amount")
	userID := c.QueryParam("user_id")

	// Prepare request payload
	requestBody, _ := json.Marshal(map[string]interface{}{
		"amount":        amount + "00",  // Razorpay needs amount in paise
		"currency":      "INR",
		"accept_partial": false,
		"description":   "UPI Payment for Wolonote",
		"customer": map[string]string{
			"name":    "User " + userID,
			"email":   "user@example.com",
			"contact": "9999999999",
		},
		"notify": map[string]bool{
			"sms":  true,
			"email": true,
		},
		"reminder_enable": true,
		"upi_link":        true,
	})

	// Create HTTP request
	req, _ := http.NewRequest("POST", razorpayUPIURL, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to initiate UPI Collect request"})
	}
	defer resp.Body.Close()

	// Decode response
	var razorpayResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&razorpayResponse)

	return c.JSON(http.StatusOK, razorpayResponse)
}

// Function to handle UPI PayOut (Withdrawals)
func upiPayout(c echo.Context) error {
	apiKey := os.Getenv("RAZORPAY_KEY")
	apiSecret := os.Getenv("RAZORPAY_SECRET")

	amount := c.QueryParam("amount")
	userID := c.QueryParam("user_id")
	upiID := c.QueryParam("upi_id") // User's UPI ID for withdrawal

	// Prepare request payload
	requestBody, _ := json.Marshal(map[string]interface{}{
		"account_number": "232323XXXXXX",  // Worlder's Business Bank Account
		"amount":         amount + "00",   // Amount in paise
		"currency":       "INR",
		"mode":           "UPI",
		"purpose":        "refund",
		"fund_account": map[string]interface{}{
			"account_type": "vpa",
			"vpa": map[string]string{
				"address": upiID, // User's actual UPI ID
			},
		},
		"notes": map[string]string{
			"user_id": userID,
		},
	})

	// Create HTTP request
	req, _ := http.NewRequest("POST", razorpayPayoutURL, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process UPI PayOut"})
	}
	defer resp.Body.Close()

	// Decode response
	var razorpayResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&razorpayResponse)

	return c.JSON(http.StatusOK, razorpayResponse)
}

func main() {
	loadEnv() // Load API keys from .env

	e := echo.New()

	// Define API Endpoints
	e.GET("/upi/collect", upiCollect)  // Initiate UPI Top-Up
	e.GET("/upi/payout", upiPayout)    // Process UPI Withdrawal

	// Start server on port 8080
	e.Logger.Fatal(e.Start(":8080"))
}
