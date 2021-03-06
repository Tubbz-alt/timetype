package timetype

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClock_GoString(t *testing.T) {
	s := Clock(time.Date(0, time.January, 1, 13, 24, 0, 0, time.UTC)).GoString()
	assert.Equal(t, "timetype.NewClock(13, 24, 0, UTC)", s)
}

func TestClock_String(t *testing.T) {
	s := Clock(time.Date(0, time.January, 1, 17, 54, 0, 0, time.UTC)).String()
	assert.Equal(t, "17:54:00 UTC", s)
}

func TestClock_UnmarshalJSON(t *testing.T) {
	var c Clock
	err := c.UnmarshalJSON([]byte("\"19:24:00.000000\""))
	require.NoError(t, err)
	assert.Equal(t, Clock(time.Date(0, time.January, 1, 19, 24, 0, 0, time.UTC)), c)

	// errors
	err = c.UnmarshalJSON([]byte("19:24:00.000000")) // time should be presented as string
	require.Error(t, err)
	assert.IsType(t, &errExternal{}, err, "time should be escaped in quotes")

	err = c.UnmarshalJSON([]byte("32145")) // invalid clock format
	assert.EqualError(t, err, "timetype: invalid clock", "clock should be in format \"15:04:05\"")
	assert.Equal(t, ErrInvalidClock, err)

	err = c.UnmarshalJSON([]byte("\"19:24:c00.000000\""))
	require.Error(t, err)
	assert.IsType(t, &UnknownFormatError{}, err, "invalid character \"c\" in seconds")
}

func TestNewClock(t *testing.T) {
	assert.Equal(t, Clock(time.Date(0, time.January, 1, 13, 24, 32, 0, time.Local)),
		NewClock(13, 24, 32, 0, time.Local))

	assert.Equal(t, Clock(time.Date(0, time.January, 1, 23, 59, 59, 0, time.UTC)),
		NewUTCClock(23, 59, 59, 0))
}

func TestErrExternal_Error(t *testing.T) {
	assert.EqualError(t, wrapExternalErr(errors.New("some test error")), "some test error")
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	var d Duration
	err := d.UnmarshalJSON([]byte("\"1h5m3s\""))
	require.NoError(t, err)
	assert.Equal(t, Duration(time.Hour+5*time.Minute+3*time.Second), d)

	err = d.UnmarshalJSON([]byte("3903000000000"))
	require.NoError(t, err)
	assert.Equal(t, Duration(time.Hour+5*time.Minute+3*time.Second), d)

	// errors
	err = d.UnmarshalJSON([]byte("true"))
	require.EqualError(t, err, "timetype: invalid duration", "passed bool to type duration")
	assert.Equal(t, ErrInvalidDuration, err)

	err = d.UnmarshalJSON([]byte("1h5m3s"))
	require.Error(t, err)
	assert.IsType(t, &errExternal{}, err, "duration should be escaped in quotes or passed as integer")

	err = d.UnmarshalJSON([]byte("\"123\""))
	require.Error(t, err)
	assert.IsType(t, &errExternal{}, err, "passed empty string to time parser")
}

func TestUnknownFormatError_Error(t *testing.T) {
	ue := UnknownFormatError{
		Errors: []error{
			errors.New("one"),
			errors.New("two"),
			errors.New("three"),
		},
		Layouts: []string{
			ISO8601Clock,
			ISO8601ClockMicro,
			"some unknown layout",
		},
		Val: "123456",
	}
	assert.EqualError(t, &ue,
		"timetype: failed to parse \"123456\" in layouts: [\"15:04:05\", \"15:04:05.000000\", \"some unknown layout\"]")
}

func TestDuration_Scan(t *testing.T) {
	tbl := []struct {
		arg      interface{}
		expected Duration
		err      string
	}{
		{
			arg:      nil,
			expected: Duration(0),
		},
		{
			arg:      5 * time.Minute,
			expected: Duration(5 * time.Minute),
		},
		{
			arg:      float64(10*time.Second + 1*time.Microsecond),
			expected: Duration(10*time.Second + 1*time.Microsecond),
		},
		{
			arg:      int64(32 * time.Hour),
			expected: Duration(32 * time.Hour),
		},
		{
			arg:      time.Duration(32 * time.Hour),
			expected: Duration(32 * time.Hour),
		},
		{
			arg:      `"5h3m2s"`,
			expected: Duration(5*time.Hour + 3*time.Minute + 2*time.Second),
		},
		{
			arg:      []byte(`"2h3m"`),
			expected: Duration(2*time.Hour + 3*time.Minute),
		},
		{
			arg: 'c',
			err: "timetype: invalid duration",
		},
	}
	for i, tt := range tbl {
		var d Duration
		err := d.Scan(tt.arg)
		if tt.err != "" {
			assert.EqualError(t, err, tt.err, "case #%d", i)
		} else {
			assert.NoError(t, err, "case #%d", i)
		}
		assert.Equal(t, tt.expected, d, "case #%d", i)
	}
}

