package numtrie

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	node := New[int]()

	require.NotNil(t, node)
	require.Len(t, node.children, indexSize)
	require.Nil(t, node.value)
}

func TestNode_Add(t *testing.T) {
	t.Parallel()

	node := New[int]()
	val := 17

	require.True(t, node.Add("1-2-3", &val))
	require.False(t, node.Add("123", &val))
	require.False(t, node.Add("1B3", &val))
	require.True(t, node.Add("12", &val))
	require.True(t, node.Add("1", &val))
	require.False(t, node.Add("12", &val))
	require.True(t, node.Add("CFZ", &val))
	require.False(t, node.Add("239", &val))
}

func TestNode_Get(t *testing.T) {
	t.Parallel()

	node := New[int]()

	valA := 17
	node.Add("1", &valA)

	valB := 41
	node.Add("123", &valB)

	valC := 53
	node.Add("4567", &valC)

	valD := 59
	node.Add("456", &valD)

	valE := 61
	node.Add("07", &valE)

	valF := 67
	node.Add("0732", &valF)

	tests := []struct {
		name   string
		num    string
		exp    *int
		status int8
	}{
		{
			name:   "no match empty",
			num:    "",
			exp:    nil,
			status: StatusMatchEmpty,
		},
		{
			name:   "no match",
			num:    "999",
			exp:    nil,
			status: StatusMatchNo,
		},
		{
			name:   "full match B",
			num:    "123",
			exp:    &valB,
			status: StatusMatchFull,
		},
		{
			name:   "full match B with extra chars",
			num:    "1-2-3--",
			exp:    &valB,
			status: StatusMatchFull,
		},
		{
			name:   "full match C",
			num:    "4567",
			exp:    &valC,
			status: StatusMatchFull,
		},
		{
			name:   "prefix match +1",
			num:    "1234",
			exp:    &valB,
			status: StatusMatchPrefix,
		},
		{
			name:   "prefix match +2",
			num:    "12345",
			exp:    &valB,
			status: StatusMatchPrefix,
		},
		{
			name:   "partial match -1",
			num:    "456",
			exp:    &valD,
			status: StatusMatchPartial,
		},
		{
			name:   "partial match -2",
			num:    "45",
			exp:    nil,
			status: StatusMatchPartial,
		},
		{
			name:   "partial match -3",
			num:    "4",
			exp:    nil,
			status: StatusMatchPartial,
		},
		{
			name:   "partial prefix match +1",
			num:    "451",
			exp:    nil,
			status: StatusMatchPartialPrefix,
		},
		{
			name:   "partial prefix match +2",
			num:    "4511",
			exp:    nil,
			status: StatusMatchPartialPrefix,
		},
		{
			name:   "partial prefix match with val",
			num:    "4561",
			exp:    &valD,
			status: StatusMatchPartialPrefix,
		},
		{
			name:   "partial match -4 with current val",
			num:    "0",
			exp:    nil,
			status: StatusMatchPartial,
		},
		{
			name:   "partial match -2 with current val",
			num:    "07",
			exp:    &valE,
			status: StatusMatchPartial,
		},
		{
			name:   "partial match -1 with parent val",
			num:    "073",
			exp:    &valE,
			status: StatusMatchPartial,
		},
		{
			name:   "partial match -2 with parent val",
			num:    "075",
			exp:    &valE,
			status: StatusMatchPartialPrefix,
		},
		{
			name:   "full match with current val",
			num:    "0732",
			exp:    &valF,
			status: StatusMatchFull,
		},
		{
			name:   "partial prefix match with parent value",
			num:    "0739",
			exp:    &valE,
			status: StatusMatchPartialPrefix,
		},
		{
			name:   "prefix match with parent value",
			num:    "07325",
			exp:    &valF,
			status: StatusMatchPrefix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, status := node.Get(tt.num)

			require.Equal(t, tt.exp, got)
			require.Equal(t, tt.status, status)
		})
	}
}
