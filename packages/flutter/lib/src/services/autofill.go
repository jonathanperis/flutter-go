package services

// Port of packages/flutter/lib/src/services/autofill.dart (Flutter 3.50.1).

import (
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AutofillHints is the collection of autofill hint strings for common
// content types (Dart's abstract final class of string constants: a
// package variable of a struct of them).
var AutofillHints = struct {
	AddressCity, AddressCityAndState, AddressState, Birthday, BirthdayDay, BirthdayMonth, BirthdayYear,
	CountryCode, CountryName, CreditCardExpirationDate, CreditCardExpirationDay, CreditCardExpirationMonth,
	CreditCardExpirationYear, CreditCardFamilyName, CreditCardGivenName, CreditCardMiddleName, CreditCardName,
	CreditCardNumber, CreditCardSecurityCode, CreditCardType, Email, FamilyName, FullStreetAddress, Gender,
	GivenName, Impp, JobTitle, Language, Location, MiddleInitial, MiddleName, Name, NamePrefix, NameSuffix,
	NewPassword, NewUsername, Nickname, OneTimeCode, EmailOTPCode, OrganizationName, Password, Photo,
	PostalAddress, PostalAddressExtended, PostalAddressExtendedPostalCode, PostalCode, StreetAddressLevel1,
	StreetAddressLevel2, StreetAddressLevel3, StreetAddressLevel4, StreetAddressLine1, StreetAddressLine2,
	StreetAddressLine3, Sublocality, TelephoneNumber, TelephoneNumberAreaCode, TelephoneNumberCountryCode,
	TelephoneNumberDevice, TelephoneNumberExtension, TelephoneNumberLocal, TelephoneNumberLocalPrefix,
	TelephoneNumberLocalSuffix, TelephoneNumberNational, TransactionAmount, TransactionCurrency, URL,
	Username string
}{
	AddressCity: "addressCity", AddressCityAndState: "addressCityAndState", AddressState: "addressState",
	Birthday: "birthday", BirthdayDay: "birthdayDay", BirthdayMonth: "birthdayMonth", BirthdayYear: "birthdayYear",
	CountryCode: "countryCode", CountryName: "countryName",
	CreditCardExpirationDate: "creditCardExpirationDate", CreditCardExpirationDay: "creditCardExpirationDay",
	CreditCardExpirationMonth: "creditCardExpirationMonth", CreditCardExpirationYear: "creditCardExpirationYear",
	CreditCardFamilyName: "creditCardFamilyName", CreditCardGivenName: "creditCardGivenName",
	CreditCardMiddleName: "creditCardMiddleName", CreditCardName: "creditCardName",
	CreditCardNumber: "creditCardNumber", CreditCardSecurityCode: "creditCardSecurityCode",
	CreditCardType: "creditCardType", Email: "email", FamilyName: "familyName",
	FullStreetAddress: "fullStreetAddress", Gender: "gender", GivenName: "givenName", Impp: "impp",
	JobTitle: "jobTitle", Language: "language", Location: "location", MiddleInitial: "middleInitial",
	MiddleName: "middleName", Name: "name", NamePrefix: "namePrefix", NameSuffix: "nameSuffix",
	NewPassword: "newPassword", NewUsername: "newUsername", Nickname: "nickname", OneTimeCode: "oneTimeCode",
	EmailOTPCode: "emailOTPCode", OrganizationName: "organizationName", Password: "password", Photo: "photo",
	PostalAddress: "postalAddress", PostalAddressExtended: "postalAddressExtended",
	PostalAddressExtendedPostalCode: "postalAddressExtendedPostalCode", PostalCode: "postalCode",
	StreetAddressLevel1: "streetAddressLevel1", StreetAddressLevel2: "streetAddressLevel2",
	StreetAddressLevel3: "streetAddressLevel3", StreetAddressLevel4: "streetAddressLevel4",
	StreetAddressLine1: "streetAddressLine1", StreetAddressLine2: "streetAddressLine2",
	StreetAddressLine3: "streetAddressLine3", Sublocality: "sublocality", TelephoneNumber: "telephoneNumber",
	TelephoneNumberAreaCode: "telephoneNumberAreaCode", TelephoneNumberCountryCode: "telephoneNumberCountryCode",
	TelephoneNumberDevice: "telephoneNumberDevice", TelephoneNumberExtension: "telephoneNumberExtension",
	TelephoneNumberLocal: "telephoneNumberLocal", TelephoneNumberLocalPrefix: "telephoneNumberLocalPrefix",
	TelephoneNumberLocalSuffix: "telephoneNumberLocalSuffix", TelephoneNumberNational: "telephoneNumberNational",
	TransactionAmount: "transactionAmount", TransactionCurrency: "transactionCurrency", URL: "url",
	Username: "username",
}

// AutofillConfiguration is a collection of autofill related information
// that represents an AutofillClient. Compare with Equal.
type AutofillConfiguration struct {
	// Enabled is whether autofill should be enabled for the
	// TextInputClient.
	Enabled bool
	// UniqueIdentifier is the unique identifier of the TextInputClient.
	UniqueIdentifier string
	// AutofillHints are the autofill hints of the TextInputClient.
	AutofillHints []string
	// CurrentEditingValue is the current TextEditingValue of the
	// TextInputClient.
	CurrentEditingValue TextEditingValue
	// HintText is the optional hint text placed on the view that typically
	// suggests what sort of input the field accepts (nil: null).
	HintText *string
}

// AutofillConfigurationOptions are AutofillConfiguration's optional named
// parameters.
type AutofillConfigurationOptions struct{ HintText *string }

// NewAutofillConfiguration is AutofillConfiguration({required
// uniqueIdentifier, required autofillHints, required currentEditingValue,
// hintText}): an enabled configuration.
func NewAutofillConfiguration(uniqueIdentifier string, autofillHints []string, currentEditingValue TextEditingValue, opts ...AutofillConfigurationOptions) AutofillConfiguration {
	var o AutofillConfigurationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return AutofillConfiguration{Enabled: true, UniqueIdentifier: uniqueIdentifier, AutofillHints: autofillHints,
		CurrentEditingValue: currentEditingValue, HintText: o.HintText}
}

// AutofillConfigurationDisabled is AutofillConfiguration.disabled: an
// AutofillConfiguration that indicates the AutofillClient does not wish to
// be autofilled.
var AutofillConfigurationDisabled = AutofillConfiguration{
	Enabled: false, UniqueIdentifier: "", AutofillHints: []string{}, CurrentEditingValue: TextEditingValueEmpty,
}

// ToJSON returns a representation of this object as a JSON object (nil
// when autofill is disabled).
func (c AutofillConfiguration) ToJSON() []dart.MapEntry[string, any] {
	if !c.Enabled {
		return nil
	}
	m := []dart.MapEntry[string, any]{
		{Key: "uniqueIdentifier", Value: c.UniqueIdentifier},
		{Key: "hints", Value: c.AutofillHints},
		{Key: "editingValue", Value: c.CurrentEditingValue.ToJSON()},
	}
	if c.HintText != nil {
		m = append(m, dart.MapEntry[string, any]{Key: "hintText", Value: *c.HintText})
	}
	return m
}

// Equal is AutofillConfiguration's ==.
func (c AutofillConfiguration) Equal(other AutofillConfiguration) bool {
	return other.Enabled == c.Enabled &&
		other.UniqueIdentifier == c.UniqueIdentifier &&
		foundation.ListEquals(other.AutofillHints, c.AutofillHints) &&
		other.CurrentEditingValue.Equal(c.CurrentEditingValue) &&
		stringPtrEqual(other.HintText, c.HintText)
}

func (c AutofillConfiguration) String() string {
	d := []string{
		"enabled: " + strconv.FormatBool(c.Enabled),
		"uniqueIdentifier: " + c.UniqueIdentifier,
		"autofillHints: " + dartListString(c.AutofillHints, func(s string) string { return s }),
		"currentEditingValue: " + c.CurrentEditingValue.String(),
	}
	if c.HintText != nil {
		d = append(d, "hintText: "+*c.HintText)
	}
	return "AutofillConfiguration(" + strings.Join(d, ", ") + ")"
}

// AutofillClient is an object that represents an autofillable input field
// in the autofill workflow.
type AutofillClient interface {
	// AutofillID is the unique identifier of this AutofillClient.
	AutofillID() string
	// TextInputConfiguration is the TextInputConfiguration that describes
	// this AutofillClient.
	TextInputConfiguration() TextInputConfiguration
	// Autofill requests this AutofillClient update its TextEditingValue to
	// the given value.
	Autofill(newEditingValue TextEditingValue)
}

// AutofillScope is an ordered group within which AutofillClients are
// logically connected.
type AutofillScope interface {
	// GetAutofillClient gets the AutofillClient associated with the given
	// identifier (nil: null).
	GetAutofillClient(autofillID string) AutofillClient
	// AutofillClients are all the AutofillClients currently live in this
	// scope.
	AutofillClients() []AutofillClient
	// Attach allows a TextInputClient to attach to this scope.
	Attach(trigger TextInputClient, configuration TextInputConfiguration) *TextInputConnection
}

// AutofillScopeMixin is mixin AutofillScopeMixin: an AutofillScope's
// Attach, over the scope's AutofillClients. Embed it and call SetSelf with
// the scope.
type AutofillScopeMixin struct{ self AutofillScope }

// SetSelf points the mixin at the scope it is mixed into.
func (m *AutofillScopeMixin) SetSelf(self AutofillScope) { m.self = self }

// Attach attaches trigger with a configuration listing every client's
// configuration (_AutofillScopeTextInputConfiguration).
func (m *AutofillScopeMixin) Attach(trigger TextInputClient, configuration TextInputConfiguration) *TextInputConnection {
	if m.self == nil {
		panic("services: AutofillScopeMixin used before SetSelf; call SetSelf with the scope when creating it")
	}
	for _, client := range m.self.AutofillClients() {
		if !client.TextInputConfiguration().AutofillConfiguration.Enabled {
			panic(dart.NewAssertionError("Every client in AutofillScope.autofillClients must enable autofill"))
		}
	}
	return TextInputAttach(trigger, newAutofillScopeTextInputConfiguration(m.self, configuration))
}

// newAutofillScopeTextInputConfiguration is
// _AutofillScopeTextInputConfiguration: the current client's configuration
// (only the fields Dart's super call passes) plus every client's
// configuration (`fields`, read when encoded, as Dart's lazy Iterable).
func newAutofillScopeTextInputConfiguration(scope AutofillScope, current TextInputConfiguration) TextInputConfiguration {
	c := NewTextInputConfiguration(TextInputConfigurationOptions{
		ViewID:                current.ViewID,
		InputType:             &current.InputType,
		ObscureText:           current.ObscureText,
		Autocorrect:           &current.Autocorrect,
		SmartDashesType:       &current.SmartDashesType,
		SmartQuotesType:       &current.SmartQuotesType,
		EnableSuggestions:     &current.EnableSuggestions,
		InputAction:           &current.InputAction,
		TextCapitalization:    &current.TextCapitalization,
		KeyboardAppearance:    &current.KeyboardAppearance,
		ActionLabel:           current.ActionLabel,
		AutofillConfiguration: &current.AutofillConfiguration,
	})
	c.allConfigurations = func() []TextInputConfiguration {
		clients := scope.AutofillClients()
		out := make([]TextInputConfiguration, len(clients))
		for i, client := range clients {
			out[i] = client.TextInputConfiguration()
		}
		return out
	}
	return c
}
