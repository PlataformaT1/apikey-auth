package company

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	nameMinLength = 1
	nameMaxLength = 50
	rfcMinLength  = 12
	rfcMaxLength  = 13
	mailMinLength = 1
	mailMaxLength = 100
)

func NameRules() []validation.Rule {
	return []validation.Rule{
		validation.Required,
		validation.Length(nameMinLength, nameMaxLength),
	}
}

func DescriptionRules() []validation.Rule {
	return []validation.Rule{
		validation.Required,
		validation.Length(nameMinLength, nameMaxLength),
	}
}

func ActiveRules() []validation.Rule {
	return []validation.Rule{}
}

func PhoneRules() []validation.Rule {
	return []validation.Rule{
		validation.Required,
		validation.Match(regexp.MustCompile(`^\d{10}$`)),
	}
}

func MailRules() []validation.Rule {
	return []validation.Rule{
		validation.Required,
		validation.Length(mailMinLength, mailMaxLength),
		validation.Match(regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)),
	}
}

func RfcRules() []validation.Rule {
	return []validation.Rule{
		validation.Required,
		validation.Length(rfcMinLength, rfcMaxLength)}
}

func (r *Company) Validate() error {
	err := validation.ValidateStruct(r,
		validation.Field(&r.Name, NameRules()...),
		validation.Field(&r.Description, DescriptionRules()...),
		validation.Field(&r.Active, ActiveRules()...),
		validation.Field(&r.Phone, PhoneRules()...),
		validation.Field(&r.Mail, MailRules()...),
		validation.Field(&r.Rfc, RfcRules()...),
	)
	if err != nil {
		return err
	}

	return nil
}
