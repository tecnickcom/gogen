package jirasrv

// IssueUpdate represents the payload to update a Jira issue.
// Ref.: https://docs.atlassian.com/software/jira/docs/api/REST/9.17.0
type IssueUpdate struct {
	Transition      *Transition          `json:"transition,omitempty"`
	Fields          map[string]any       `json:"fields,omitempty"`
	Update          map[string][]FieldOp `json:"update,omitempty"`
	HistoryMetadata *HistoryMetadata     `json:"historyMetadata,omitempty"`
	Properties      []EntityProperty     `json:"properties,omitempty"`
}

// Transition represents a Jira issue transition.
type Transition struct {
	ID             string               `json:"id,omitempty"`
	Name           string               `json:"name,omitempty"`
	Description    string               `json:"description,omitempty"`
	OpsbarSequence int                  `json:"opsbarSequence,omitempty"`
	To             *Status              `json:"to,omitempty"`
	Fields         map[string]FieldMeta `json:"fields,omitempty"`
}

// Status represents a Jira issue status.
type Status struct {
	StatusColor    string          `json:"statusColor,omitempty"`
	Description    string          `json:"description,omitempty"`
	IconURL        string          `json:"iconUrl,omitempty"`
	Name           string          `json:"name,omitempty"`
	ID             string          `json:"id,omitempty"`
	StatusCategory *StatusCategory `json:"statusCategory,omitempty"`
}

// StatusCategory represents a Jira issue status category.
type StatusCategory struct {
	ID        int    `json:"id,omitempty"`
	Key       string `json:"key,omitempty"`
	ColorName string `json:"colorName,omitempty"`
	Name      string `json:"name,omitempty"`
}

// FieldMeta represents metadata about a field in a Jira issue.
type FieldMeta struct {
	Required        bool      `json:"required"`
	Schema          *JSONType `json:"schema,omitempty"`
	Name            string    `json:"name,omitempty"`
	FieldID         string    `json:"fieldId,omitempty"`
	AutoCompleteURL string    `json:"autoCompleteUrl,omitempty"`
	HasDefaultValue bool      `json:"hasDefaultValue,omitempty"`
	Operations      []string  `json:"operations,omitempty"`
	AllowedValues   []any     `json:"allowedValues,omitempty"`
	DefaultValue    any       `json:"defaultValue,omitempty"`
}

// JSONType represents the JSON schema type of a field in a Jira issue.
type JSONType struct {
	Type     string `json:"type,omitempty"`
	Items    string `json:"items,omitempty"`
	System   string `json:"system,omitempty"`
	Custom   string `json:"custom,omitempty"`
	CustomID int    `json:"customId,omitempty"`
}

// FieldOp represents a field operation in a Jira issue update.
type FieldOp map[string]any

// HistoryMetadata represents metadata about a Jira issue history item.
type HistoryMetadata struct {
	Type                   string                      `json:"type,omitempty"`
	Description            string                      `json:"description,omitempty"`
	DescriptionKey         string                      `json:"descriptionKey,omitempty"`
	ActivityDescription    string                      `json:"activityDescription,omitempty"`
	ActivityDescriptionKey string                      `json:"activityDescriptionKey,omitempty"`
	EmailDescription       string                      `json:"emailDescription,omitempty"`
	EmailDescriptionKey    string                      `json:"emailDescriptionKey,omitempty"`
	Actor                  *HistoryMetadataParticipant `json:"actor,omitempty"`
	Generator              *HistoryMetadataParticipant `json:"generator,omitempty"`
	Cause                  *HistoryMetadataParticipant `json:"cause,omitempty"`
	ExtraData              map[string]string           `json:"extraData,omitempty"`
}

// HistoryMetadataParticipant represents a participant in a Jira issue history item.
type HistoryMetadataParticipant struct {
	ID             string `json:"id,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	DisplayNameKey string `json:"displayNameKey,omitempty"`
	Type           string `json:"type,omitempty"`
	AvatarURL      string `json:"avatarUrl,omitempty"`
	URL            string `json:"url,omitempty"`
}

// EntityProperty represents a property of a Jira issue.
type EntityProperty struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}
