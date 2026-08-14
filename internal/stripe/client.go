package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"stripe-invoice-go/internal/domain"
	"time"
)

type Client struct {
	httpClient    *http.Client
	sessionCookie string
	bearerToken   string
}

type BoostrapResponse struct {
	Profile struct {
		User UserResponse `json:"user"`
	} `json:"profile"`
}

type UserResponse struct {
	Id             string `json:"id"`
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	EmailConfirmed bool   `json:"emailConfirmed"`
	SessionType    string `json:"sessionType"`
	SessionApiKey  string `json:"sessionApiKey"`
}

type MerchantsResponse struct {
	Id        string     `json:"id"`
	Object    string     `json:"object"`
	Merchants []Merchant `json:"merchants"`
}

type Merchant struct {
	Nickname string `json:"nickname"`
	Token    string `json:"token"`
}

type InvoiceDocumentsResponse struct {
	Data []InvoiceResponse `json:"data"`
}

type InvoiceResponse struct {
	Link          string `json:"link"`
	CreatedString string `json:"createdString"`
}

var ErrStatusIsntOk = errors.New("status is not ok")
var ErrNoBearerToken = errors.New("no bearer token on response data")
var ErrInvoiceNotFound = errors.New("no invoice found for period")

func NewClient(ctx context.Context, cookie string) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		sessionCookie: cookie,
	}

	req, err := c.newAuthenticatedRequest(
		ctx,
		http.MethodGet,
		BootstrapAuthUrl().String(),
		"https://dashboard.stripe.com/dashboard",
		"",
	)

	if err != nil {
		return nil, err
	}

	body, err := c.executeRequestAndGetBody(req)
	if err != nil {
		return nil, err
	}

	var data BoostrapResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if len(data.Profile.User.SessionApiKey) == 0 {
		return nil, ErrNoBearerToken
	}

	c.bearerToken = data.Profile.User.SessionApiKey
	log.Printf("Actually logged: %+v", data.Profile.User.DisplayName)
	return c, nil
}

func (c *Client) FetchMerchants(ctx context.Context) ([]domain.Merchant, error) {
	req, err := c.newAuthenticatedRequest(
		ctx,
		http.MethodGet,
		ListMerchantUrl().String(),
		"https://dashboard.stripe.com/settings/user",
		"",
	)

	if err != nil {
		return nil, err
	}

	body, err := c.executeRequestAndGetBody(req)
	if err != nil {
		return nil, err
	}

	var data MerchantsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	merchants, err := transformMerchantRequestToMerchantDomain(data)
	if err != nil {
		return nil, err
	}

	return merchants, nil
}

func (c *Client) ListInvoiceDocuments(ctx context.Context, accountToken string, p domain.Period) (domain.Invoice, error) {
	req, err := c.newAuthenticatedRequest(
		ctx,
		http.MethodGet,
		ListInvoiceUrl().String(),
		"https://dashboard.stripe.com/settings/documents",
		accountToken,
	)

	if err != nil {
		return domain.Invoice{}, err
	}

	body, err := c.executeRequestAndGetBody(req)
	if err != nil {
		return domain.Invoice{}, err
	}

	var data InvoiceDocumentsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return domain.Invoice{}, err
	}

	invoice, err := getInvoiceFromListDocumentsResponse(data, p)
	if err != nil {
		return domain.Invoice{}, err
	}

	return invoice, nil
}

func (c *Client) DownloadPDF(ctx context.Context, accountToken string, url string) ([]byte, error) {
	req, err := c.newAuthenticatedRequest(
		ctx,
		http.MethodGet,
		DownloadPDFUrl(url).String(),
		"https://dashboard.stripe.com/settings/documents",
		accountToken,
	)

	if err != nil {
		return nil, err
	}

	body, err := c.executeRequestAndGetBody(req)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) newAuthenticatedRequest(
	ctx context.Context,
	method string,
	url string,
	referer string,
	merchantToken string,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{
		Name:  "__Host-session",
		Value: c.sessionCookie,
	})
	req.Header.Set("referer", referer)
	req.Header.Set("x-requested-with", "fetch")
	req.Header.Set("stripe-livemode", "true")
	req.Header.Set("accept", "*/*")

	if merchantToken != "" {
		req.Header.Set("stripe-account", merchantToken)
	}

	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	return req, nil
}

func (c *Client) executeRequestAndGetBody(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrStatusIsntOk
	}

	return body, nil
}

func transformMerchantRequestToMerchantDomain(response MerchantsResponse) ([]domain.Merchant, error) {
	var merchants []domain.Merchant
	for _, merchantResponse := range response.Merchants {
		if len(merchantResponse.Token) < 8 {
			log.Printf("skipping merchant: %v", merchantResponse.Nickname)
			continue
		}

		token := merchantResponse.Token
		merchants = append(merchants, domain.Merchant{
			Token:    token,
			Name:     merchantResponse.Nickname,
			Identify: token[len(token)-8:],
		})
	}

	return merchants, nil
}

func getInvoiceFromListDocumentsResponse(data InvoiceDocumentsResponse, period domain.Period) (domain.Invoice, error) {
	for _, invoiceResponse := range data.Data {
		if invoiceResponse.CreatedString == "" {
			continue
		}

		castToPeriod, err := domain.ParsePeriodFromStripeDate(invoiceResponse.CreatedString)
		if err != nil {
			return domain.Invoice{}, err
		}

		if castToPeriod == period {
			log.Println("Invoice Found")
			return domain.Invoice{
				Link:   invoiceResponse.Link,
				Period: period,
			}, nil
		}
	}

	log.Println("Invoice not found in list")
	return domain.Invoice{}, ErrInvoiceNotFound
}
