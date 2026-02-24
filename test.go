package main

import (
	"fmt"
	"net/http"
)

func middlewareAddHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Powered-By", "Go-Bear")
		next.ServeHTTP(w, r)
	})
}

func middlewareRequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("X-API-Key")
		if header == "secret-bear-key" {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error": "unauthorized"}`))
			return
		}
	})
}


type PaymentProcessor interface {
    Pay(amount float64) error
}

type CreditCardProcessor struct {
    CardNumber string
}

type PayPalProcessor struct {
    Email string
}

func (c CreditCardProcessor) Pay(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount cannot be zero or less, current value is %v", amount)
	}
	
	fmt.Printf("Charging Card: %v\n", c.CardNumber)

	return nil
}

func (p PayPalProcessor) Pay(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount cannot be zero or less, current value is %v", amount)
	}

	fmt.Printf("Processing Payment regarding Email: %v", p.Email)

	return nil
}