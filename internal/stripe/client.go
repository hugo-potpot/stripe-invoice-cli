package stripe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

type MerchantBody struct {
	OperationName string            `json:"operationName"`
	Variables     MerchantVariables `json:"variables"`
	Query         string            `json:"query"`
}

type MerchantVariables struct {
	V2Context V2Context `json:"v2Context"`
}

type V2Context struct {
	LiveMode string `json:"liveMode"`
}

const merchantsQuery = `query V2GetUserAccessibleAccountsQuery($v2Context: StripeContextInput!) {
  v2GetUserAccessibleAccounts(context: $v2Context) {
    standalone_workspaces {
      id
      name
      merchant_id
    }
  }
}
`

type MerchantsResponse struct {
	MerchantDataResponse MerchantDataResponse `json:"data"`
	Errors               []GraphQLError       `json:"errors"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

type MerchantDataResponse struct {
	V2GetUserAccessibleAccounts V2GetUserAccessibleAccounts `json:"v2GetUserAccessibleAccounts"`
}

type V2GetUserAccessibleAccounts struct {
	StandaloneWorkspaces []Merchant `json:"standalone_workspaces"`
}

type Merchant struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	MerchantId string `json:"merchant_id"`
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
var ErrGraphQL = errors.New("graphql error")

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
		"",
		nil,
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
	slog.InfoContext(ctx, "bootstrap authenticated", "user", data.Profile.User.DisplayName)
	return c, nil
}

func (c *Client) FetchMerchants(ctx context.Context) ([]domain.Merchant, error) {
	req, err := c.createMerchantsRequest(ctx, ListMerchantUrl().String())

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

	if len(data.Errors) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrGraphQL, data.Errors[0].Message)
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
		accountToken,
		nil,
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
		accountToken,
		nil,
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
	merchantToken string,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{
		Name:  "__Host-session",
		Value: c.sessionCookie,
	})
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

func (c *Client) createMerchantsRequest(
	ctx context.Context,
	url string,
) (*http.Request, error) {
	payload, err := json.Marshal(MerchantBody{
		OperationName: "V2GetUserAccessibleAccountsQuery",
		Variables: MerchantVariables{
			V2Context: V2Context{
				LiveMode: "FORCE_LIVE",
			},
		},
		Query: merchantsQuery,
	})

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.AddCookie(&http.Cookie{
		Name:  "__Host-session",
		Value: c.sessionCookie,
	})
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("accept", "*/*")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("stripe-version", "2026-08-12-19.internal")
	req.Header.Set("authorization", "STRIPE-V2-SIG")

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
		slog.ErrorContext(req.Context(), "unexpected stripe response status", "status", resp.StatusCode, "body", string(body))
		return nil, ErrStatusIsntOk
	}

	return body, nil
}

func transformMerchantRequestToMerchantDomain(response MerchantsResponse) ([]domain.Merchant, error) {
	var merchants []domain.Merchant
	for _, merchantResponse := range response.MerchantDataResponse.V2GetUserAccessibleAccounts.StandaloneWorkspaces {
		if len(merchantResponse.MerchantId) < 8 {
			slog.Warn("skipping merchant with short merchant id", "name", merchantResponse.Name, "merchant_id", merchantResponse.MerchantId)
			continue
		}

		token := merchantResponse.MerchantId
		merchants = append(merchants, domain.Merchant{
			Token:    token,
			Name:     merchantResponse.Name,
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
			slog.Debug("invoice found for period", "period", period.String(), "link", invoiceResponse.Link)
			return domain.Invoice{
				Link:   invoiceResponse.Link,
				Period: period,
			}, nil
		}
	}

	slog.Debug("no invoice found for period", "period", period.String())
	return domain.Invoice{}, domain.ErrInvoiceNotFound
}
