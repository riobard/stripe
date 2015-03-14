package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/client"
)

// the version string is injected during the build process
var version string

var (
	dev  bool              // development mode?
	keys map[string]string // public key -> secret key
)

func main() {
	addr := flag.String("addr", ":8080", "server address")
	flag.BoolVar(&dev, "dev", false, "development mode")
	keyFile := flag.String("keys", "/etc/stripe-keys.json", "Stripe keys")
	printVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *printVersion {
		println(version)
		return
	}

	f, err := os.Open(*keyFile)
	if err != nil {
		log.Fatalf("failed to open key file: %v", err)
	}

	if err := json.NewDecoder(f).Decode(&keys); err != nil {
		log.Fatalf("failed to parse key file: %v", err)
	}

	f.Close()

	http.HandleFunc("/", handle)

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

type Request struct {
	Pubkey   string // Stripe public key used to generate the token
	Token    string // Stripe checkout.js generated token
	Email    string // customer email
	Plan     string
	Quantity uint64 // How many of the plan to subscribe
	Once     bool   // Cancel the plan once subscribed
}

func handle(w http.ResponseWriter, r *http.Request) {
	if dev { // Allow CORS in dev mode
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	}

	switch r.Method {
	case "POST":
		break
	case "OPTIONS":
		// Allow CORS preflight requests
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if dev {
		log.Printf("%#v", req)
	}

	seckey, ok := keys[req.Pubkey]
	if !ok {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	}

	cli := &client.API{}
	cli.Init(seckey, nil)

	ps := &stripe.CustomerParams{
		Email:    req.Email,
		Plan:     req.Plan,
		Quantity: req.Quantity,
	}

	err := ps.SetSource(req.Token)
	if err != nil {
		log.Printf("unsupported source %v: %v", req.Token, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	cus, err := cli.Customers.New(ps)
	if err != nil {
		log.Printf("failed to subscribe (email=%q token=%q plan=%q x %v): %s", req.Email, req.Token, req.Plan, req.Quantity, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	if req.Once {
		for _, sub := range cus.Subs.Values {
			if err := cli.Subs.Cancel(sub.ID, &stripe.SubParams{Customer: cus.ID, EndCancel: true}); err != nil {
				log.Printf("failed to unsubscribe customer ID = %v: %s sub ID = %v", cus.ID, sub.ID, err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
		}
	}

	w.Write([]byte("OK"))
}
