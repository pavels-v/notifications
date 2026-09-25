package tmpl

import (
	"fmt"
	"regexp"
	"strings"

	"notifications/internal/domain"
)

const dateLayout = "2006-01-02"

var placeholderRe = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

func fields(c domain.Customer) map[string]string {
	return map[string]string{
		"credit_number": c.CreditNumber,
		"full_name":     c.FullName,
		"amount":        FormatAmount(c.AmountMinor, c.Currency),
		"due_date":      c.DueDate.Format(dateLayout),
	}
}

func Render(body string, c domain.Customer) (string, error) {
	values := fields(c)

	var unknown []string
	out := placeholderRe.ReplaceAllStringFunc(body, func(match string) string {
		name := match[1 : len(match)-1]
		value, ok := values[name]
		if !ok {
			unknown = append(unknown, name)
			return match
		}
		return value
	})

	if len(unknown) > 0 {
		return "", fmt.Errorf("%w: %s", domain.ErrUnknownPlaceholder, strings.Join(unknown, ", "))
	}

	return out, nil
}

func FormatAmount(minor int64, currency string) string {
	return fmt.Sprintf("%d.%02d %s", minor/100, minor%100, currency)
}