func TestDuration_Value(t *testing.T) {
	tbl := []struct {
		arg      Duration
		expected driver.Value
	}{
		{
			arg:      Duration(2*time.Hour + 3*time.Minute),
			expected: int64(2*time.Hour + 3*time.Minute),
		},
		{
			arg:      Duration(5*time.Hour + 3*time.Minute + 2*time.Second),
			expected: int64(5*time.Hour + 3*time.Minute + 2*time.Second),
		},
		{
			arg:      Duration(1 * time.Second),
			expected: int64(1 * time.Second),
		},
		{
			arg:      Duration(1 * time.Millisecond),
			expected: int64(1 * time.Millisecond),
		},
		{
			arg:      Duration(1 * time.Nanosecond),
			expected: int64(1 * time.Nanosecond),
		},
	}

	for i, tt := range tbl {
		actual, err := tt.arg.Value()
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, actual, "case #%d", i)
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	bytes, err := Duration(time.Hour + 5*time.Minute + 3*time.Second).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, []byte(`"1h5m3s"`), bytes)
}

func TestClock_MarshalJSON(t *testing.T) {
	bytes, err := Clock(time.Date(0, time.January, 1, 19, 24, 0, 0, time.UTC)).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, []byte(`"19:24:00.000000"`), bytes)
}

func TestClock_Scan(t *testing.T) {
	tbl := []struct {
		arg      interface{}
		expected Clock
		err      string
	}{
		{
			arg:      nil,
			expected: Clock(time.Time{}),
		},
		{
			arg:      time.Date(0, time.January, 1, 2, 19, 30, 0, time.UTC),
			expected: Clock(time.Date(0, time.January, 1, 2, 19, 30, 0, time.UTC)),
		},
		{
			arg:      `19:24:00.000000`,
			expected: Clock(time.Date(0, time.January, 1, 19, 24, 0, 0, time.UTC)),
		},
		{
			arg:      []byte(`2:21:55.000000`),
			expected: Clock(time.Date(0, time.January, 1, 2, 21, 55, 0, time.UTC)),
		},
		{
			arg:      2567,
			expected: Clock{},
			err:      "timetype: invalid clock",
		},
		{
			arg:      "abacaba",
			expected: Clock{},
			err:      "timetype: failed to parse \"abacaba\" in layouts: [\"15:04:05\", \"15:04:05.000000\"]",
		},
		{
			arg:      []byte("abacaba"),
			expected: Clock{},
			err:      "timetype: failed to parse \"abacaba\" in layouts: [\"15:04:05\", \"15:04:05.000000\"]",
		},
	}

	for i, tt := range tbl {
		c := Clock{}
		err := c.Scan(tt.arg)
		if tt.err != "" {
			assert.EqualError(t, err, tt.err, "case #%d", i)
		} else {
			assert.NoError(t, err, "case #%d", i)
		}
		assert.Equal(t, tt.expected, c, "case #%d", i)
	}
}

func TestClock_Value(t *testing.T) {
	tbl := []struct {
		arg      Clock
		expected driver.Value
	}{
		{
			arg:      Clock(time.Date(0, time.January, 1, 19, 24, 0, 0, time.UTC)),
			expected: driver.Value(`19:24:00.000000`),
		},
		{
			arg:      Clock(time.Date(0, time.January, 1, 2, 21, 55, 0, time.UTC)),
			expected: driver.Value(`02:21:55.000000`),
		},
		{
			arg:      Clock(time.Date(0, time.January, 1, 2, 19, 30, 0, time.UTC)),
			expected: driver.Value(`02:19:30.000000`),
		},
	}

	for i, tt := range tbl {
		actual, err := tt.arg.Value()
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, actual, "case #%d", i)
	}
}
