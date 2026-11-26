package services_test

// Port of packages/flutter/test/services/delta_text_input_test.dart
// (Flutter 3.50.1); nothing skipped (Dart's AssertionError text also
// holds the failed condition; the messages are checked).

import (
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// fakeDeltaTextInputClient is FakeDeltaTextInputClient.
type fakeDeltaTextInputClient struct {
	fakeTextInputClient
	deltas []services.TextEditingDelta
}

func (c *fakeDeltaTextInputClient) UpdateEditingValueWithDeltas(deltas []services.TextEditingDelta) {
	c.latestMethodCall = "updateEditingValueWithDeltas"
	c.deltas = deltas
}

func (c *fakeDeltaTextInputClient) configuration() services.TextInputConfiguration {
	return services.NewTextInputConfiguration(services.TextInputConfigurationOptions{EnableDeltaModel: true})
}

func sendDeltas(clientID int, deltaJSON string) {
	sendTextInput(methodMessage("TextInputClient.updateEditingStateWithDeltas",
		[]any{clientID, jsonDecode(`{"deltas": [` + deltaJSON + `]}`)}))
}

func TestDeltaTextInputClientConfigurationAndUpdates(t *testing.T) {
	textInputChannelTest(t, func(*fakeTextChannel) {
		client := &fakeDeltaTextInputClient{fakeTextInputClient: *newFakeTextInputClient(services.TextEditingValueEmpty)}
		services.TextInputAttach(client, client.configuration())
		expect(client.configuration().EnableDeltaModel, "enableDeltaModel")
		expect(client.latestMethodCall == "", "%s", client.latestMethodCall)
		sendDeltas(1, deltaJSON("", "let there be text", 0, 0, 17, -1, -1))
		expect(client.latestMethodCall == "updateEditingValueWithDeltas", "%s", client.latestMethodCall)
		// Own: the delta reached the client.
		insertion, ok := client.deltas[0].(*services.TextEditingDeltaInsertion)
		expect(ok && len(client.deltas) == 1 && insertion.TextInserted == "let there be text", "%v", client.deltas)
	})
}

func TestDeltaInvalidTextRangeFailsLoudly(t *testing.T) {
	cases := []struct {
		name, text, delta string
		want              []string
	}{
		{"NonTextUpdate", "1", deltaJSON("1", "", -1, -1, 3, -1, -1), []string{
			"The selection range: TextSelection.collapsed(offset: 3, affinity: TextAffinity.downstream, isDirectional: false)",
			"is not within the bounds of text: 1 of length: 1"}},
		{"Faulty deltaStart and deltaEnd", "", deltaJSON("", "hello", 0, 1, 5, -1, -1), []string{
			"The delta range: TextRange(start: 0, end: 5)", "is not within the bounds of text:  of length: 0"}},
		{"Faulty Selection", "", deltaJSON("", "hello", 0, 0, 6, -1, -1), []string{
			"The selection range: TextSelection.collapsed(offset: 6, affinity: TextAffinity.downstream, isDirectional: false)",
			"is not within the bounds of text: hello of length: 5"}},
		{"Faulty Composing Region", "worl", deltaJSON("worl", "world", 0, 4, 5, 0, 6), []string{
			"The composing range: TextRange(start: 0, end: 6)", "is not within the bounds of text: world of length: 5"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			record := recordErrors(t)
			textInputChannelTest(t, func(*fakeTextChannel) {
				client := &fakeDeltaTextInputClient{fakeTextInputClient: *newFakeTextInputClient(tev(c.text))}
				services.TextInputAttach(client, client.configuration())
				sendDeltas(-1, c.delta)
				expect(len(*record) == 1, "%d errors", len(*record))
				text := dart.ToString((*record)[0].Exception)
				for _, w := range c.want {
					expect(strings.Contains(text, w), "%q lacks %q", text, w)
				}
			})
		})
	}
}
