package di

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDag(t *testing.T) {
	Convey("Create a dag", t, func(){
		dag := NewDAG()
		So(dag, ShouldNotBeNil)

		// Assert that only unique vertices can be created
		So(dag.NewVertex("shirt", nil), ShouldBeNil)
		So(dag.NewVertex("tie", nil), ShouldBeNil)
		So(dag.NewVertex("belt", nil), ShouldBeNil)
		So(dag.NewVertex("pants", nil), ShouldBeNil)
		So(dag.NewVertex("jacket", nil), ShouldBeNil)
		So(dag.NewVertex("shirt", nil), ShouldBeError)

		// Assert that edges can be created to only existing vertices
		So(dag.Edge("tie", "shirt"), ShouldBeNil)
		So(dag.Edge("jacket", "tie", "belt"), ShouldBeNil)
		So(dag.Edge("belt", "pants"), ShouldBeNil)
		So(dag.Edge("shirt", "does_not_exist"), ShouldBeError)
		So(dag.Edge("does_not_exist", "shirt"), ShouldBeError)

		// Assert that cycles cannot happen
		So(dag.Edge("shirt", "shirt"), ShouldBeError)
		So(dag.Edge("shirt", "jacket"), ShouldBeError)

		expectedSortedNodes := []Vertex{
			{"pants", nil},
			{"belt", nil},
			{"shirt", nil},
			{"tie", nil},
			{"jacket", nil},
		}
		sortedNodes := dag.Sort()
		So(sortedNodes, ShouldResemble, expectedSortedNodes)
	})
}
