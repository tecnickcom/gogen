package sqlutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "succeeds with defaults",
			wantErr: false,
		},
		{
			name:    "fails with nil quoteIDFunc",
			opts:    []Option{WithQuoteIDFunc(nil)},
			wantErr: true,
		},
		{
			name:    "fails with nil quoteValueFunc",
			opts:    []Option{WithQuoteValueFunc(nil)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(tt.opts...)

			require.Equal(t, tt.wantErr, err != nil, "New() error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}

func Test_defaultSQLUtil(t *testing.T) {
	t.Parallel()

	c := defaultSQLUtil()
	require.NotNil(t, c.quoteIDFunc)
	require.NotNil(t, c.quoteValueFunc)
}

func Test_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		c       *SQLUtil
		wantErr bool
	}{
		{
			name: "fail with invalid quoteIDFunc function",
			c: func() *SQLUtil {
				c := defaultSQLUtil()
				c.quoteIDFunc = nil

				return c
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid quoteValueFunc function",
			c: func() *SQLUtil {
				c := defaultSQLUtil()
				c.quoteValueFunc = nil

				return c
			}(),
			wantErr: true,
		},
		{
			name: "succeed with no errors",
			c: func() *SQLUtil {
				c := defaultSQLUtil()
				return c
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.c.validate()

			require.Equal(t, tt.wantErr, err != nil, "validate() error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}

func Test_defaultQuoteID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "empty string",
			id:   "",
			want: "",
		},
		{
			name: "single value",
			id:   "test1",
			want: "`test1`",
		},
		{
			name: "two parts",
			id:   "parent.child",
			want: "`parent`.`child`",
		},
		{
			name: "multiple parts",
			id:   "parent.child.name",
			want: "`parent`.`child`.`name`",
		},
		{
			name: "multiple parts with space",
			id:   "one two.three four",
			want: "`one two`.`three four`",
		},
		{
			name: "escape backtick",
			id:   "test`4",
			want: "`test``4`",
		},
		{
			name: "escape multiple backtick",
			id:   "test```4",
			want: "`test``````4`",
		},
		{
			name: "special characters",
			id:   "test" + string([]byte{'\'', '"', '`', 0, '\n', '\r', '\\', '\032'}),
			want: "`test'\"``\\0\\n\\r\\\\\\Z`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := defaultQuoteID(tt.id)

			require.Equal(t, tt.want, got, "QuoteID() got = %v, want %v", got, tt.want)
		})
	}
}

func Test_defaultQuoteValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{
			name: "empty string",
			val:  "",
			want: "",
		},
		{
			name: "simple value",
			val:  "test1",
			want: "'test1'",
		},
		{
			name: "value with spaces",
			val:  "one two three",
			want: "'one two three'",
		},
		{
			name: "escape single quote",
			val:  "test'2",
			want: "'test''2'",
		},
		{
			name: "escape multiple quotes",
			val:  "test'''2",
			want: "'test''''''2'",
		},
		{
			name: "special characters",
			val:  "test" + string([]byte{'\'', '"', '`', 0, '\n', '\r', '\\', '\032'}),
			want: "'test''\"`\\0\\n\\r\\\\\\Z'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := defaultQuoteValue(tt.val)

			require.Equal(t, tt.want, got, "QuoteID() got = %v, want %v", got, tt.want)
		})
	}
}

// Test_QuoteValue_Escaping exercises the default escaping behavior through the
// exported QuoteValue entry point, covering single quotes, backticks and every
// control character the escaper handles (\0, \n, \r, \\, \032/\Z).
func Test_QuoteValue_Escaping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{
			name: "single quote doubled",
			val:  "o'reilly",
			want: "'o''reilly'",
		},
		{
			name: "backtick left untouched in value",
			val:  "a`b",
			want: "'a`b'",
		},
		{
			name: "null byte escaped",
			val:  "a" + string([]byte{0}) + "b",
			want: `'a\0b'`,
		},
		{
			name: "newline escaped",
			val:  "a\nb",
			want: `'a\nb'`,
		},
		{
			name: "carriage return escaped",
			val:  "a\rb",
			want: `'a\rb'`,
		},
		{
			name: "backslash escaped",
			val:  `a\b`,
			want: `'a\\b'`,
		},
		{
			name: "substitute (\\032) escaped to Z",
			val:  "a\032b",
			want: `'a\Zb'`,
		},
		{
			name: "all control characters and quote together",
			val:  string([]byte{'\'', 0, '\n', '\r', '\\', '\032', '`'}),
			want: "'''\\0\\n\\r\\\\\\Z`'",
		},
	}

	c, err := New()
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := c.QuoteValue(tt.val)

			require.Equal(t, tt.want, got, "QuoteValue() got = %v, want %v", got, tt.want)
		})
	}
}

// Test_QuoteID_Escaping exercises the default escaping behavior through the
// exported QuoteID entry point, covering backtick-doubling, single quotes and
// every control character the escaper handles (\0, \n, \r, \\, \032/\Z).
func Test_QuoteID_Escaping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "embedded backtick doubled",
			id:   "col`umn",
			want: "`col``umn`",
		},
		{
			name: "single quote left untouched in identifier",
			id:   "col'umn",
			want: "`col'umn`",
		},
		{
			name: "dotted identifier with backtick per segment",
			id:   "sch`ema.ta`ble",
			want: "`sch``ema`.`ta``ble`",
		},
		{
			name: "null byte escaped",
			id:   "a" + string([]byte{0}) + "b",
			want: "`a\\0b`",
		},
		{
			name: "newline escaped",
			id:   "a\nb",
			want: "`a\\nb`",
		},
		{
			name: "carriage return escaped",
			id:   "a\rb",
			want: "`a\\rb`",
		},
		{
			name: "backslash escaped",
			id:   `a\b`,
			want: "`a\\\\b`",
		},
		{
			name: "substitute (\\032) escaped to Z",
			id:   "a\032b",
			want: "`a\\Zb`",
		},
		{
			name: "all control characters and backtick together",
			id:   string([]byte{0, '\n', '\r', '\\', '\032', '`'}),
			want: "`\\0\\n\\r\\\\\\Z```",
		},
	}

	c, err := New()
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := c.QuoteID(tt.id)

			require.Equal(t, tt.want, got, "QuoteID() got = %v, want %v", got, tt.want)
		})
	}
}

func Test_QuoteID(t *testing.T) {
	t.Parallel()

	fn := func(s string) string { return "TEST" + s }
	c := &SQLUtil{quoteIDFunc: fn}
	s := "5237"
	got := c.QuoteID(s)
	want := fn(s)

	require.Equal(t, want, got, "QuoteID() got = %v, want %v", got, want)
}

func Test_QuoteValue(t *testing.T) {
	t.Parallel()

	fn := func(s string) string { return "TEST" + s }
	c := &SQLUtil{quoteValueFunc: fn}
	s := "5237"
	got := c.QuoteValue(s)
	want := fn(s)

	require.Equal(t, want, got, "QuoteValue() got = %v, want %v", got, want)
}
