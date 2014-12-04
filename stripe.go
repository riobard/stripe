package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var ApiUrl = "https://api.stripe.com/v1"

type Stripe struct {
	key string
}

type Customer struct {
	ID string `json:"id"`
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Param   string `json:"param"`
}

func (err Error) Error() string {
	return fmt.Sprintf("Stripe error [%s]: %s", err.Type, err.Message)
}

func NewStripe(key string) *Stripe {
	return &Stripe{key: key}
}

func (s *Stripe) do(method string, action string, form url.Values) (*http.Response, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequest(method, ApiUrl+action, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.key)

	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

func (s *Stripe) Subscribe(email, plan, token string) (*Customer, error) {
	d := url.Values{}
	d.Set("email", email)
	d.Set("source", token)
	d.Set("plan", plan)
	rsp, err := s.do("POST", "/customers", d)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 {
		var t struct {
			Error Error `json:"error"`
		}
		if err := json.NewDecoder(rsp.Body).Decode(&t); err != nil {
			return nil, err
		}
		return nil, t.Error
	}

	var cus Customer
	if err := json.NewDecoder(rsp.Body).Decode(&cus); err != nil {
		return nil, err
	}
	return &cus, nil
}
