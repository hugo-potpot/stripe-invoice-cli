package domain

import "errors"

var ErrInvoiceNotFound = errors.New("no invoice found for period")
var ErrNoMerchants = errors.New("no merchant found for account")
