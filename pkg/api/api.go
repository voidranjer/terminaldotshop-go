package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/stripe/stripe-go/v78"
	"github.com/terminaldotshop/terminal-sdk-go"
	"github.com/terminaldotshop/terminal/go/pkg/resource"

	"github.com/stripe/stripe-go/v78/token"
)

func Init() {
	stripe.Key = resource.Resource.StripePublic.Value
}

type FingerprintRequest struct {
	Fingerprint string `json:"fingerprint"`
}

type UserCredentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GetErrorMessage(err error) string {
	if apiError, ok := err.(*terminal.Error); ok {
		return strings.Trim(apiError.JSON.ExtraFields["message"].Raw(), "\"")
	} else {
		return err.Error()
	}
}

func FetchUserToken(publicKey string) (*UserCredentials, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", "ssh")
	data.Set("client_secret", resource.Resource.AuthFingerprintKey.Value)
	data.Set("fingerprint", publicKey)
	data.Set("provider", "ssh")
	resp, err := http.PostForm(resource.Resource.Auth.Url+"/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return nil, errors.New(fmt.Sprintf("failed to auth: " + string(body)))
	}
	credentials := UserCredentials{}
	err = json.NewDecoder(resp.Body).Decode(&credentials)
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}

func StripeCreditCard(card *stripe.CardParams) (*stripe.Token, *string) {
	tokenParams := &stripe.TokenParams{Card: card}
	tokenResult, err := token.New(tokenParams)

	if err != nil {
		error := ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			error = stripeErr.Msg
		} else {
			error = err.Error()
		}
		return tokenResult, &error
	}

	return tokenResult, nil
}
