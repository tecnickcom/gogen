package jirasrv

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/timeutil"
)

func TestComment_MarshalJSON(t *testing.T) {
	t.Parallel()

	dtj := timeutil.DateTime[timeutil.TJira](time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)))

	tests := []struct {
		name string
		data Comment
	}{
		{
			name: "with minimal fields",
			data: Comment{},
		},
		{
			name: "with all fields",
			data: Comment{
				Self: "http://example.invalid/jira/rest/api/2/comment/10000",
				ID:   "10000",
				Author: &User{
					Self:        "http://example.invalid/jira/rest/api/2/user?username=john",
					Name:        "john",
					Key:         "john",
					Email:       "john@example.invalid",
					AvatarURLs:  map[string]string{"48x48": "http://example.invalid/jira/secure/useravatar?size=small&ownerId=john"},
					DisplayName: "John Doe",
					Active:      true,
					TimeZone:    "Australia/Sydney",
				},
				Body:         "This is a comment",
				RenderedBody: "<p>This is a comment</p>",
				UpdateAuthor: &User{
					Self:        "http://example.invalid/jira/rest/api/2/user?username=john",
					Name:        "john",
					Key:         "john",
					Email:       "john@example.invalid",
					AvatarURLs:  map[string]string{"48x48": "http://example.invalid/jira/secure/useravatar?size=small&ownerId=john"},
					DisplayName: "john Doe",
					Active:      true,
					TimeZone:    "Australia/Sydney",
				},
				Created: &dtj,
				Updated: &dtj,
				Visibility: &Visibility{
					Type:  "role",
					Value: "Administrators",
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

			var unmarshaled Comment

			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			require.Equal(t, tt.data, unmarshaled)
		})
	}
}
