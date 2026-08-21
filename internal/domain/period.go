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

func (p Period) String() string {
	return fmt.Sprintf("%04d-%02d", p.Year, int(p.Month))
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

func (p Period) PreviousMonth() Period {
	previousTime := time.Date(p.Year, p.Month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	return Period{
		Year:  previousTime.Year(),
		Month: previousTime.Month(),
	}
}
