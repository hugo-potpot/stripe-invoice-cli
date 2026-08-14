package domain

import (
	"errors"
	"fmt"
	"time"
)

var ErrFailedToParsePeriod = errors.New("failed to parse period")

type Period struct {
	Year  int
	Month time.Month
}

func ParsePeriod(s string) (Period, error) {
	parsedTime, err := time.Parse("2006-01", s)
	if err != nil {
		return Period{}, fmt.Errorf("parsing %q: %w", s, ErrFailedToParsePeriod)
	}

	return Period{
		Year:  parsedTime.Year(),
		Month: parsedTime.Month(),
	}, nil
}

func ParsePeriodFromStripeDate(stripeDate string) (Period, error) {
	parsedTime, err := time.Parse(time.DateOnly, stripeDate)

	if err != nil {
		return Period{}, fmt.Errorf("parsing %q: %w", stripeDate, ErrFailedToParsePeriod)
	}

	return Period{
		Year:  parsedTime.Year(),
		Month: parsedTime.Month(),
	}, nil
}
