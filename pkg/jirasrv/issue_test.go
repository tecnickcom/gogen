package jirasrv

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIssueUpdate_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data IssueUpdate
	}{
		{
			name: "empty",
			data: IssueUpdate{},
		},
		{
			name: "with all fields",
			data: IssueUpdate{
				Transition: &Transition{
					ID:             "1",
					Name:           "Start Progress",
					Description:    "Transition to In Progress",
					OpsbarSequence: 1,
					To: &Status{
						StatusColor: "blue",
						Description: "In Progress",
						IconURL:     "http://example.invalid/icon.png",
						Name:        "In Progress",
						ID:          "3",
						StatusCategory: &StatusCategory{
							ID:        2,
							Key:       "in-progress",
							ColorName: "blue",
							Name:      "In Progress",
						},
					},
					Fields: map[string]FieldMeta{
						"customfield_10000": {
							Required: true,
							Schema: &JSONType{
								Type:   "string",
								System: "string",
							},
							Name:    "Custom Field 10000",
							FieldID: "customfield_10000",
						},
					},
				},
				Fields: map[string]any{
					"summary":     "Updated issue summary",
					"description": "Updated issue description",
				},
				Update: map[string][]FieldOp{
					"labels": {
						{"add": "new-label"},
						{"remove": "old-label"},
					},
				},
				HistoryMetadata: &HistoryMetadata{
					Type:                   "jira",
					Description:            "Issue updated via API",
					DescriptionKey:         "issue.updated",
					ActivityDescription:    "Updated issue fields",
					ActivityDescriptionKey: "issue.activity.updated",
					EmailDescription:       "Issue updated",
					EmailDescriptionKey:    "issue.email.updated",
					Actor: &HistoryMetadataParticipant{
						ID:             "15",
						DisplayName:    "API User",
						DisplayNameKey: "api.user",
						Type:           "user",
						AvatarURL:      "http://example.invalid/avatar1.png",
						URL:            "http://example.invalid/users/1",
					},
					Generator: &HistoryMetadataParticipant{
						ID:             "2",
						DisplayName:    "API User",
						DisplayNameKey: "api.user",
						Type:           "user",
						AvatarURL:      "http://example.invalid/avatar2.png",
						URL:            "http://example.invalid/users/2",
					},
					Cause: &HistoryMetadataParticipant{
						ID:             "3",
						DisplayName:    "API User",
						DisplayNameKey: "api.user",
						Type:           "user",
						AvatarURL:      "http://example.invalid/avatar3.png",
						URL:            "http://example.invalid/users/3",
					},
					ExtraData: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
				Properties: []EntityProperty{
					{
						Key:   "property1",
						Value: "value1",
					},
					{
						Key:   "property2",
						Value: map[string]any{"subkey": "subvalue"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.data)
			require.NoError(t, err)

			var unmarshaled IssueUpdate

			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			require.Equal(t, tt.data, unmarshaled)
		})
	}
}
