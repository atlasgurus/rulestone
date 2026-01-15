package tests_test

import (
	"fmt"
	"time"

	c "github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOf(t *testing.T) {
	t.Run("immutable_set test", func(t *testing.T) {
		rule := c.NewRule(
			1,
			c.NewAndCond(
				c.NewOrCond(c.NewCategoryCond(9514)),
				c.NewOrCond(c.NewCategoryCond(862)),
				c.NewOrCond(c.NewCategoryCond(9259)),
				c.NewOrCond(c.NewCategoryCond(5834)),
				c.NewOrCond(c.NewCategoryCond(9180)),
				c.NewOrCond(c.NewCategoryCond(5594),
					c.NewCategoryCond(3600),
					c.NewCategoryCond(3435)),
				c.NewOrCond(c.NewCategoryCond(2934)),
				c.NewNotCond(
					c.NewOrCond(c.NewCategoryCond(4869), c.NewCategoryCond(9324))),
			))

		fmt.Println(rule)
	})
}

// TestNewInterfaceOperand_TimeTypes tests time.Time and *time.Time handling in NewInterfaceOperand
func TestNewInterfaceOperand_TimeTypes(t *testing.T) {
	ctx := types.NewAppContext()

	t.Run("time.Time value", func(t *testing.T) {
		now := time.Now()
		op := c.NewInterfaceOperand(now, ctx)

		require.EqualValues(t, c.TimeOperandKind, op.GetKind())
		require.True(t, now.Equal(time.Time(op.(c.TimeOperand))))
	})

	t.Run("*time.Time non-nil pointer", func(t *testing.T) {
		now := time.Now()
		op := c.NewInterfaceOperand(&now, ctx)

		require.EqualValues(t, c.TimeOperandKind, op.GetKind())
		require.True(t, now.Equal(time.Time(op.(c.TimeOperand))))
	})

	t.Run("*time.Time nil pointer", func(t *testing.T) {
		var nilTime *time.Time = nil
		op := c.NewInterfaceOperand(nilTime, ctx)

		require.EqualValues(t, c.NullOperandKind, op.GetKind())
	})

	t.Run("time.Time zero value", func(t *testing.T) {
		var zeroTime time.Time
		op := c.NewInterfaceOperand(zeroTime, ctx)

		require.EqualValues(t, c.TimeOperandKind, op.GetKind())
		require.True(t, time.Time(op.(c.TimeOperand)).IsZero())
	})

	t.Run("time.Time Unix epoch", func(t *testing.T) {
		epoch := time.Unix(0, 0)
		op := c.NewInterfaceOperand(epoch, ctx)

		require.EqualValues(t, c.TimeOperandKind, op.GetKind())
		require.True(t, epoch.Equal(time.Time(op.(c.TimeOperand))))
	})

	t.Run("time.Time with location", func(t *testing.T) {
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		now := time.Now().In(loc)
		op := c.NewInterfaceOperand(now, ctx)

		require.EqualValues(t, c.TimeOperandKind, op.GetKind())
		require.True(t, now.Equal(time.Time(op.(c.TimeOperand))))
	})
}
