package stripe

import (
	"net/url"
)

const baseUrl string = "dashboard.stripe.com"

func BootstrapAuthUrl() *url.URL {
	return &url.URL{
		Scheme:     "https",
		Host:       baseUrl,
		Path:       "/conversations/api/config",
		RawQuery:   "audience=stripe&entrypoint=dashboard",
		ForceQuery: true,
	}
}

func ListMerchantUrl() *url.URL {
	return &url.URL{
		Scheme:     "https",
		Host:       baseUrl,
		Path:       "/ajax/current_user_retrieve",
		RawQuery:   "include_only[]=id,merchants.token,merchants.nickname,object",
		ForceQuery: true,
	}
}

func ListInvoiceUrl() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   baseUrl,
		Path:   "/v1/settings/plans_and_fees/invoice_history_documents",
	}
}

func DownloadPDFUrl(link string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   baseUrl,
		Path:   link,
	}
}
