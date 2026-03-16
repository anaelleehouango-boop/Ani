package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Response structure for generic API messages
type Response struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// WithdrawalRequest structure for withdrawals (OTP removed as per user request)
type WithdrawalRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Amount      int    `json:"amount"`
	Method      string `json:"method"` // e.g., "wave", "orangeMoney", "freeMoney"
}

// DepositRequest structure for deposits
type DepositRequest struct {
	PhoneNumber string `json:"phoneNumber"` // Required for balance tracking
	Amount      int    `json:"amount"`
}

// In-memory store for user balances (simulated).
// Key: phone number, Value: balance.
// In a real application, this would be a database.
var balanceStore = make(map[string]int)
var balanceMutex sync.Mutex // Mutex to protect balanceStore for concurrent access

// init function for random seed
func init() {
	rand.Seed(time.Now().UnixNano())
	// Initialize some test balances for known phone numbers
	balanceStore["771234567"] = 10000 // Example initial balance
	balanceStore["779876543"] = 5000  // Another example
	log.Printf("Initialized test balances for 771234567: %d, 779876543: %d", balanceStore["771234567"], balanceStore["779876543"])
}

// helloHandler serves a simple JSON response
func helloHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Message: "Hello from Go!",
		Status:  "success",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWithdrawal processes a withdrawal request (OTP validation removed as per user request)
func handleWithdrawal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WithdrawalRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "Invalid request body"})
		return
	}

	if req.PhoneNumber == "" || req.Amount <= 0 || req.Method == "" {
		sendJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "Phone number, amount, and method are required"})
		return
	}

	balanceMutex.Lock()
	defer balanceMutex.Unlock() // Ensure mutex is unlocked even on early returns

	currentBalance, exists := balanceStore[req.PhoneNumber]
	if !exists {
		// Initialize balance for new phone numbers with a default if they don't exist
		balanceStore[req.PhoneNumber] = 1000 // Default initial balance
		currentBalance = 1000
		log.Printf("Initialized balance for new phone number %s: %d during withdrawal attempt.", req.PhoneNumber, currentBalance)
	}

	if currentBalance < req.Amount {
		sendJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: fmt.Sprintf("Solde insuffisant. Votre solde actuel est %d CFA.", currentBalance)})
		return
	}

	// Simulate deduction
	balanceStore[req.PhoneNumber] -= req.Amount
	newBalance := balanceStore[req.PhoneNumber]

	// --- REAL PAYMENT GATEWAY INTEGRATION POINT (WITHDRAWAL) ---
	// In a real application, THIS is where you would call an external API
	// (e.g., Wave API, Orange Money API, Free Money API) to initiate the actual
	// mobile money withdrawal transaction.
	//
	// Example (pseudocode):
	// paymentProviderResponse, err := paymentGateway.InitiateWithdrawal(req.PhoneNumber, req.Amount, req.Method)
	// if err != nil || paymentProviderResponse.Status != "SUCCESS" {
	//     // IMPORTANT: Revert balance if external withdrawal fails to maintain consistency
	//     balanceStore[req.PhoneNumber] += req.Amount // Revert
	//     log.Printf("External withdrawal failed for %s, amount %d. Balance reverted. Error: %v", req.PhoneNumber, req.Amount, err)
	//     sendJSONResponse(w, http.StatusInternalServerError, Response{Status: "error", Message: "Failed to initiate external withdrawal. Please try again."})
	//     return
	// }
	// --- END REAL PAYMENT GATEWAY INTEGRATION POINT ---

	log.Printf("Withdrawal of %d CFA for %s via %s. New balance: %d", req.Amount, req.PhoneNumber, req.Method, newBalance)
	sendJSONResponse(w, http.StatusOK, Response{Status: "success", Message: fmt.Sprintf("Retrait de %d CFA via %s vers %s réussi! Nouveau solde: %d CFA", req.Amount, req.Method, req.PhoneNumber, newBalance)})
}

// handleDeposit processes a deposit request
func handleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DepositRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "Invalid request body"})
		return
	}

	if req.PhoneNumber == "" || req.Amount <= 0 {
		sendJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "Phone number and amount are required"})
		return
	}

	balanceMutex.Lock()
	defer balanceMutex.Unlock() // Ensure mutex is unlocked even on early returns

	currentBalance, exists := balanceStore[req.PhoneNumber]
	if !exists {
		// Initialize balance for new phone numbers on deposit as well
		balanceStore[req.PhoneNumber] = 0
		currentBalance = 0
		log.Printf("Initialized balance for new phone number %s during deposit.", req.PhoneNumber)
	}

	// Simulate addition
	balanceStore[req.PhoneNumber] += req.Amount
	newBalance := balanceStore[req.PhoneNumber]

	// --- REAL PAYMENT GATEWAY INTEGRATION POINT (DEPOSIT) ---
	// For deposits, typically a user would initiate a transaction from their
	// mobile money app to a specific merchant number/code, and the payment
	// provider would send a webhook or callback to your server to confirm
	// the successful deposit.
	//
	// This `handleDeposit` function currently simulates the final step (adding
	// funds to the user's balance) as if a webhook confirmed a successful deposit.
	// In a real system, you'd likely have:
	// 1. A frontend prompt instructing the user on how to deposit.
	// 2. A webhook endpoint listening for confirmations from payment providers.
	// 3. This `handleDeposit` logic would then be part of that webhook handler.
	// --- END REAL PAYMENT GATEWAY INTEGRATION POINT ---

	log.Printf("Deposit of %d CFA for %s. New balance: %d", req.Amount, req.PhoneNumber, newBalance)
	sendJSONResponse(w, http.StatusOK, Response{Status: "success", Message: fmt.Sprintf("Dépôt de %d CFA vers %s réussi! Nouveau solde: %d CFA", req.Amount, req.PhoneNumber, newBalance)})
}

// Helper to send JSON responses
func sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// homeHandler serves the main landing page HTML
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<html>
<body style="font-family:system-ui;background:#1a1a1a;color:#eee;display:flex;align-items:center;justify-content:center;height:100vh;margin:0">
<div style="text-align:center">
<h1>🐹 Go Server</h1>
<p>Running on :8080</p>
<p><a href="/api/hello" style="color:#00ADD8">Try the API →</a></p>
<p><a href="/game" style="color:#00ADD8">Play the Game →</a></p>
</div>
</body>
</html>`)
}

// gameHandler serves the game HTML file
func gameHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "game.html")
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/hello", helloHandler)
	http.HandleFunc("/game", gameHandler)
	http.HandleFunc("/api/withdraw", handleWithdrawal) // New handler
	http.HandleFunc("/api/deposit", handleDeposit)     // New handler

	fmt.Println("🐹 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}