package apikey

import (
	"regexp"
	"time"

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

func ExpiredAtRule() []validation.Rule {
	return []validation.Rule{
		validation.Required,
		validation.By(func(value interface{}) error {
			if val, ok := value.(time.Time); ok {
				if val.Before(time.Now()) {
					return validation.NewError("validation_expired", "API KEY expirado, utilize su admin para actulizar su fecha de expiracion o genera una nueva clave para seguir operando")
				}
			}
			return nil
		}),
	}
}

func DescriptionRules() []validation.Rule {
	return []validation.Rule{
		validation.Required,
		validation.Length(nameMinLength, nameMaxLength),
	}
}

func ActiveRules() []validation.Rule {
	return []validation.Rule{
		validation.Required.Error("API KEY desactivado, utilize su admin para reactivarlo o genera una nueva clave para seguir operando"),
	}
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

func CompanyIdRules() []validation.Rule {
	return []validation.Rule{
		validation.Required,
	}
}

func (r *ApiKey) Validate() error {
	err := validation.ValidateStruct(r,
		validation.Field(&r.ExpiredAt, ExpiredAtRule()...),
		validation.Field(&r.IsActive, ActiveRules()...),
	)
	if err != nil {
		return err
	}

	return nil
}
