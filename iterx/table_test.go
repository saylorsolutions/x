package iterx

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestTableMetadata_FilterColumns(t *testing.T) {
	table := [][]int{
		{0, 1, 2},
	}
	tests := map[string]struct {
		excluded int
		expected [][]int
	}{
		"Exclude head": {
			excluded: 0,
			expected: [][]int{{1, 2}},
		},
		"Exclude tail": {
			excluded: 2,
			expected: [][]int{{0, 1}},
		},
		"Exclude middle": {
			excluded: 1,
			expected: [][]int{{0, 2}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			result := SelectTable(table).FilterColumns(SkipColumns[int](tc.excluded)).Table()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTableMetadata_FilterRows(t *testing.T) {
	table := [][]int{
		{0},
		{1},
		{2},
	}
	tests := map[string]struct {
		excluded int
		expected [][]int
	}{
		"Exclude head row": {
			excluded: 0,
			expected: [][]int{{1}, {2}},
		},
		"Exclude tail row": {
			excluded: 2,
			expected: [][]int{{0}, {1}},
		},
		"Exclude middle row": {
			excluded: 1,
			expected: [][]int{{0}, {2}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			result := SelectTable(table).FilterRows(SkipRows[int](tc.excluded)).Table()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformRows(t *testing.T) {
	table := SelectTable([][]string{
		{"A", "54", "blah"},
		{"B", "45", "blah"},
	})
	type transformedRow struct {
		name string
		age  int
		err  error
	}
	transform := func(row MapIter[int, string]) transformedRow {
		cols := row.Values().Slice()
		age, err := strconv.Atoi(cols[1])
		if err != nil {
			return transformedRow{
				err: err,
			}
		}
		return transformedRow{
			name: cols[0],
			age:  age,
		}
	}
	rows := TransformRows(table.FilterColumns(SkipColumns[string](2)), transform)
	assert.Equal(t, 2, rows.Count())
	assert.Equal(t, []int{0, 1}, rows.Keys().Slice())
	rows.ForEach(func(row int, val transformedRow) bool {
		assert.NoError(t, val.err)
		switch row {
		case 0:
			assert.Equal(t, "A", val.name)
			assert.Equal(t, 54, val.age)
		case 1:
			assert.Equal(t, "B", val.name)
			assert.Equal(t, 45, val.age)
		default:
			t.Error("Too many rows")
		}
		return true
	})
}

func TestTransformLabeledTable(t *testing.T) {
	table := SelectTable([][]string{
		{"A", "54", "blah"},
		{"B", "45", "blah"},
	})
	type transformedRow struct {
		name string
		age  int
		err  error
	}
	transform := func(row MapIter[string, string]) transformedRow {
		cols := row.Map()
		age, err := strconv.Atoi(cols["age"])
		if err != nil {
			return transformedRow{
				err: err,
			}
		}
		return transformedRow{
			name: cols["name"],
			age:  age,
		}
	}
	rows := TransformLabeledRows(table.FilterColumns(SkipColumns[string](2)), []string{"name", "age", "desc"}, transform)
	assert.Equal(t, 2, rows.Count())
	assert.Equal(t, []int{0, 1}, rows.Keys().Slice())
	rows.ForEach(func(row int, val transformedRow) bool {
		assert.NoError(t, val.err)
		switch row {
		case 0:
			assert.Equal(t, "A", val.name)
			assert.Equal(t, 54, val.age)
		case 1:
			assert.Equal(t, "B", val.name)
			assert.Equal(t, 45, val.age)
		default:
			t.Error("Too many rows")
		}
		return true
	})
}

func TestTableIterator_AppendColumn(t *testing.T) {
	table := SelectTable([][]float64{
		{2, 2},
		{3, 3},
		{4, 4},
	})
	result := table.AppendColumn(func(row MapIter[int, float64]) float64 {
		return Sum(row.Values())
	}).AppendColumn(func(row MapIter[int, float64]) float64 {
		return Sum(row.Values())
	}).AppendColumn(func(row MapIter[int, float64]) float64 {
		return Sum(row.Values())
	}).SelectColumns(2, 3, 4).Table()
	expected := [][]float64{
		{4, 8, 16},
		{6, 12, 24},
		{8, 16, 32},
	}
	assert.Equal(t, expected, result)
}

func TestTableIterator_RotateTable(t *testing.T) {
	table := SelectTable([][]string{
		{"A1", "B1"},
		{"A2", "B2"},
	})
	result := TransformSlice(table.RotateTable().Values(), func(val SliceIter[string]) []string {
		return val.Slice()
	}).Slice()
	expected := [][]string{
		{"A1", "A2"},
		{"B1", "B2"},
	}
	assert.Equal(t, expected, result)
}

func TestSelectTable_PanicOnDifferentRowWidths(t *testing.T) {
	assert.Panics(t, func() {
		SelectTable([][]int{
			{0, 1, 2},
			{0, 1},
			{0, 1, 2},
		})
	})
	assert.NotPanics(t, func() {
		width, height := SelectTable([][]int{
			{0, 1},
			{0, 1},
			{0, 1},
		}).Dimensions()
		assert.Equal(t, 2, width)
		assert.Equal(t, 3, height)
	})
}
