package tests_test

import (
	"testing"
	"time"

	c "github.com/atlasgurus/rulestone/condition"
	"github.com/stretchr/testify/require"
)

// TestTimeOperand_Conversions tests all conversion paths for TimeOperand
func TestTimeOperand_Conversions(t *testing.T) {
	refTime := time.Date(2024, 1, 15, 12, 30, 45, 123456789, time.UTC)
	expectedNano := refTime.UnixNano()

	t.Run("TimeOperand to IntOperand", func(t *testing.T) {
		timeOp := c.NewTimeOperand(refTime)
		intOp := timeOp.Convert(c.IntOperandKind)

		require.EqualValues(t, c.IntOperandKind, intOp.GetKind())
		require.Equal(t, c.IntOperand(expectedNano), intOp)
	})

	t.Run("TimeOperand to FloatOperand", func(t *testing.T) {
		timeOp := c.NewTimeOperand(refTime)
		floatOp := timeOp.Convert(c.FloatOperandKind)

		require.EqualValues(t, c.FloatOperandKind, floatOp.GetKind())
		require.Equal(t, c.FloatOperand(expectedNano), floatOp)
	})

	t.Run("TimeOperand to StringOperand", func(t *testing.T) {
		timeOp := c.NewTimeOperand(refTime)
		strOp := timeOp.Convert(c.StringOperandKind)

		require.EqualValues(t, c.StringOperandKind, strOp.GetKind())
		// Should be RFC3339Nano format
		expectedStr := refTime.Format(time.RFC3339Nano)
		require.Equal(t, c.StringOperand(expectedStr), strOp)
	})

	t.Run("IntOperand to TimeOperand (Unix nano)", func(t *testing.T) {
		intOp := c.NewIntOperand(expectedNano)
		timeOp := intOp.Convert(c.TimeOperandKind)

		require.EqualValues(t, c.TimeOperandKind, timeOp.GetKind())
		// Should reconstruct the time
		reconstructed := time.Time(timeOp.(c.TimeOperand))
		require.True(t, refTime.Equal(reconstructed))
	})

	t.Run("StringOperand to TimeOperand (RFC3339)", func(t *testing.T) {
		strOp := c.NewStringOperand("2024-01-15T12:30:45Z")
		timeOp := strOp.Convert(c.TimeOperandKind)

		require.EqualValues(t, c.TimeOperandKind, timeOp.GetKind())
		parsed := time.Time(timeOp.(c.TimeOperand))
		expected := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)
		require.True(t, expected.Equal(parsed))
	})

	t.Run("StringOperand to TimeOperand (various formats)", func(t *testing.T) {
		testCases := []string{
			"2024-01-15",
			"2024-01-15T12:30:45Z",
			"2024-01-15 12:30:45",
			"Jan 15, 2024",
			"01/15/2024",
		}

		for _, timeStr := range testCases {
			t.Run(timeStr, func(t *testing.T) {
				strOp := c.NewStringOperand(timeStr)
				timeOp := strOp.Convert(c.TimeOperandKind)

				// Should not be ErrorOperand
				require.NotEqualValues(t, c.ErrorOperandKind, timeOp.GetKind())
				require.EqualValues(t, c.TimeOperandKind, timeOp.GetKind())
			})
		}
	})

	t.Run("Invalid string to TimeOperand returns error", func(t *testing.T) {
		strOp := c.NewStringOperand("not a date")
		timeOp := strOp.Convert(c.TimeOperandKind)

		require.EqualValues(t, c.ErrorOperandKind, timeOp.GetKind())
	})

	t.Run("FloatOperand to TimeOperand (Unix nano)", func(t *testing.T) {
		floatOp := c.NewFloatOperand(float64(expectedNano))
		timeOp := floatOp.Convert(c.TimeOperandKind)

		require.EqualValues(t, c.TimeOperandKind, timeOp.GetKind())
		reconstructed := time.Time(timeOp.(c.TimeOperand))
		// Note: float64 can lose precision for large integers like Unix nanoseconds
		// so we check that the times are within a reasonable delta (1 microsecond)
		diff := refTime.Sub(reconstructed)
		if diff < 0 {
			diff = -diff
		}
		require.True(t, diff < time.Microsecond, "Time difference %v should be less than 1 microsecond", diff)
	})

	t.Run("TimeOperand to NullOperand", func(t *testing.T) {
		timeOp := c.NewTimeOperand(refTime)
		nullOp := timeOp.Convert(c.NullOperandKind)

		require.EqualValues(t, c.NullOperandKind, nullOp.GetKind())
	})

	t.Run("TimeOperand to TimeOperand (identity)", func(t *testing.T) {
		timeOp := c.NewTimeOperand(refTime)
		timeOp2 := timeOp.Convert(c.TimeOperandKind)

		require.EqualValues(t, c.TimeOperandKind, timeOp2.GetKind())
		require.True(t, refTime.Equal(time.Time(timeOp2.(c.TimeOperand))))
	})
}

// TestTimeOperand_Comparison tests TimeOperand comparison operations
func TestTimeOperand_Comparison(t *testing.T) {
	time1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)

	t.Run("Greater", func(t *testing.T) {
		op1 := c.NewTimeOperand(time1)
		op2 := c.NewTimeOperand(time2)

		require.False(t, op1.Greater(op2))
		require.True(t, op2.Greater(op1))
	})

	t.Run("Equals", func(t *testing.T) {
		op1 := c.NewTimeOperand(time1)
		op2 := c.NewTimeOperand(time1)
		op3 := c.NewTimeOperand(time2)

		require.True(t, op1.Equals(op2))
		require.False(t, op1.Equals(op3))
	})

	t.Run("Equals with different timezones, same instant", func(t *testing.T) {
		utcTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		estLoc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		estTime := utcTime.In(estLoc)

		op1 := c.NewTimeOperand(utcTime)
		op2 := c.NewTimeOperand(estTime)

		// Should be equal because they represent the same instant
		require.True(t, op1.Equals(op2))
	})

	t.Run("Greater with different timezones", func(t *testing.T) {
		utcTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		estLoc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		laterTime := utcTime.Add(1 * time.Hour)
		estTime := laterTime.In(estLoc)

		op1 := c.NewTimeOperand(utcTime)
		op2 := c.NewTimeOperand(estTime)

		require.True(t, op2.Greater(op1))
		require.False(t, op1.Greater(op2))
	})
}

// TestTimeOperand_Hashing tests TimeOperand hash generation
func TestTimeOperand_Hashing(t *testing.T) {
	time1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)

	op1 := c.NewTimeOperand(time1)
	op2 := c.NewTimeOperand(time2)
	op3 := c.NewTimeOperand(time3)

	t.Run("Same times have same hash", func(t *testing.T) {
		require.Equal(t, op1.GetHash(), op2.GetHash())
	})

	t.Run("Different times have different hash", func(t *testing.T) {
		require.NotEqual(t, op1.GetHash(), op3.GetHash())
	})

	t.Run("Same instant in different timezones have same hash", func(t *testing.T) {
		utcTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		estLoc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		estTime := utcTime.In(estLoc)

		opUTC := c.NewTimeOperand(utcTime)
		opEST := c.NewTimeOperand(estTime)

		// Same instant should have same hash
		require.Equal(t, opUTC.GetHash(), opEST.GetHash())
	})
}
