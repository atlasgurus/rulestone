package tests_test

import (
	"fmt"
	c "github.com/atlasgurus/rulestone/condition"
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
