package somename

import (
	"testing"

	"git.front.kjuulh.io/kjuulh/scaffold/internal/tests"
)

func TestScaffold(t *testing.T) {
	tests.
		Test(t, "somename").
		ScaffoldDefaultTest("default").
		ScaffoldTest("scaffold package with name",
			func(fixture *tests.ScaffoldFixture) {
				fixture.WithVariable("package", "somename")
			},
		)
}
