package services_test

// Port of packages/flutter/test/services/autofill_test.dart (Flutter
// 3.50.1). Skipped: "hashCode works correctly" (Go values have no
// hashCode).

import (
	"fmt"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// fakeAutofillClient is FakeAutofillClient.
type fakeAutofillClient struct {
	fakeTextInputClient
	textInputConfiguration services.TextInputConfiguration
	currentAutofillScope   services.AutofillScope
}

func (c *fakeAutofillClient) AutofillID() string { return fmt.Sprintf("%p", c) }
func (c *fakeAutofillClient) TextInputConfiguration() services.TextInputConfiguration {
	return c.textInputConfiguration
}
func (c *fakeAutofillClient) UpdateEditingValue(v services.TextEditingValue) {
	c.currentTextEditingValue = v
	c.latestMethodCall = "updateEditingValue"
}
func (c *fakeAutofillClient) CurrentAutofillScope() services.AutofillScope {
	return c.currentAutofillScope
}
func (c *fakeAutofillClient) Autofill(v services.TextEditingValue) { c.UpdateEditingValue(v) }

// fakeAutofillScope is FakeAutofillScope (with AutofillScopeMixin).
type fakeAutofillScope struct {
	services.AutofillScopeMixin
	ids     []string
	clients map[string]services.AutofillClient
}

func newFakeAutofillScope() *fakeAutofillScope {
	s := &fakeAutofillScope{clients: map[string]services.AutofillClient{}}
	s.SetSelf(s)
	return s
}

func (s *fakeAutofillScope) AutofillClients() []services.AutofillClient {
	out := make([]services.AutofillClient, len(s.ids))
	for i, id := range s.ids {
		out[i] = s.clients[id]
	}
	return out
}

func (s *fakeAutofillScope) GetAutofillClient(id string) services.AutofillClient {
	return s.clients[id]
}

func (s *fakeAutofillScope) register(client services.AutofillClient) {
	if _, ok := s.clients[client.AutofillID()]; !ok {
		s.ids = append(s.ids, client.AutofillID())
		s.clients[client.AutofillID()] = client
	}
}

func TestAutofillConfigurationDoesNotThrowIfTheHintListIsEmpty(t *testing.T) {
	services.NewAutofillConfiguration("id", []string{}, services.TextEditingValueEmpty)
}

func TestAutofillClientsSendConfigurationAndRespondToUpdateEditingStateWithTag(t *testing.T) {
	textInputChannelTest(t, func(channel *fakeTextChannel) {
		scope := newFakeAutofillScope()
		client1 := &fakeAutofillClient{fakeTextInputClient: *newFakeTextInputClient(tev("test1"))}
		client2 := &fakeAutofillClient{fakeTextInputClient: *newFakeTextInputClient(tev("test2"))}
		for i, c := range []*fakeAutofillClient{client1, client2} {
			autofill := services.NewAutofillConfiguration(c.AutofillID(), []string{fmt.Sprintf("client%d", i+1)}, c.currentTextEditingValue)
			c.textInputConfiguration = services.NewTextInputConfiguration(services.TextInputConfigurationOptions{AutofillConfiguration: &autofill})
		}
		scope.register(client1)
		scope.register(client2)
		client1.currentAutofillScope = scope
		client2.currentAutofillScope = scope

		scope.Attach(client1, client1.textInputConfiguration)

		expected := append(client1.textInputConfiguration.ToJSON(), mapEntry("fields", []any{
			client1.textInputConfiguration.ToJSON(), client2.textInputConfiguration.ToJSON(),
		}))
		mustValidate(channel, call("TextInput.setClient", []any{1, expected}))

		text2 := tev("Text 2")
		channel.incoming(call("TextInputClient.updateEditingStateWithTag", []any{0,
			map[string]any{client2.AutofillID(): jsonMessage.DecodeMessage(jsonMessage.EncodeMessage(text2.ToJSON()))}})).Await()
		expect(client2.currentTextEditingValue.Equal(text2), "%v", client2.currentTextEditingValue)
	})
}

func TestAutofillConfigurationEqualityOperator(t *testing.T) {
	hint := "hint"
	a := services.NewAutofillConfiguration("id1", []string{"client1"}, services.TextEditingValueEmpty, services.AutofillConfigurationOptions{HintText: &hint})
	b := services.NewAutofillConfiguration("id1", []string{"client1"}, services.TextEditingValueEmpty, services.AutofillConfigurationOptions{HintText: &hint})
	if !a.Equal(b) || a.Enabled != b.Enabled || a.UniqueIdentifier != b.UniqueIdentifier || *a.HintText != *b.HintText {
		t.Fatal("not equal")
	}
	// Own: hints and the hint text matter; disabled is not enabled.
	other := "other"
	if a.Equal(services.NewAutofillConfiguration("id1", []string{"client2"}, services.TextEditingValueEmpty, services.AutofillConfigurationOptions{HintText: &hint})) ||
		a.Equal(services.NewAutofillConfiguration("id1", []string{"client1"}, services.TextEditingValueEmpty, services.AutofillConfigurationOptions{HintText: &other})) ||
		a.Equal(services.AutofillConfigurationDisabled) {
		t.Fatal("equal")
	}
	if got := string(jsonMessage.EncodeMessage(a.ToJSON())); got != `{"uniqueIdentifier":"id1","hints":["client1"],"editingValue":{"text":"","selectionBase":-1,"selectionExtent":-1,"selectionAffinity":"TextAffinity.downstream","selectionIsDirectional":false,"composingBase":-1,"composingExtent":-1},"hintText":"hint"}` {
		t.Fatal(got)
	}
	if services.AutofillConfigurationDisabled.ToJSON() != nil {
		t.Fatal("disabled encodes")
	}
}
