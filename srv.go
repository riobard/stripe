package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

// the version string is injected durign the build process
var version string

var (
	stripe *Stripe
	addr   string
	dir    string
)

func main() {
	flag.StringVar(&addr, "addr", ":8080", "server address")
	flag.StringVar(&dir, "dir", "/usr/local/etc/stripe", "directory")
	printVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *printVersion {
		println(version)
		return
	}

	key := os.Getenv("STRIPE_KEY")
	if key == "" {
		log.Fatal("STRIPE_KEY undefined")
	}
	stripe = NewStripe(key)
	http.HandleFunc("/", handle)

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET", "HEAD":
		http.ServeFile(w, r, dir+"/index.html")

	case "POST":
		email := r.PostFormValue("stripeEmail")
		tok := r.PostFormValue("stripeToken")
		plan := r.PostFormValue("stripePlan")

		if _, err := stripe.Subscribe(email, plan, tok); err != nil {
			log.Printf("[Stripe] failed to subscribe (email=%q plan=%q token=%q): %s", email, plan, tok, err)
			http.ServeFile(w, r, dir+"/failure.html")
			return
		}

		http.ServeFile(w, r, dir+"/success.html")

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
