package mineralmodel

import (
	"reflect"
	"testing"

	"geoworld/internal/world"
)

func testGeometry() world.GridGeometry {
	return world.GridGeometry{
		Width:       320,
		Height:      200,
		OriginX:     -20416,
		OriginZ:     -12736,
		SpanXBlocks: 40832,
		SpanZBlocks: 25472,
	}
}

func TestBuildResolvesUnavailablePrimaryThenReplansCompanion(t *testing.T) {
	dictionary := []string{"copper", "iron", "tin", "zinc"}
	available := []string{"copper", "iron"}
	model := Build(123456, testGeometry(), dictionary, available)

	foundUnavailablePrimary := false
	for _, assignment := range model.Assignments {
		if contains(available, assignment.PlannedPrimary) {
			continue
		}
		foundUnavailablePrimary = true
		if !contains(available, assignment.Primary) {
			t.Fatalf("resolved primary %q is not available: %+v", assignment.Primary, assignment)
		}
		if assignment.Primary == assignment.Companion {
			t.Fatalf("primary and companion must remain distinct: %+v", assignment)
		}

		replanned := pickCompanion(
			model.Dictionary,
			assignment.Primary,
			uint64(assignment.ID)^hashString(assignment.Primary)^companionSalt,
		)
		expected := replanned
		if expected == assignment.Primary || !contains(model.Available, expected) {
			expected = pickAvailableCompanion(
				model.Available,
				assignment.Primary,
				uint64(assignment.ID)^hashString(assignment.Primary)^replaceSalt^companionSalt,
			)
		}
		if assignment.Companion != expected {
			t.Fatalf(
				"companion was not replanned from replacement primary: got=%q want=%q assignment=%+v",
				assignment.Companion,
				expected,
				assignment,
			)
		}
	}
	if !foundUnavailablePrimary {
		t.Fatal("test seed did not produce an unavailable planned primary")
	}
}

func TestPersistedAssignmentsDoNotChangeWhenFutureMineralsAppear(t *testing.T) {
	dictionary := []string{"copper", "iron", "tin", "zinc"}
	available := []string{"copper", "iron"}
	original := Build(123456, testGeometry(), dictionary, available)

	frozen := append([]DistrictAssignment(nil), original.Assignments...)

	// A later mod installation changes what a newly generated model could use,
	// but must not mutate assignments already stored in the old World Model.
	_ = Build(
		123456,
		testGeometry(),
		append(dictionary, "uranium"),
		append(available, "zinc", "uranium"),
	)

	if !reflect.DeepEqual(frozen, original.Assignments) {
		t.Fatal("existing model assignments changed after future mineral discovery")
	}
}

func TestResolvedAssignmentsUseOnlyGenerationTimeAvailableFamilies(t *testing.T) {
	model := Build(
		-4852173505788415804,
		testGeometry(),
		[]string{"copper", "iron", "tin", "zinc", "uranium"},
		[]string{"iron", "zinc"},
	)
	for _, assignment := range model.Assignments {
		if !contains(model.Available, assignment.Primary) {
			t.Fatalf("primary %q is not in generation-time available set", assignment.Primary)
		}
		if !contains(model.Available, assignment.Companion) {
			t.Fatalf("companion %q is not in generation-time available set", assignment.Companion)
		}
		if assignment.Primary == assignment.Companion {
			t.Fatalf("primary and companion must differ: %+v", assignment)
		}
	}
}
